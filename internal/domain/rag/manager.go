package rag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	config_types "xiaozhi-esp32-server-golang/internal/domain/config/types"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/spf13/viper"
)

const (
	defaultKnowledgeProvider            = "dify"
	defaultKnowledgeTopK                = 5
	defaultKnowledgeSearchSingleTimeout = 2500 * time.Millisecond
	defaultKnowledgeSearchTotalTimeout  = 2500 * time.Millisecond
	defaultKnowledgeSearchMaxParallel   = 8
)

// Search groups and sorts by knowledge base provider.
func Search(
	ctx context.Context,
	query string,
	topK int,
	knowledgeBases []config_types.KnowledgeBaseRef,
	knowledgeBaseIDs []uint,
) ([]config_types.KnowledgeSearchHit, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	if topK <= 0 {
		topK = defaultKnowledgeTopK
	}
	if len(knowledgeBases) == 0 {
		return []config_types.KnowledgeSearchHit{}, nil
	}

	selectedKBSet := make(map[uint]struct{}, len(knowledgeBaseIDs))
	for _, kbID := range knowledgeBaseIDs {
		if kbID == 0 {
			continue
		}
		selectedKBSet[kbID] = struct{}{}
	}

	defaultProvider := strings.TrimSpace(viper.GetString("knowledge.default_provider"))
	grouped := make(map[string][]config_types.KnowledgeBaseRef)
	for _, kb := range knowledgeBases {
		if strings.EqualFold(strings.TrimSpace(kb.Status), "inactive") {
			continue
		}
		if strings.TrimSpace(kb.ExternalKBID) == "" {
			continue
		}
		if len(selectedKBSet) > 0 {
			if _, ok := selectedKBSet[kb.ID]; !ok {
				continue
			}
		}
		provider := strings.ToLower(strings.TrimSpace(kb.Provider))
		if provider == "" {
			provider = strings.ToLower(defaultProvider)
		}
		if provider == "" {
			provider = defaultKnowledgeProvider
		}
		kb.Provider = provider
		grouped[provider] = append(grouped[provider], kb)
	}
	if len(grouped) == 0 {
		return []config_types.KnowledgeSearchHit{}, nil
	}

	totalCtx := ctx
	cancel := func() {}
	if timeout := getKnowledgeSearchTotalTimeout(); timeout > 0 {
		totalCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	hits := make([]config_types.KnowledgeSearchHit, 0, topK)
	errs := make([]string, 0)
	successProviderCount := 0
	for provider, providerKBs := range grouped {
		searcher := getSearcher(provider)
		if searcher == nil {
			errs = append(errs, fmt.Sprintf("provider %s is not supported", provider))
			continue
		}
		providerConfig, ok := getProviderConfig(provider)
		if !ok {
			errs = append(errs, fmt.Sprintf("provider %s is missing global configuration", provider))
			continue
		}

		providerHits, err := searcher.Search(totalCtx, q, topK, providerKBs, providerConfig)
		if err != nil {
			errs = append(errs, fmt.Sprintf("provider %s retrieval failed: %v", provider, err))
			continue
		}
		successProviderCount++
		hits = append(hits, providerHits...)
	}

	if len(hits) == 0 {
		if successProviderCount == 0 && len(errs) > 0 {
			return nil, errors.New(strings.Join(errs, "; "))
		}
		return []config_types.KnowledgeSearchHit{}, nil
	}

	sort.SliceStable(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > topK {
		hits = hits[:topK]
	}

	if len(errs) > 0 {
		log.Warnf("Failed to retrieve some providers in the knowledge base: %s", strings.Join(errs, "; "))
	}
	return hits, nil
}

func getSearcher(provider string) Searcher {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "dify":
		return &difySearcher{}
	case "ragflow":
		return &ragflowSearcher{}
	case "weknora":
		return &weknoraSearcher{}
	default:
		return nil
	}
}

func getProviderConfig(provider string) (map[string]interface{}, bool) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return nil, false
	}
	all := viper.GetStringMap("knowledge.providers")
	if len(all) == 0 {
		return nil, false
	}

	for key, raw := range all {
		if strings.ToLower(strings.TrimSpace(key)) != provider {
			continue
		}
		cfg, ok := normalizeMap(raw)
		if !ok {
			return map[string]interface{}{}, true
		}
		return cfg, true
	}
	return nil, false
}

func normalizeMap(input interface{}) (map[string]interface{}, bool) {
	if input == nil {
		return nil, false
	}
	if m, ok := input.(map[string]interface{}); ok {
		return m, true
	}
	b, err := json.Marshal(input)
	if err != nil {
		return nil, false
	}
	var ret map[string]interface{}
	if err := json.Unmarshal(b, &ret); err != nil {
		return nil, false
	}
	return ret, true
}

func getKnowledgeSearchSingleTimeout() time.Duration {
	return getKnowledgeSearchDuration("knowledge.search.single_timeout_ms", defaultKnowledgeSearchSingleTimeout)
}

func getKnowledgeSearchTotalTimeout() time.Duration {
	return getKnowledgeSearchDuration("knowledge.search.total_timeout_ms", defaultKnowledgeSearchTotalTimeout)
}

func getKnowledgeSearchMaxParallel() int {
	raw := viper.Get("knowledge.search.max_parallel")
	if raw == nil {
		return defaultKnowledgeSearchMaxParallel
	}
	switch v := raw.(type) {
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 {
			return int(v)
		}
	case float64:
		if v > 0 {
			return int(v)
		}
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && i > 0 {
			return i
		}
	}
	return defaultKnowledgeSearchMaxParallel
}

func getKnowledgeSearchDuration(configKey string, defaultValue time.Duration) time.Duration {
	raw := viper.Get(configKey)
	if raw == nil {
		return defaultValue
	}
	switch v := raw.(type) {
	case int:
		if v > 0 {
			return time.Duration(v) * time.Millisecond
		}
	case int64:
		if v > 0 {
			return time.Duration(v) * time.Millisecond
		}
	case float64:
		if v > 0 {
			return time.Duration(v) * time.Millisecond
		}
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return defaultValue
		}
		if i, err := strconv.Atoi(s); err == nil && i > 0 {
			return time.Duration(i) * time.Millisecond
		}
	}
	return defaultValue
}
