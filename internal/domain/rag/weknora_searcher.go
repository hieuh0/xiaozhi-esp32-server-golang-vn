package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	config_types "xiaozhi-esp32-server-golang/internal/domain/config/types"
	log "xiaozhi-esp32-server-golang/logger"
)

type weknoraSearcher struct{}

func (s *weknoraSearcher) Search(
	ctx context.Context,
	query string,
	topK int,
	knowledgeBases []config_types.KnowledgeBaseRef,
	providerConfig map[string]interface{},
) ([]config_types.KnowledgeSearchHit, error) {
	baseURL, _ := providerConfig["base_url"].(string)
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("weknora base_url cannot be empty")
	}

	apiKey, _ := providerConfig["api_key"].(string)
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("weknora api_key cannot be empty")
	}

	globalScoreThreshold := 0.2
	if raw, ok := providerConfig["score_threshold"]; ok {
		globalScoreThreshold = parseFloat(raw)
	}
	if globalScoreThreshold < 0 {
		globalScoreThreshold = 0
	}
	if globalScoreThreshold > 1 {
		globalScoreThreshold = 1
	}

	endpoint := buildWeknoraRetrieveURL(baseURL, "/knowledge-search")
	client := &http.Client{}
	maxParallel := getKnowledgeSearchMaxParallel()
	if maxParallel <= 0 {
		maxParallel = 1
	}
	sem := make(chan struct{}, maxParallel)
	perKBTimeout := getKnowledgeSearchSingleTimeout()

	type searchResult struct {
		hits []config_types.KnowledgeSearchHit
		err  error
	}
	resultCh := make(chan searchResult, len(knowledgeBases))

	var wg sync.WaitGroup
	launchCount := 0
	for _, kb := range knowledgeBases {
		kb := kb
		datasetID := strings.TrimSpace(kb.ExternalKBID)
		if datasetID == "" {
			continue
		}
		launchCount++
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-ctx.Done():
				resultCh <- searchResult{err: ctx.Err()}
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			reqCtx := ctx
			cancel := func() {}
			if perKBTimeout > 0 {
				reqCtx, cancel = context.WithTimeout(ctx, perKBTimeout)
			}
			defer cancel()

			hits, err := s.searchOneKnowledgeBase(reqCtx, client, endpoint, apiKey, strings.TrimSpace(query), kb, globalScoreThreshold)
			resultCh <- searchResult{hits: hits, err: err}
		}()
	}
	wg.Wait()
	close(resultCh)

	if launchCount == 0 {
		return []config_types.KnowledgeSearchHit{}, nil
	}

	ret := make([]config_types.KnowledgeSearchHit, 0, topK)
	errs := make([]string, 0)
	for result := range resultCh {
		if result.err != nil {
			errs = append(errs, result.err.Error())
			continue
		}
		ret = append(ret, result.hits...)
	}
	if len(ret) == 0 && len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "; "))
	}
	if len(errs) > 0 {
		log.Warnf("Weknora knowledge base search partially failed: %s", strings.Join(errs, "; "))
	}
	return ret, nil
}

func (s *weknoraSearcher) searchOneKnowledgeBase(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	apiKey string,
	query string,
	kb config_types.KnowledgeBaseRef,
	globalScoreThreshold float64,
) ([]config_types.KnowledgeSearchHit, error) {
	datasetID := strings.TrimSpace(kb.ExternalKBID)
	if datasetID == "" {
		return []config_types.KnowledgeSearchHit{}, nil
	}

	// WeKnora score scales may be far below the generic 0~1 thresholds used by
	// other providers, so do not apply local score filtering here.
	_ = globalScoreThreshold
	if kb.RetrievalThreshold != nil {
		_ = *kb.RetrievalThreshold
	}

	payload := map[string]interface{}{
		"query":              query,
		"knowledge_base_ids": []string{datasetID},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Failed to create Weknora request (knowledge_base_id=%s): %w", datasetID, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to call Weknora (knowledge_base_id=%s): %w", datasetID, err)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Weknora returns exception (knowledge_base_id=%s): %d %s", datasetID, resp.StatusCode, string(bodyBytes))
	}

	var weknoraResp struct {
		Success *bool       `json:"success"`
		Code    interface{} `json:"code"`
		Message string      `json:"message"`
		Msg     string      `json:"msg"`
		Data    []struct {
			Content        string                 `json:"content"`
			KnowledgeTitle string                 `json:"knowledge_title"`
			Score          float64                `json:"score"`
			Similarity     float64                `json:"similarity"`
			Metadata       map[string]interface{} `json:"metadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &weknoraResp); err != nil {
		return nil, fmt.Errorf("Failed to parse Weknora (knowledge_base_id=%s): %w", datasetID, err)
	}
	if weknoraResp.Success != nil && !*weknoraResp.Success {
		msg := strings.TrimSpace(weknoraResp.Message)
		if msg == "" {
			msg = strings.TrimSpace(weknoraResp.Msg)
		}
		return nil, fmt.Errorf("Weknora request failed (knowledge_base_id=%s): %s", datasetID, msg)
	}
	if code, ok := parseCodeInt(weknoraResp.Code); ok && code != 0 {
		msg := strings.TrimSpace(weknoraResp.Message)
		if msg == "" {
			msg = strings.TrimSpace(weknoraResp.Msg)
		}
		return nil, fmt.Errorf("Weknora request failed (knowledge_base_id=%s): code=%d message=%s", datasetID, code, msg)
	}

	title := strings.TrimSpace(kb.Name)
	if title == "" {
		title = datasetID
	}
	ret := make([]config_types.KnowledgeSearchHit, 0, len(weknoraResp.Data))
	for _, item := range weknoraResp.Data {
		content := strings.TrimSpace(item.Content)
		if content == "" && item.Metadata != nil {
			if chunkText, ok := item.Metadata["chunk_text"]; ok {
				content = strings.TrimSpace(fmt.Sprintf("%v", chunkText))
			}
		}
		if content == "" || content == "<nil>" {
			continue
		}

		score := item.Score
		if score <= 0 {
			score = item.Similarity
		}
		if score <= 0 && item.Metadata != nil {
			score = parseFloat(item.Metadata["score"])
		}
		chunkTitle := strings.TrimSpace(item.KnowledgeTitle)
		if chunkTitle == "" {
			chunkTitle = title
		}
		ret = append(ret, config_types.KnowledgeSearchHit{
			Content: content,
			Title:   chunkTitle,
			Score:   score,
		})
	}
	return ret, nil
}

func buildWeknoraRetrieveURL(baseURL, path string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	lower := strings.ToLower(trimmed)
	if strings.HasSuffix(lower, "/api/v1") {
		return trimmed + path
	}
	if strings.HasSuffix(lower, "/api") {
		return trimmed + "/v1" + path
	}
	return trimmed + "/api/v1" + path
}

func parseCodeInt(input interface{}) (int, bool) {
	switch v := input.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i), true
		}
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, false
		}
		if i, err := strconv.Atoi(s); err == nil {
			return i, true
		}
	}
	return 0, false
}
