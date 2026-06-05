package controllers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"xiaozhi/manager/backend/models"
	mcpmarket "xiaozhi/manager/backend/services/mcp_market"

	"gorm.io/gorm"
)

type mcpServiceNameState struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled"`
}

func splitMCPServiceNames(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}

	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func normalizeMCPServiceNamesCSV(raw string) string {
	return strings.Join(splitMCPServiceNames(raw), ",")
}

func buildMCPServiceNameSet(names []string) map[string]struct{} {
	result := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		result[name] = struct{}{}
	}
	return result
}

func validateMCPServiceNamesCSV(raw string, allowed map[string]struct{}) (string, error) {
	normalized := normalizeMCPServiceNamesCSV(raw)
	if normalized == "" {
		return "", nil
	}

	selected := splitMCPServiceNames(normalized)
	invalid := make([]string, 0)
	for _, name := range selected {
		if _, ok := allowed[name]; !ok {
			invalid = append(invalid, name)
		}
	}
	if len(invalid) > 0 {
		return "", fmt.Errorf("contains disabled or non-existent MCP services: %s", strings.Join(invalid, ","))
	}
	return normalized, nil
}

func listEnabledGlobalMCPServiceNames(db *gorm.DB) ([]string, error) {
	manualMCP, err := loadManualMCPMapForAgentSettings(db)
	if err != nil {
		return nil, err
	}

	merged, err := mergeManualAndEnabledMarketMCPMap(db, manualMCP)
	if err != nil {
		return nil, err
	}

	return collectEnabledGlobalMCPServiceNamesFromMap(merged)
}

func loadManualMCPMapForAgentSettings(db *gorm.DB) (map[string]interface{}, error) {
	var configs []models.Config
	if err := db.Where("type = ?", mcpmarket.MCPConfigType).Order("is_default DESC, id ASC").Find(&configs).Error; err != nil {
		return nil, err
	}

	mcpMap := defaultMCPMap()
	if len(configs) > 0 {
		payload, err := parseJSONMap(configs[0].JsonData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MCP config: %w", err)
		}
		if v, ok := payload["mcp"]; ok {
			if mv := asMap(v); mv != nil {
				mcpMap = mv
			}
		} else if v, ok := payload["global"]; ok {
			mcpMap = map[string]interface{}{"global": v}
		} else if len(payload) > 0 {
			mcpMap = payload
		}
	}

	ensureMCPGlobalDefaults(mcpMap)
	return mcpMap, nil
}

func mergeManualAndEnabledMarketMCPMap(db *gorm.DB, manualMCP map[string]interface{}) (map[string]interface{}, error) {
	var marketServices []models.MCPMarketService
	if err := db.Where("enabled = ?", true).Order("id ASC").Find(&marketServices).Error; err != nil {
		return nil, err
	}

	merged, _, err := mergeManualAndMarketServers(manualMCP, marketServices)
	if err != nil {
		return nil, err
	}
	return merged, nil
}

func decodeMCPServiceNameStates(v interface{}) ([]mcpServiceNameState, error) {
	if v == nil {
		return []mcpServiceNameState{}, nil
	}
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var states []mcpServiceNameState
	if err := json.Unmarshal(jsonBytes, &states); err != nil {
		return nil, err
	}
	return states, nil
}

func collectEnabledGlobalMCPServiceNamesFromMap(mcpMap map[string]interface{}) ([]string, error) {
	if mcpMap == nil {
		return []string{}, nil
	}
	ensureMCPGlobalDefaults(mcpMap)
	global := asMap(mcpMap["global"])
	if global == nil {
		return []string{}, nil
	}

	if enabled, ok := global["enabled"].(bool); ok && !enabled {
		return []string{}, nil
	}

	states, err := decodeMCPServiceNameStates(global["servers"])
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(states))
	seen := make(map[string]struct{})
	for _, state := range states {
		name := strings.TrimSpace(state.Name)
		if name == "" {
			continue
		}
		if state.Enabled != nil && !*state.Enabled {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func (uc *UserController) normalizeAndValidateAgentMCPServices(raw string) (string, error) {
	options, err := listEnabledGlobalMCPServiceNames(uc.DB)
	if err != nil {
		return "", err
	}
	allowed := buildMCPServiceNameSet(options)
	return validateMCPServiceNamesCSV(raw, allowed)
}

func (ac *AdminController) normalizeAndValidateAgentMCPServices(raw string) (string, error) {
	options, err := listEnabledGlobalMCPServiceNames(ac.DB)
	if err != nil {
		return "", err
	}
	allowed := buildMCPServiceNameSet(options)
	return validateMCPServiceNamesCSV(raw, allowed)
}
