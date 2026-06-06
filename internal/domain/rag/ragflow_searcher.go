package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	config_types "xiaozhi-esp32-server-golang/internal/domain/config/types"
	log "xiaozhi-esp32-server-golang/logger"
)

type ragflowSearcher struct{}

func (s *ragflowSearcher) Search(
	ctx context.Context,
	query string,
	topK int,
	knowledgeBases []config_types.KnowledgeBaseRef,
	providerConfig map[string]interface{},
) ([]config_types.KnowledgeSearchHit, error) {
	baseURL, _ := providerConfig["base_url"].(string)
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("ragflow base_url cannot be empty")
	}

	apiKey, _ := providerConfig["api_key"].(string)
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("ragflow api_key cannot be empty")
	}

	globalSimilarityThreshold := 0.2
	if raw, ok := providerConfig["similarity_threshold"]; ok {
		globalSimilarityThreshold = parseFloat(raw)
	}
	if globalSimilarityThreshold < 0 {
		globalSimilarityThreshold = 0
	}
	if globalSimilarityThreshold > 1 {
		globalSimilarityThreshold = 1
	}
	vectorSimilarityWeight := parseFloat(providerConfig["vector_similarity_weight"])
	if vectorSimilarityWeight <= 0 {
		vectorSimilarityWeight = 0.3
	}
	keyword := parseBool(providerConfig["keyword"])
	highlight := parseBool(providerConfig["highlight"])
	endpoint := buildRagflowRetrieveURL(baseURL, "/retrieval")
	client := &http.Client{}
	seen := make(map[string]struct{}, len(knowledgeBases))
	filteredKBs := make([]config_types.KnowledgeBaseRef, 0, len(knowledgeBases))
	for _, kb := range knowledgeBases {
		datasetID := strings.TrimSpace(kb.ExternalKBID)
		if datasetID == "" {
			continue
		}
		if _, ok := seen[datasetID]; ok {
			continue
		}
		seen[datasetID] = struct{}{}
		filteredKBs = append(filteredKBs, kb)
	}
	if len(filteredKBs) == 0 {
		return []config_types.KnowledgeSearchHit{}, nil
	}

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
	resultCh := make(chan searchResult, len(filteredKBs))

	var wg sync.WaitGroup
	for _, kb := range filteredKBs {
		kb := kb
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

			hits, err := s.searchOneDataset(reqCtx, client, endpoint, apiKey, strings.TrimSpace(query), topK, kb, globalSimilarityThreshold, vectorSimilarityWeight, keyword, highlight)
			resultCh <- searchResult{hits: hits, err: err}
		}()
	}
	wg.Wait()
	close(resultCh)

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
		log.Warnf("RAGFlow knowledge base search part failed: %s", strings.Join(errs, "; "))
	}
	return ret, nil
}

func (s *ragflowSearcher) searchOneDataset(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	apiKey string,
	query string,
	topK int,
	kb config_types.KnowledgeBaseRef,
	globalSimilarityThreshold float64,
	vectorSimilarityWeight float64,
	keyword bool,
	highlight bool,
) ([]config_types.KnowledgeSearchHit, error) {
	datasetID := strings.TrimSpace(kb.ExternalKBID)
	if datasetID == "" {
		return []config_types.KnowledgeSearchHit{}, nil
	}

	similarityThreshold := globalSimilarityThreshold
	if kb.RetrievalThreshold != nil {
		similarityThreshold = *kb.RetrievalThreshold
		if similarityThreshold < 0 {
			similarityThreshold = 0
		}
		if similarityThreshold > 1 {
			similarityThreshold = 1
		}
	}

	payload := map[string]interface{}{
		"question":                 query,
		"dataset_ids":              []string{datasetID},
		"top_k":                    topK,
		"page":                     1,
		"page_size":                topK,
		"similarity_threshold":     similarityThreshold,
		"vector_similarity_weight": vectorSimilarityWeight,
		"keyword":                  keyword,
		"highlight":                highlight,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Failed to create RAGFlow request (dataset_id=%s): %w", datasetID, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to call RAGFlow (dataset_id=%s): %w", datasetID, err)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("RAGFlow returns exception (dataset_id=%s): %d %s", datasetID, resp.StatusCode, string(bodyBytes))
	}

	var ragflowResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Chunks []struct {
				Content          string  `json:"content"`
				Highlight        string  `json:"highlight"`
				Similarity       float64 `json:"similarity"`
				VectorSimilarity float64 `json:"vector_similarity"`
				KBID             string  `json:"kb_id"`
				DocumentName     string  `json:"document_name"`
			} `json:"chunks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &ragflowResp); err != nil {
		return nil, fmt.Errorf("Failed to parse RAGFlow (dataset_id=%s): %w", datasetID, err)
	}
	if ragflowResp.Code != 0 {
		return nil, fmt.Errorf("RAGFlow request failed (dataset_id=%s): code=%d message=%s", datasetID, ragflowResp.Code, strings.TrimSpace(ragflowResp.Message))
	}

	title := strings.TrimSpace(kb.Name)
	if title == "" {
		title = datasetID
	}
	ret := make([]config_types.KnowledgeSearchHit, 0, len(ragflowResp.Data.Chunks))
	for _, chunk := range ragflowResp.Data.Chunks {
		content := strings.TrimSpace(chunk.Content)
		if h := strings.TrimSpace(chunk.Highlight); h != "" {
			content = h
		}
		if content == "" {
			continue
		}
		chunkTitle := title
		if chunkTitle == "" {
			chunkTitle = strings.TrimSpace(chunk.DocumentName)
		}
		score := chunk.Similarity
		if score <= 0 {
			score = chunk.VectorSimilarity
		}
		ret = append(ret, config_types.KnowledgeSearchHit{
			Content: content,
			Title:   chunkTitle,
			Score:   score,
		})
	}
	return ret, nil
}

func buildRagflowRetrieveURL(baseURL, path string) string {
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

func parseBool(input interface{}) bool {
	switch v := input.(type) {
	case bool:
		return v
	case string:
		v = strings.ToLower(strings.TrimSpace(v))
		return v == "1" || v == "true" || v == "yes" || v == "on"
	case float64:
		return v != 0
	case int:
		return v != 0
	}
	return false
}
