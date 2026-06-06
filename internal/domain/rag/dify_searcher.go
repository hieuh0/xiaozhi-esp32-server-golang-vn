package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	config_types "xiaozhi-esp32-server-golang/internal/domain/config/types"
	log "xiaozhi-esp32-server-golang/logger"
)

type difySearcher struct{}

func (s *difySearcher) Search(
	ctx context.Context,
	query string,
	topK int,
	knowledgeBases []config_types.KnowledgeBaseRef,
	providerConfig map[string]interface{},
) ([]config_types.KnowledgeSearchHit, error) {
	baseURL, _ := providerConfig["base_url"].(string)
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("dify base_url cannot be empty")
	}

	apiKey, _ := providerConfig["api_key"].(string)
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("dify api_key cannot be empty")
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

			hits, err := s.searchOneDataset(reqCtx, client, baseURL, apiKey, strings.TrimSpace(query), topK, kb, globalScoreThreshold)
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
		log.Warnf("Dify knowledge base search part failed: %s", strings.Join(errs, "; "))
	}
	return ret, nil
}

func (s *difySearcher) searchOneDataset(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	apiKey string,
	query string,
	topK int,
	kb config_types.KnowledgeBaseRef,
	globalScoreThreshold float64,
) ([]config_types.KnowledgeSearchHit, error) {
	datasetID := strings.TrimSpace(kb.ExternalKBID)
	if datasetID == "" {
		return []config_types.KnowledgeSearchHit{}, nil
	}

	scoreThreshold := globalScoreThreshold
	if kb.RetrievalThreshold != nil {
		scoreThreshold = *kb.RetrievalThreshold
		if scoreThreshold < 0 {
			scoreThreshold = 0
		}
		if scoreThreshold > 1 {
			scoreThreshold = 1
		}
	}

	retrieveURL := buildDifyURL(baseURL, fmt.Sprintf("/datasets/%s/retrieve", url.PathEscape(datasetID)))
	payload := map[string]interface{}{
		"query": query,
		"retrieval_model": map[string]interface{}{
			"top_k":                   topK,
			"score_threshold":         scoreThreshold,
			"score_threshold_enabled": scoreThreshold > 0,
			"search_method":           "semantic_search",
			"reranking_enable":        false,
		},
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, retrieveURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Failed to create Dify request (dataset_id=%s): %w", datasetID, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Failed to call Dify (dataset_id=%s): %w", datasetID, err)
	}
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Dify returns exception (dataset_id=%s): %d %s", datasetID, resp.StatusCode, string(bodyBytes))
	}

	var difyResp struct {
		Records []struct {
			Score   float64 `json:"score"`
			Segment struct {
				Content string `json:"content"`
			} `json:"segment"`
		} `json:"records"`
		Data struct {
			Records []struct {
				Score   float64 `json:"score"`
				Segment struct {
					Content string `json:"content"`
				} `json:"segment"`
			} `json:"records"`
		} `json:"data"`
	}
	decodeErr := json.NewDecoder(resp.Body).Decode(&difyResp)
	resp.Body.Close()
	if decodeErr != nil {
		return nil, fmt.Errorf("Parsing Dify returns failure (dataset_id=%s): %w", datasetID, decodeErr)
	}

	records := difyResp.Records
	if len(records) == 0 {
		records = difyResp.Data.Records
	}
	title := strings.TrimSpace(kb.Name)
	if title == "" {
		title = datasetID
	}
	ret := make([]config_types.KnowledgeSearchHit, 0, len(records))
	for _, record := range records {
		content := strings.TrimSpace(record.Segment.Content)
		if content == "" {
			continue
		}
		ret = append(ret, config_types.KnowledgeSearchHit{
			Content: content,
			Title:   title,
			Score:   record.Score,
		})
	}
	return ret, nil
}

func buildDifyURL(baseURL, path string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(strings.ToLower(trimmed), "/v1") {
		return trimmed + path
	}
	return trimmed + "/v1" + path
}

func parseFloat(input interface{}) float64 {
	switch v := input.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return f
		}
	}
	return 0
}
