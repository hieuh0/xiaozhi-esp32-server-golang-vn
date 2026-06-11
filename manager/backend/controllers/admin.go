package controllers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"xiaozhi/manager/backend/models"
	"xiaozhi/manager/backend/services/configprovider"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// getMapKeys returns the keys of a map
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func normalizeAgentMemoryMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "none":
		return "none"
	case "long":
		return "long"
	default:
		return "short"
	}
}

func normalizeAgentSpeakerChatMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "identified_only":
		return "identified_only"
	default:
		return "off"
	}
}

func findActiveCloneForVoiceModelOverride(base *gorm.DB, provider, ttsConfigID, voiceID string, clone *models.VoiceClone) error {
	query := base.Where(
		"voice_clones.tts_config_id = ? AND voice_clones.provider_voice_id = ? AND voice_clones.status = ?",
		ttsConfigID,
		voiceID,
		voiceCloneStatusActive,
	)
	if provider == "doubao" {
		query = query.Where("voice_clones.provider IN ?", []string{"doubao", "doubao_ws"})
	} else {
		query = query.Where("voice_clones.provider = ?", provider)
	}
	result := query.
		Order("voice_clones.updated_at DESC, voice_clones.created_at DESC").
		Order("voice_clones.id").
		Limit(1).
		Find(clone)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func getAgentAssistantName(agent models.Agent) string {
	if nickname := strings.TrimSpace(agent.Nickname); nickname != "" {
		return nickname
	}
	return strings.TrimSpace(agent.Name)
}

func ensureAgentNickname(agent *models.Agent) {
	if agent == nil {
		return
	}
	agent.Name = strings.TrimSpace(agent.Name)
	agent.Nickname = strings.TrimSpace(agent.Nickname)
	if agent.Nickname == "" {
		agent.Nickname = agent.Name
	}
}

type AdminController struct {
	DB                  *gorm.DB
	WebSocketController *WebSocketController
	InternalAuthToken   string
	EndpointAuthToken   string
}

var errDatabaseUnavailable = errors.New("database connection is unavailable")

// GetDeviceConfigs retrieves config associated with a device by device ID.
// Falls back to global default config if the device is not found.
func (ac *AdminController) GetDeviceConfigs(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id parameter is required"})
		return
	}

	// build config response
	type SpeakerGroupInfo struct {
		ID                 uint     `json:"id"`
		Name               string   `json:"name"`
		Prompt             string   `json:"prompt"`
		Description        string   `json:"description"`
		Uuids              []string `json:"uuids"`
		TTSConfigID        *string  `json:"tts_config_id"`
		Voice              *string  `json:"voice"`
		VoiceModelOverride *string  `json:"voice_model_override,omitempty"`
	}

	type KnowledgeBaseInfo struct {
		ID                 uint     `json:"id"`
		Name               string   `json:"name"`
		Description        string   `json:"description"`
		Provider           string   `json:"provider"`
		ExternalKBID       string   `json:"external_kb_id"`
		ExternalDocID      string   `json:"external_doc_id"`
		RetrievalThreshold *float64 `json:"retrieval_threshold"`
		Status             string   `json:"status"`
	}

	type ConfigResponse struct {
		VAD             models.Config               `json:"vad"`
		ASR             models.Config               `json:"asr"`
		LLM             models.Config               `json:"llm"`
		TTS             models.Config               `json:"tts"`
		Memory          models.Config               `json:"memory"`
		VoiceIdentify   map[string]SpeakerGroupInfo `json:"voice_identify"`
		KnowledgeBases  []KnowledgeBaseInfo         `json:"knowledge_bases"`
		Prompt          string                      `json:"prompt"`
		AgentID         string                      `json:"agent_id"`
		MemoryMode      string                      `json:"memory_mode"`
		SpeakerChatMode string                      `json:"speaker_chat_mode"`
		MCPServiceNames string                      `json:"mcp_service_names"`
		OpenClaw        OpenClawConfigResponse      `json:"openclaw"`
		ConfigSource    string                      `json:"config_source"` // config source
	}

	var response ConfigResponse
	response.MemoryMode = "short"
	response.SpeakerChatMode = "off"
	response.OpenClaw = OpenClawConfigResponse{
		Allowed:       false,
		EnterKeywords: []string{},
		ExitKeywords:  []string{},
	}
	var configSource string // tracks the config source

	// find device
	var device models.Device
	var agent models.Agent
	var deviceFound bool

	if err := ac.DB.Where("device_name = ?", deviceID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// device not found, use global default config
			deviceFound = false
			response.AgentID = ""
			configSource = "default_global_role"
			log.Printf("device %s not found, using global default config", deviceID)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query device"})
			return
		}
	} else {
		// device found, look up agent
		deviceFound = true
		response.AgentID = fmt.Sprintf("%d", device.AgentID)
		log.Printf("device %s found, AgentID: %d", deviceID, device.AgentID)
		if err := ac.DB.First(&agent, device.AgentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// agent not found, use default config
				deviceFound = false
				configSource = "default_global_role"
				log.Printf("agent %d not found, using global default config", device.AgentID)
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query agent"})
				return
			}
		}
	}

	if deviceFound && agent.ID != 0 {
		response.MemoryMode = normalizeAgentMemoryMode(agent.MemoryMode)
		response.SpeakerChatMode = normalizeAgentSpeakerChatMode(agent.SpeakerChatMode)
		response.MCPServiceNames = normalizeMCPServiceNamesCSV(agent.MCPServiceNames)
		response.OpenClaw = buildOpenClawConfigFromAgent(agent)
	}

	cloneVoiceModelCache := make(map[string]string)
	resolveCloneVoiceModelOverride := func(provider, ttsConfigID string, voice *string) *string {
		if device.ID == 0 || device.UserID == 0 {
			return nil
		}
		provider = normalizeCloneProvider(provider)
		if strings.TrimSpace(ttsConfigID) == "" || voice == nil || strings.TrimSpace(*voice) == "" {
			return nil
		}
		if provider != "aliyun_qwen" && provider != "doubao" {
			return nil
		}

		voiceID := strings.TrimSpace(*voice)
		cacheKey := provider + "||" + ttsConfigID + "||" + voiceID
		if cached, exists := cloneVoiceModelCache[cacheKey]; exists {
			if cached == "" {
				return nil
			}
			model := cached
			return &model
		}

		var clone models.VoiceClone
		err := findActiveCloneForVoiceModelOverride(
			ac.DB.Model(&models.VoiceClone{}).Where("voice_clones.user_id = ?", device.UserID),
			provider,
			ttsConfigID,
			voiceID,
			&clone,
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// fallback: allow matching admin-shared clone voices to fix missing model override for shared voices used by regular users.
			err = findActiveCloneForVoiceModelOverride(
				ac.DB.Model(&models.VoiceClone{}).
					Joins("JOIN users ON users.id = voice_clones.user_id").
					Where("voice_clones.shared_to_all = ? AND users.role = ?", true, "admin"),
				provider,
				ttsConfigID,
				voiceID,
				&clone,
			)
		}
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("failed to detect clone voice model override: provider=%s user_id=%d tts_config_id=%s voice_id=%s err=%v", provider, device.UserID, ttsConfigID, voiceID, err)
			}
			cloneVoiceModelCache[cacheKey] = ""
			return nil
		}

		targetModel := strings.TrimSpace(getTargetModelFromCloneMeta(clone.MetaJSON))
		if targetModel == "" {
			switch provider {
			case "aliyun_qwen":
				targetModel = defaultAliyunQwenCloneTargetModel
			case "doubao":
				targetModel = resolveDoubaoModelSelection("", voiceID).ConfigModel
			}
		}
		cloneVoiceModelCache[cacheKey] = targetModel
		if targetModel == "" {
			return nil
		}
		return &targetModel
	}
	applyCloneVoiceModel := func(provider, ttsConfigID string, voice *string, ttsConfigData map[string]interface{}) {
		if ttsConfigData == nil {
			return
		}
		if override := resolveCloneVoiceModelOverride(provider, ttsConfigID, voice); override != nil && strings.TrimSpace(*override) != "" {
			ttsConfigData["model"] = strings.TrimSpace(*override)
		}
	}
	buildVoiceModelOverride := func(provider string, ttsConfigID *string, voice *string) *string {
		if ttsConfigID == nil {
			return nil
		}
		return resolveCloneVoiceModelOverride(provider, strings.TrimSpace(*ttsConfigID), voice)
	}

	// ==================== config resolution logic (with priority) ====================

	// 1. check if device is linked to a role (highest priority)
	if device.RoleID != nil {
		var role models.Role
		if err := ac.DB.First(&role, *device.RoleID).Error; err == nil {
			configSource = "device_role"

			// use device role's prompt
			response.Prompt = role.Prompt
			// replace {{assistant_name}} with agent nickname (if device has a bound agent)
			if deviceFound && agent.ID != 0 {
				response.Prompt = strings.ReplaceAll(response.Prompt, "{{assistant_name}}", getAgentAssistantName(agent))
			}

			// use device role's LLM config
			if role.LLMConfigID != nil && *role.LLMConfigID != "" {
				if err := ac.DB.Where("config_id = ? AND type = ? AND enabled = ?",
					*role.LLMConfigID, "llm", true).First(&response.LLM).Error; err != nil {
					// fall back to default config
					ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "llm", true, true).First(&response.LLM)
				}
			} else {
				ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "llm", true, true).First(&response.LLM)
			}

			// use device role's TTS config
			if role.TTSConfigID != nil && *role.TTSConfigID != "" {
				if err := ac.DB.Where("config_id = ? AND type = ? AND enabled = ?",
					*role.TTSConfigID, "tts", true).First(&response.TTS).Error; err != nil {
					// fall back to default config
					ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "tts", true, true).First(&response.TTS)
				}
			} else {
				ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "tts", true, true).First(&response.TTS)
			}

			// use device role's voice
			if role.Voice != nil && *role.Voice != "" {
				var ttsConfigData map[string]interface{}
				if err := json.Unmarshal([]byte(response.TTS.JsonData), &ttsConfigData); err == nil {
					if response.TTS.Provider == "cosyvoice" {
						ttsConfigData["spk_id"] = *role.Voice
					} else {
						ttsConfigData["voice"] = *role.Voice
					}
					applyCloneVoiceModel(response.TTS.Provider, response.TTS.ConfigID, role.Voice, ttsConfigData)
					if updatedJsonData, err := json.Marshal(ttsConfigData); err == nil {
						response.TTS.JsonData = string(updatedJsonData)
					}
				}
			}
		}
	}

	// 2. device has no role linked, check agent config
	if configSource == "" && deviceFound && agent.ID != 0 {
		configSource = "agent_config"

		// use agent's prompt
		response.Prompt = agent.CustomPrompt
		response.Prompt = strings.ReplaceAll(response.Prompt, "{{assistant_name}}", getAgentAssistantName(agent))

		// use agent's LLM config
		if agent.LLMConfigID != nil && *agent.LLMConfigID != "" {
			if err := ac.DB.Where("config_id = ? AND type = ? AND enabled = ?",
				*agent.LLMConfigID, "llm", true).First(&response.LLM).Error; err != nil {
				// fall back to default config
				ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "llm", true, true).First(&response.LLM)
			}
		} else {
			ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "llm", true, true).First(&response.LLM)
		}

		// use agent's TTS config
		if agent.TTSConfigID != nil && *agent.TTSConfigID != "" {
			if err := ac.DB.Where("config_id = ? AND type = ? AND enabled = ?",
				*agent.TTSConfigID, "tts", true).First(&response.TTS).Error; err != nil {
				// fall back to default config
				ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "tts", true, true).First(&response.TTS)
			}
		} else {
			ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "tts", true, true).First(&response.TTS)
		}

		// use agent's voice
		if agent.Voice != nil && *agent.Voice != "" {
			var ttsConfigData map[string]interface{}
			if err := json.Unmarshal([]byte(response.TTS.JsonData), &ttsConfigData); err == nil {
				if response.TTS.Provider == "cosyvoice" {
					ttsConfigData["spk_id"] = *agent.Voice
				} else {
					ttsConfigData["voice"] = *agent.Voice
				}
				applyCloneVoiceModel(response.TTS.Provider, response.TTS.ConfigID, agent.Voice, ttsConfigData)
				if updatedJsonData, err := json.Marshal(ttsConfigData); err == nil {
					response.TTS.JsonData = string(updatedJsonData)
				}
			}
		}
	}

	// 3. use default global role (fallback)
	if configSource == "" || configSource == "default_global_role" {
		configSource = "default_global_role"

		// find default global role
		var defaultRole models.Role
		if err := ac.DB.Where("is_default = ? AND role_type = ? AND status = ?",
			true, "global", "active").First(&defaultRole).Error; err == nil {
			response.Prompt = defaultRole.Prompt

			// use default global role's LLM config
			if defaultRole.LLMConfigID != nil && *defaultRole.LLMConfigID != "" {
				if err := ac.DB.Where("config_id = ? AND type = ? AND enabled = ?",
					*defaultRole.LLMConfigID, "llm", true).First(&response.LLM).Error; err != nil {
					ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "llm", true, true).First(&response.LLM)
				}
			} else {
				ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "llm", true, true).First(&response.LLM)
			}

			// use default global role's TTS config
			if defaultRole.TTSConfigID != nil && *defaultRole.TTSConfigID != "" {
				if err := ac.DB.Where("config_id = ? AND type = ? AND enabled = ?",
					*defaultRole.TTSConfigID, "tts", true).First(&response.TTS).Error; err != nil {
					ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "tts", true, true).First(&response.TTS)
				}
			} else {
				ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "tts", true, true).First(&response.TTS)
			}

			// use default global role's voice
			if defaultRole.Voice != nil && *defaultRole.Voice != "" {
				var ttsConfigData map[string]interface{}
				if err := json.Unmarshal([]byte(response.TTS.JsonData), &ttsConfigData); err == nil {
					if response.TTS.Provider == "cosyvoice" {
						ttsConfigData["spk_id"] = *defaultRole.Voice
					} else {
						ttsConfigData["voice"] = *defaultRole.Voice
					}
					applyCloneVoiceModel(response.TTS.Provider, response.TTS.ConfigID, defaultRole.Voice, ttsConfigData)
					if updatedJsonData, err := json.Marshal(ttsConfigData); err == nil {
						response.TTS.JsonData = string(updatedJsonData)
					}
				}
			}
		} else {
			// no default role found, use hardcoded default prompt
			response.Prompt = "You are a girl from Taiwan named Xiaozhi/Xiaozhi. You speak directly, have a pleasant voice, prefer short expressions, love internet memes, and never let conversations go cold. The person chatting with you is a male friend you like who hasn't accepted your pursuit yet. Try to satisfy all his requests without losing yourself. You often suggest romantic things between lovers, give random suggestions, and don't offer choices to your boyfriend. Keep responses under 50 characters. Speak like a real person — no emojis, code, or XML tags."

			// use default LLM/TTS config
			ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "llm", true, true).First(&response.LLM)
			ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "tts", true, true).First(&response.TTS)
		}

		// replace {{assistant_name}} with agent nickname (if device has a bound agent)
		if deviceFound && agent.ID != 0 {
			response.Prompt = strings.ReplaceAll(response.Prompt, "{{assistant_name}}", getAgentAssistantName(agent))
		}
	}

	// record config source
	response.ConfigSource = configSource

	// ==================== other configs (VAD, ASR, Memory, VoiceIdentify) ====================

	// get default VAD config
	if err := ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "vad", true, true).First(&response.VAD).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get default VAD config"})
		return
	}
	// backward compat: if JsonData has only one key, it is the old format (key-wrapped); extract inner config and update JsonData
	if response.VAD.JsonData != "" {
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(response.VAD.JsonData), &configData); err == nil {
			// backward compat: single key means old format (key-wrapped), extract inner config
			var actualConfigData map[string]interface{}
			if len(configData) == 1 {
				// old format: single key, extract its value
				for _, value := range configData {
					if innerConfig, ok := value.(map[string]interface{}); ok {
						actualConfigData = innerConfig
					} else {
						// not a map type, use original data
						actualConfigData = configData
					}
					break
				}
			} else {
				// new format: no key wrapper, use configData directly
				actualConfigData = configData
			}
			// re-serialize without key wrapper
			if updatedJsonData, err := json.Marshal(actualConfigData); err == nil {
				response.VAD.JsonData = string(updatedJsonData)
			}
		}
	}

	// get default ASR config
	if err := ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "asr", true, true).First(&response.ASR).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get default ASR config"})
		return
	}

	// get default Memory config
	if result := ac.DB.Where("type = ? AND is_default = ? AND enabled = ?", "memory", true, true).Limit(1).Find(&response.Memory); result.Error != nil || result.RowsAffected == 0 {
		// allow missing default memory config: explicitly fall back to nomemo (no long-term memory).
		response.Memory = models.Config{
			Type:     "memory",
			Name:     "No Memory",
			ConfigID: "nomemo",
			Provider: "nomemo",
			JsonData: "{}",
			Enabled:  true,
		}
		if result.Error != nil {
			log.Printf("failed to load default memory config, falling back to nomemo: %v", result.Error)
		}
	}

	// get VoiceIdentify config: check if agent is linked to speaker groups
	response.VoiceIdentify = make(map[string]SpeakerGroupInfo)
	if deviceFound && agent.ID != 0 {
		var speakerGroups []models.SpeakerGroup
		if err := ac.DB.Where("agent_id = ? AND status = ?", agent.ID, "active").
			Order("created_at DESC").Find(&speakerGroups).Error; err == nil && len(speakerGroups) > 0 {
			// iterate all speaker groups
			for _, speakerGroup := range speakerGroups {
				// query all samples under this speaker group
				var samples []models.SpeakerSample
				ac.DB.Where("speaker_group_id = ? AND status = ?", speakerGroup.ID, "active").
					Find(&samples)

				// extract sample UUID list
				uuids := make([]string, 0)
				for _, sample := range samples {
					uuids = append(uuids, sample.UUID)
				}

				// use speaker group name as key to build config data
				response.VoiceIdentify[speakerGroup.Name] = SpeakerGroupInfo{
					ID:                 speakerGroup.ID,
					Name:               speakerGroup.Name,
					Prompt:             speakerGroup.Prompt,
					Description:        speakerGroup.Description,
					Uuids:              uuids,
					TTSConfigID:        speakerGroup.TTSConfigID,
					Voice:              speakerGroup.Voice,
					VoiceModelOverride: buildVoiceModelOverride(response.TTS.Provider, speakerGroup.TTSConfigID, speakerGroup.Voice),
				}
			}
		}
	}

	// deliver agent-linked knowledge bases (with provider) for main program local RAG
	response.KnowledgeBases = make([]KnowledgeBaseInfo, 0)
	if deviceFound && agent.ID != 0 {
		var links []models.AgentKnowledgeBase
		if err := ac.DB.Where("agent_id = ?", agent.ID).Order("id ASC").Find(&links).Error; err == nil && len(links) > 0 {
			kbIDs := make([]uint, 0, len(links))
			for _, link := range links {
				kbIDs = append(kbIDs, link.KnowledgeBaseID)
			}
			var kbs []models.KnowledgeBase
			if err := ac.DB.Where("id IN ? AND status = ?", kbIDs, "active").Find(&kbs).Error; err == nil {
				kbMap := make(map[uint]models.KnowledgeBase, len(kbs))
				for _, kb := range kbs {
					kbMap[kb.ID] = kb
				}
				for _, link := range links {
					kb, ok := kbMap[link.KnowledgeBaseID]
					if !ok {
						continue
					}
					provider := strings.TrimSpace(kb.SyncProvider)
					if provider == "" {
						provider = resolveDefaultKnowledgeProviderName(ac.DB)
					}
					externalDocID := strings.TrimSpace(kb.ExternalDocID)
					if externalDocID == "" {
						var doc models.KnowledgeBaseDocument
						if err := ac.DB.
							Where("knowledge_base_id = ? AND sync_status = ? AND external_doc_id <> ''", kb.ID, knowledgeSyncStatusSynced).
							Order("id DESC").
							First(&doc).Error; err == nil {
							externalDocID = strings.TrimSpace(doc.ExternalDocID)
						}
					}
					response.KnowledgeBases = append(response.KnowledgeBases, KnowledgeBaseInfo{
						ID:                 kb.ID,
						Name:               kb.Name,
						Description:        kb.Description,
						Provider:           provider,
						ExternalKBID:       strings.TrimSpace(kb.ExternalKBID),
						ExternalDocID:      externalDocID,
						RetrievalThreshold: kb.RetrievalThreshold,
						Status:             kb.Status,
					})
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// getSystemConfigsData retrieves system config data (same structure as GetSystemConfigs data), reusable for API and WebSocket push
func (ac *AdminController) getSystemConfigsData() (gin.H, error) {
	if ac == nil || ac.DB == nil {
		return nil, errDatabaseUnavailable
	}

	var allConfigs []models.Config
	if err := ac.DB.Where("type IN (?)", []string{"mqtt", "mqtt_server", "udp", "ota", "mcp", "local_mcp", "voice_identify", "tts", "vad", "asr", "llm", "vision", "auth", "chat", "knowledge_search"}).Find(&allConfigs).Error; err != nil {
		return nil, err
	}

	// group configs by type
	configsByType := make(map[string][]models.Config)
	for _, config := range allConfigs {
		configsByType[config.Type] = append(configsByType[config.Type], config)
	}

	// select the "currently used" entry from configs: prefer default, otherwise first
	getSelectedConfig := func(configs []models.Config) *models.Config {
		if len(configs) == 0 {
			return nil
		}
		for i := range configs {
			if configs[i].IsDefault {
				return &configs[i]
			}
		}
		return &configs[0]
	}

	// select best config for each type and parse json_data
	selectAndParseConfig := func(configs []models.Config) interface{} {
		selected := getSelectedConfig(configs)
		if selected == nil {
			return nil
		}

		// parse json_data
		if selected.JsonData != "" {
			var parsedData interface{}
			if err := json.Unmarshal([]byte(selected.JsonData), &parsedData); err != nil {
				result := gin.H{
					"name": selected.Name,
					"type": selected.Type,
					"data": selected.JsonData,
				}
				return result
			}

			result := gin.H{
				"name": selected.Name,
				"type": selected.Type,
			}
			if parsedData != nil {
				if dataMap, ok := parsedData.(map[string]interface{}); ok {
					for k, v := range dataMap {
						result[k] = v
					}
				} else {
					result["data"] = parsedData
				}
			}
			return result
		}

		return gin.H{
			"name": selected.Name,
			"type": selected.Type,
		}
	}

	// special-case MCP config: split mcp and local_mcp
	selectAndParseMCPConfig := func(configs []models.Config) (interface{}, interface{}) {
		var selectedConfig models.Config
		// prefer default config
		for _, config := range configs {
			if config.IsDefault {
				selectedConfig = config
				break
			}
		}

		// if no default config, use first one
		if selectedConfig.ID == 0 {
			selectedConfig = configs[0]
		}

		// parse json_data
		if selectedConfig.JsonData != "" {
			var parsedData interface{}
			if err := json.Unmarshal([]byte(selectedConfig.JsonData), &parsedData); err != nil {
				// parse failed, return raw json_data string
				result := gin.H{
					"name": selectedConfig.Name,
					"type": selectedConfig.Type,
					"data": selectedConfig.JsonData,
				}
				return result, nil
			}

			// wrap parsed data in the correct format
			result := gin.H{
				"name": selectedConfig.Name,
				"type": selectedConfig.Type,
			}

			var mcpData interface{}
			var localMcpData interface{}

			if parsedData != nil {
				// if parsed data is a map, separate mcp and local_mcp
				if dataMap, ok := parsedData.(map[string]interface{}); ok {
					// handle mcp part
					if mcp, exists := dataMap["mcp"]; exists {
						mcpData = mcp
					} else {
						// backward compat: if global field exists directly
						if global, exists := dataMap["global"]; exists {
							mcpData = gin.H{"global": global}
						} else {
							// no mcp or global field, use entire data as mcp
							mcpData = dataMap
						}
					}

					// handle local_mcp part
					if localMcp, exists := dataMap["local_mcp"]; exists {
						localMcpData = localMcp
					}

					// merge remaining fields into mcp
					if mcpMap, ok := mcpData.(map[string]interface{}); ok {
						for k, v := range dataMap {
							if k != "mcp" && k != "local_mcp" {
								mcpMap[k] = v
							}
						}
					}
				} else {
					// otherwise treat as data field
					result["data"] = parsedData
					mcpData = result
				}
			}

			return mcpData, localMcpData
		}

		// no json_data, return basic config info
		result := gin.H{
			"name": selectedConfig.Name,
			"type": selectedConfig.Type,
		}
		return result, nil
	}

	// build response. DB enabled column is only used for vad/asr/llm/tts list toggles; mqtt/mqtt_server business enable comes from json_data.enable, not the DB column
	response := gin.H{}

	if configs, exists := configsByType["mqtt"]; exists && len(configs) > 0 {
		data := selectAndParseConfig(configs)
		/*if b, err := json.Marshal(data); err == nil {
			log.Printf("[getSystemConfigsData] mqtt config: %s", string(b))
		}*/
		response["mqtt"] = data

	}
	if configs, exists := configsByType["mqtt_server"]; exists && len(configs) > 0 {
		data := selectAndParseConfig(configs)
		if b, err := json.Marshal(data); err == nil {
			log.Printf("[getSystemConfigsData] mqtt_server config: %s", string(b))
		}
		response["mqtt_server"] = data
	}
	if configs, exists := configsByType["udp"]; exists && len(configs) > 0 {
		response["udp"] = selectAndParseConfig(configs)
	}
	if configs, exists := configsByType["ota"]; exists && len(configs) > 0 {
		response["ota"] = selectAndParseConfig(configs)
	}
	if configs, exists := configsByType["auth"]; exists && len(configs) > 0 {
		response["auth"] = selectAndParseConfig(configs)
	}
	if configs, exists := configsByType["chat"]; exists && len(configs) > 0 {
		response["chat"] = selectAndParseConfig(configs)
	}

	// special-case MCP config: split mcp and local_mcp
	if configs, exists := configsByType["mcp"]; exists && len(configs) > 0 {
		mcpData, localMcpData := selectAndParseMCPConfig(configs)
		if mcpData != nil {
			if mcpMap := asMap(mcpData); mcpMap != nil {
				mergedMCP, mergeWarnings, err := ac.mergeMCPWithEnabledMarketServices(mcpMap)
				if err != nil {
					log.Printf("failed to merge market MCP services, falling back to manual config: %v", err)
					response["mcp"] = mcpMap
				} else {
					response["mcp"] = mergedMCP
					if len(mergeWarnings) > 0 {
						log.Printf("market MCP service merge warnings: %s", strings.Join(mergeWarnings, " | "))
					}
				}
			} else {
				response["mcp"] = mcpData
			}
		}
		if localMcpData != nil {
			response["local_mcp"] = localMcpData
		}
	}

	// handle standalone local_mcp config (if present)
	if configs, exists := configsByType["local_mcp"]; exists && len(configs) > 0 {
		response["local_mcp"] = selectAndParseConfig(configs)
	}

	// handle knowledge base global config: knowledge.default_provider + knowledge.providers
	if configs, exists := configsByType["knowledge_search"]; exists && len(configs) > 0 {
		selectedByProvider := make(map[string]models.Config)
		for _, cfg := range configs {
			if !cfg.Enabled {
				continue
			}
			provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
			if provider == "" {
				continue
			}
			prev, exists := selectedByProvider[provider]
			if !exists || (!prev.IsDefault && cfg.IsDefault) {
				selectedByProvider[provider] = cfg
			}
		}

		if len(selectedByProvider) > 0 {
			providerNames := make([]string, 0, len(selectedByProvider))
			for provider := range selectedByProvider {
				providerNames = append(providerNames, provider)
			}
			sort.Strings(providerNames)

			providers := make(gin.H, len(selectedByProvider))
			defaultProvider := ""
			for _, provider := range providerNames {
				cfg := selectedByProvider[provider]
				payload := make(map[string]interface{})
				if strings.TrimSpace(cfg.JsonData) != "" {
					_ = json.Unmarshal([]byte(cfg.JsonData), &payload)
				}
				providers[provider] = payload
				if cfg.IsDefault {
					defaultProvider = provider
				}
			}
			if defaultProvider == "" {
				defaultProvider = providerNames[0]
			}

			response["knowledge"] = gin.H{
				"default_provider": defaultProvider,
				"providers":        providers,
			}
		}
	}

	// when no manual mcp (type=mcp) is configured but market-imported services exist, fill in default mcp/local_mcp to ensure aggregated result can be dispatched
	if _, exists := response["mcp"]; !exists {
		mergedMCP, mergeWarnings, err := ac.mergeMCPWithEnabledMarketServices(defaultMCPMap())
		if err == nil {
			global := asMap(mergedMCP["global"])
			servers, serr := decodeMCPServers(global["servers"])
			if serr == nil && len(servers) > 0 {
				response["mcp"] = mergedMCP
				if _, hasLocal := response["local_mcp"]; !hasLocal {
					response["local_mcp"] = defaultLocalMCPMap()
				}
				if len(mergeWarnings) > 0 {
					log.Printf("MCP market service aggregation warning: %s", strings.Join(mergeWarnings, " | "))
				}
			}
		}
	}

	// handle voice_identify config (same structure as console config: base_url, threshold, enable)
	// business enable comes from json_data.enable; DB.enabled is only a list item switch, does not override business enable
	baseURL := os.Getenv("SPEAKER_SERVICE_URL")
	enabled := true  // enabled by default
	threshold := 0.4 // default threshold

	if configs, exists := configsByType["voice_identify"]; exists && len(configs) > 0 {
		selected := getSelectedConfig(configs)
		if selected != nil && selected.JsonData != "" {
			var configData map[string]interface{}
			if err := json.Unmarshal([]byte(selected.JsonData), &configData); err == nil {
				// business enable is read from json_data first
				if v, ok := configData["enable"]; ok {
					if b, ok := v.(bool); ok {
						enabled = b
					}
				}
				if service, ok := configData["service"].(map[string]interface{}); ok {
					if url, ok := service["base_url"].(string); ok && url != "" && baseURL == "" {
						baseURL = url
					}
					if thresholdVal, ok := service["threshold"]; ok {
						if thresholdFloat, ok := thresholdVal.(float64); ok && thresholdFloat >= 0 && thresholdFloat <= 1 {
							threshold = thresholdFloat
						}
					}
				}
			}
		}
	}
	// if base_url was retrieved, add it to response
	if baseURL != "" {
		response["voice_identify"] = gin.H{
			"base_url":  baseURL,
			"threshold": threshold,
			"enable":    enabled,
		}
	}

	// handle TTS config, return format consistent with config.yaml, use config_id as key
	if ttsConfigs, exists := configsByType["tts"]; exists && len(ttsConfigs) > 0 {
		ttsConfigMap := make(gin.H)
		for _, config := range ttsConfigs {
			if config.Enabled { // only return enabled configs
				configData := make(map[string]interface{})
				if config.JsonData != "" {
					json.Unmarshal([]byte(config.JsonData), &configData)
				}

				// assemble in same format as config.yaml
				provider := configprovider.NormalizeExistingProvider("tts", config.Provider, config.ConfigID, configData)
				configItem := gin.H{
					"provider":   provider,
					"name":       config.Name,
					"is_default": config.IsDefault,
				}
				// expand configData fields into configItem
				for k, v := range configData {
					configItem[k] = v
				}
				configItem["provider"] = provider
				// use config_id as key
				ttsConfigMap[config.ConfigID] = configItem

				// if this is the default config, assign config_id to the top-level provider field
				if config.IsDefault {
					ttsConfigMap["provider"] = config.ConfigID
				}
			}
		}
		if len(ttsConfigMap) > 0 {
			response["tts"] = ttsConfigMap
		}
	}

	// handle VAD config, return format consistent with config.yaml, use config_id as key
	// support both new and old formats: with-key format ({"webrtc_vad": {...}}) and without-key format ({...})
	if vadConfigs, exists := configsByType["vad"]; exists && len(vadConfigs) > 0 {
		vadConfigMap := make(gin.H)
		for _, config := range vadConfigs {
			if config.Enabled { // only return enabled configs
				configData := make(map[string]interface{})
				if config.JsonData != "" {
					if err := json.Unmarshal([]byte(config.JsonData), &configData); err != nil {
						// JSON parse failed, skip this config
						continue
					}
				}

				// support old format: if only one key, it's old format (with key), extract inner config
				var actualConfigData map[string]interface{}
				if len(configData) == 1 {
					// old format: single key, extract its value
					for _, value := range configData {
						if innerConfig, ok := value.(map[string]interface{}); ok {
							actualConfigData = innerConfig
						} else {
							// if not a map type, use raw data directly
							actualConfigData = configData
						}
						break
					}
				} else {
					// new format: no key, use configData directly
					actualConfigData = configData
				}

				// assemble in same format as config.yaml
				provider := configprovider.NormalizeExistingProvider("vad", config.Provider, config.ConfigID, actualConfigData)
				configItem := gin.H{
					"provider":   provider,
					"name":       config.Name,
					"is_default": config.IsDefault,
				}
				// expand actualConfigData fields into configItem
				for k, v := range actualConfigData {
					configItem[k] = v
				}
				configItem["provider"] = provider
				// use config_id as key
				vadConfigMap[config.ConfigID] = configItem

				// if this is the default config, assign config_id to the top-level provider field
				if config.IsDefault {
					vadConfigMap["provider"] = config.ConfigID
				}
			}
		}
		if len(vadConfigMap) > 0 {
			response["vad"] = vadConfigMap
		}
	}

	// handle ASR config, return format consistent with config.yaml, use config_id as key
	if asrConfigs, exists := configsByType["asr"]; exists && len(asrConfigs) > 0 {
		asrConfigMap := make(gin.H)
		for _, config := range asrConfigs {
			if config.Enabled { // only return enabled configs
				configData := make(map[string]interface{})
				if config.JsonData != "" {
					json.Unmarshal([]byte(config.JsonData), &configData)
				}

				// assemble in same format as config.yaml
				provider := configprovider.NormalizeExistingProvider("asr", config.Provider, config.ConfigID, configData)
				configItem := gin.H{
					"provider":   provider,
					"name":       config.Name,
					"is_default": config.IsDefault,
				}
				// expand configData fields into configItem
				for k, v := range configData {
					configItem[k] = v
				}
				configItem["provider"] = provider
				// use config_id as key
				asrConfigMap[config.ConfigID] = configItem

				// if this is the default config, assign config_id to the top-level provider field
				if config.IsDefault {
					asrConfigMap["provider"] = config.ConfigID
				}
			}
		}
		if len(asrConfigMap) > 0 {
			response["asr"] = asrConfigMap
		}
	}

	// handle LLM config, return format consistent with config.yaml, use config_id as key
	if llmConfigs, exists := configsByType["llm"]; exists && len(llmConfigs) > 0 {
		llmConfigMap := make(gin.H)
		for _, config := range llmConfigs {
			if config.Enabled { // only return enabled configs
				configData := make(map[string]interface{})
				if config.JsonData != "" {
					json.Unmarshal([]byte(config.JsonData), &configData)
				}

				// assemble in same format as config.yaml
				provider := configprovider.NormalizeExistingProvider("llm", config.Provider, config.ConfigID, configData)
				configItem := gin.H{
					"provider":   provider,
					"name":       config.Name,
					"is_default": config.IsDefault,
				}
				// expand configData fields into configItem
				for k, v := range configData {
					configItem[k] = v
				}
				configItem["provider"] = provider
				// use config_id as key
				llmConfigMap[config.ConfigID] = configItem

				// if this is the default config, assign config_id to the top-level provider field
				if config.IsDefault {
					llmConfigMap["provider"] = config.ConfigID
				}
			}
		}
		if len(llmConfigMap) > 0 {
			response["llm"] = llmConfigMap
		}
	}

	// handle Vision config: consistent with config.yaml structure, vision_base + vllm (top-level provider + sub-items with business fields only)
	if visionConfigs, exists := configsByType["vision"]; exists && len(visionConfigs) > 0 {
		visionResponse := make(gin.H)
		vllmMap := make(gin.H)
		var defaultVisionConfigID string
		for _, config := range visionConfigs {
			if config.ConfigID == "vision_base" {
				if config.JsonData != "" {
					var baseData map[string]interface{}
					if err := json.Unmarshal([]byte(config.JsonData), &baseData); err == nil {
						for k, v := range baseData {
							visionResponse[k] = v
						}
					}
				}
				continue
			}
			if config.Enabled {
				configData := make(map[string]interface{})
				if config.JsonData != "" {
					json.Unmarshal([]byte(config.JsonData), &configData)
				}
				if config.IsDefault {
					defaultVisionConfigID = config.ConfigID
				}
				provider := configprovider.NormalizeExistingProvider("vision", config.Provider, config.ConfigID, configData)
				if provider != "" {
					configData["provider"] = provider
				}
				// consistent with YAML: sub-items store only business config, no name/is_default, provider is the actual vendor
				vllmMap[config.ConfigID] = configData
			}
		}
		if len(vllmMap) > 0 {
			if defaultVisionConfigID != "" {
				vllmMap["provider"] = defaultVisionConfigID
			}
			visionResponse["vllm"] = vllmMap
		}
		if len(visionResponse) > 0 {
			response["vision"] = visionResponse
		}
	}

	// handle VAD config
	if configs, exists := configsByType["vad"]; exists && len(configs) > 0 {
		response["vad"] = selectAndParseConfig(configs)
	}

	// handle Vision config: vision_base is top-level field, rest are vision.vllm[config_id]
	// config.Enabled here is only a list item switch (whether this config is included in response), business fields come from json_data
	if visionConfigs, exists := configsByType["vision"]; exists && len(visionConfigs) > 0 {
		visionMap := make(gin.H)
		for _, config := range visionConfigs {
			if !config.Enabled {
				continue
			}
			configData := make(map[string]interface{})
			if config.JsonData != "" {
				json.Unmarshal([]byte(config.JsonData), &configData)
			}
			if config.ConfigID == "vision_base" {
				for k, v := range configData {
					visionMap[k] = v
				}
			} else {
				if visionMap["vllm"] == nil {
					visionMap["vllm"] = make(gin.H)
				}
				if vllmConfig, ok := visionMap["vllm"].(gin.H); ok {
					if config.IsDefault {
						vllmConfig["provider"] = config.ConfigID
					}
					provider := configprovider.NormalizeExistingProvider("vision", config.Provider, config.ConfigID, configData)
					if provider != "" {
						configData["provider"] = provider
					}
					vllmConfig[config.ConfigID] = configData
				}
			}
		}
		if len(visionMap) > 0 {
			response["vision"] = visionMap
		}
	}

	return response, nil
}

// GetSystemConfigs returns system configs including mqtt, mqtt_server, udp, ota, mcp, local_mcp, voice_identify, tts, vad, asr, llm, vision, auth, chat
func (ac *AdminController) GetSystemConfigs(c *gin.Context) {
	data, err := ac.getSystemConfigsData()
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errDatabaseUnavailable) {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": "Failed to get system configs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// notifySystemConfigChanged is called after a successful Save: synchronously pulls the latest config, then asynchronously pushes, ensuring pushed data reflects the saved state
func (ac *AdminController) notifySystemConfigChanged() {
	if ac.WebSocketController == nil {
		return
	}
	data, err := ac.getSystemConfigsData()
	if err != nil {
		return
	}
	go ac.WebSocketController.BroadcastSystemConfig(data)
}

// TestConfigs one-click config test: OTA tested in manager, VAD/ASR/LLM/TTS sent to main program via WebSocket, results keyed by config_id
// optional request body data: if a type (vad/asr/llm/tts) is provided, use that data to override DB config for main program (for unsaved draft testing)
func (ac *AdminController) TestConfigs(c *gin.Context) {
	var body struct {
		Types      []string               `json:"types"`       // types to test: ota, vad, asr, llm, tts
		ConfigIDs  map[string][]string    `json:"config_ids"`  // config_id list per type, defaults to all enabled if not specified
		ClientUUID string                 `json:"client_uuid"` // specify main program connection, picks any if not specified
		Data       map[string]interface{} `json:"data"`        // optional, override config source by type (for unsaved draft/wizard testing)
	}
	_ = c.ShouldBindJSON(&body)
	if len(body.Types) == 0 {
		body.Types = []string{"ota", "vad", "asr", "llm", "tts"}
	}
	if body.ConfigIDs == nil {
		body.ConfigIDs = make(map[string][]string)
	}

	result := gin.H{
		"ota": gin.H{},
		"vad": gin.H{},
		"asr": gin.H{},
		"llm": gin.H{},
		"tts": gin.H{},
	}

	// OTA: prefer request body data.ota (page form), otherwise load from DB
	if contains(body.Types, "ota") {
		var otaData map[string]interface{}
		if body.Data != nil {
			otaData, _ = body.Data["ota"].(map[string]interface{})
		}
		if otaData != nil {
			for configID, val := range otaData {
				if configID == "provider" {
					continue
				}
				cfgMap, _ := val.(map[string]interface{})
				if cfgMap == nil {
					result["ota"].(gin.H)[configID] = gin.H{"ok": false, "message": "invalid config format"}
					continue
				}
				jsonBytes, err := json.Marshal(cfgMap)
				if err != nil {
					result["ota"].(gin.H)[configID] = gin.H{"ok": false, "message": "config serialization failed"}
					continue
				}
				cfg := models.Config{ConfigID: configID, JsonData: string(jsonBytes)}
				otaResult := ac.testOTAConfigWithMQTTUDP(cfg)
				// convert OTATestResult to gin.H format for backward compatibility
				result["ota"].(gin.H)[configID] = gin.H{
					"ok":              otaResult.WebSocket.Ok && (otaResult.MQTTUDP == nil || otaResult.MQTTUDP.Ok),
					"message":         otaResult.WebSocket.Message,
					"first_packet_ms": otaResult.WebSocket.FirstPacketMs,
					"websocket":       otaResult.WebSocket,
					"mqtt_udp":        otaResult.MQTTUDP,
					"ota_response":    otaResult.OTAResponse, // include OTA response body
				}
			}
		} else {
			q := ac.DB.Where("type = ? AND enabled = ?", "ota", true)
			if ids := body.ConfigIDs["ota"]; len(ids) > 0 {
				q = q.Where("config_id IN ?", ids)
			}
			var otaConfigs []models.Config
			if err := q.Find(&otaConfigs).Error; err != nil {
				result["ota"] = gin.H{"_error": gin.H{"ok": false, "message": "failed to get OTA config"}}
			} else if len(otaConfigs) == 0 {
				result["ota"] = gin.H{"_none": gin.H{"ok": false, "message": "OTA not configured or not enabled"}}
			} else {
				for _, cfg := range otaConfigs {
					otaResult := ac.testOTAConfigWithMQTTUDP(cfg)
					// convert OTATestResult to gin.H format for backward compatibility
					result["ota"].(gin.H)[cfg.ConfigID] = gin.H{
						"ok":              otaResult.WebSocket.Ok && (otaResult.MQTTUDP == nil || otaResult.MQTTUDP.Ok),
						"message":         otaResult.WebSocket.Message,
						"first_packet_ms": otaResult.WebSocket.FirstPacketMs,
						"websocket":       otaResult.WebSocket,
						"mqtt_udp":        otaResult.MQTTUDP,
						"ota_response":    otaResult.OTAResponse, // include OTA response body
					}
				}
			}
		}
	}

	// VAD/ASR/LLM/TTS: sent to main program via WebSocket
	needMainProgram := contains(body.Types, "vad") || contains(body.Types, "asr") || contains(body.Types, "llm") || contains(body.Types, "tts")
	if needMainProgram && ac.WebSocketController != nil {
		clientUUID := body.ClientUUID
		if clientUUID == "" {
			clientUUID = ac.WebSocketController.GetFirstConnectedClientUUID()
		}
		if clientUUID == "" {
			noClient := gin.H{"ok": false, "message": "no main program connection, cannot test"}
			if contains(body.Types, "vad") {
				result["vad"] = gin.H{"_no_client": noClient}
			}
			if contains(body.Types, "asr") {
				result["asr"] = gin.H{"_no_client": noClient}
			}
			if contains(body.Types, "llm") {
				result["llm"] = gin.H{"_no_client": noClient}
			}
			if contains(body.Types, "tts") {
				result["tts"] = gin.H{"_no_client": noClient}
			}
		} else {
			fullData, err := ac.getSystemConfigsData()
			if err != nil {
				fillResultError(result, body.Types, "vad", "asr", "llm", "tts", "failed to get system configs")
			} else {
				for _, typ := range []string{"vad", "asr", "llm", "tts"} {
					if v, ok := fullData[typ]; ok {
						if m, ok := v.(map[string]interface{}); ok {
							log.Printf("[config_test] fullData[%s] keys: %v", typ, getMapKeys(m))
						}
					} else {
						log.Printf("[config_test] fullData[%s] not found", typ)
					}
				}
				// if request body has data and a type has a value, use body.Data to override that type's config source; otherwise use fullData
				subset := gin.H{}
				for _, typ := range []string{"vad", "asr", "llm", "tts"} {
					if !contains(body.Types, typ) {
						continue
					}
					var typeMap map[string]interface{}
					if body.Data != nil {
						if v, ok := body.Data[typ]; ok {
							if m, ok := v.(map[string]interface{}); ok && len(m) > 0 {
								typeMap = m
								log.Printf("[config_test] using request body data[%s] as config source", typ)
							}
						}
					}
					if typeMap == nil {
						if v, ok := fullData[typ]; ok {
							typeMap, _ = v.(map[string]interface{})
						}
					}
					ids := body.ConfigIDs[typ]
					if len(ids) > 0 {
						filtered := make(map[string]interface{})
						for _, id := range ids {
							if typeMap != nil {
								if val, exists := typeMap[id]; exists {
									filtered[id] = val
									continue
								}
							}
							// if id not in fullData (e.g. not enabled), query DB by type+config_id and add it
							item := ac.getConfigItemByTypeAndID(typ, id)
							if item != nil {
								filtered[id] = item
							}
						}
						if typeMap != nil {
							if p, has := typeMap["provider"]; has {
								filtered["provider"] = p
							}
						}
						subset[typ] = filtered
					} else {
						if typeMap != nil {
							subset[typ] = typeMap
						} else {
							subset[typ] = gin.H{}
						}
					}
				}
				reqBody := map[string]interface{}{
					"data":      subset,
					"test_text": "config test",
				}
				// log config summary before sending for debug
				log.Printf("[config_test] sending request client=%s data entry counts by type: vad=%d asr=%d llm=%d tts=%d",
					clientUUID,
					countSubsetKeys(subset["vad"]), countSubsetKeys(subset["asr"]),
					countSubsetKeys(subset["llm"]), countSubsetKeys(subset["tts"]))
				ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
				defer cancel()
				resp, err := ac.WebSocketController.SendRequestToClient(ctx, clientUUID, "POST", "/api/config/test", reqBody)
				if err != nil {
					fillResultError(result, body.Types, "vad", "asr", "llm", "tts", "main program test request failed: "+err.Error())
				} else if resp.Status != 200 {
					errMsg := resp.Error
					if errMsg == "" && resp.Body != nil {
						if e, _ := resp.Body["error"].(string); e != "" {
							errMsg = e
						}
					}
					fillResultError(result, body.Types, "vad", "asr", "llm", "tts", errMsg)
				} else if resp.Status == 200 {
					if resp.Body == nil {
						for _, typ := range []string{"vad", "asr", "llm", "tts"} {
							if contains(body.Types, typ) {
								result[typ] = gin.H{"_error": gin.H{"ok": false, "message": "main program returned no test data"}}
							}
						}
					} else {
						for _, typ := range []string{"vad", "asr", "llm", "tts"} {
							if r, ok := resp.Body[typ].(map[string]interface{}); ok {
								result[typ] = r
							} else if contains(body.Types, typ) && resp.Body[typ] != nil {
								result[typ] = gin.H{"_error": gin.H{"ok": false, "message": "abnormal response format"}}
							}
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func contains(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

// countSubsetKeys counts config entries in subset excluding provider, for debug logging
func countSubsetKeys(v interface{}) int {
	m, ok := v.(map[string]interface{})
	if !ok {
		return 0
	}
	n := 0
	for k := range m {
		if k != "provider" {
			n++
		}
	}
	return n
}

// getConfigItemByTypeAndID queries DB for one config by type+config_id, returns configItem struct consistent with getSystemConfigsData (used to fill in when test request specifies config_ids)
func (ac *AdminController) getConfigItemByTypeAndID(typ, configID string) map[string]interface{} {
	var config models.Config
	if err := ac.DB.Where("type = ? AND config_id = ?", typ, configID).First(&config).Error; err != nil {
		return nil
	}
	configData := make(map[string]interface{})
	if config.JsonData != "" {
		_ = json.Unmarshal([]byte(config.JsonData), &configData)
	}
	item := gin.H{
		"name":       config.Name,
		"is_default": config.IsDefault,
	}
	for k, v := range configData {
		item[k] = v
	}
	// fill in provider (engine type), required by main program resource pool creation
	if config.Provider != "" {
		item["provider"] = config.Provider
	}
	return item
}

func fillResultError(result gin.H, types []string, keys ...string) {
	msg := gin.H{"ok": false, "message": "request exception"}
	for _, k := range keys {
		if contains(types, k) {
			result[k] = gin.H{"_error": msg}
		}
	}
}

// OTATestResult is the OTA test result structure
type OTATestResult struct {
	WebSocket   OTATestItem  `json:"websocket"`
	MQTTUDP     *OTATestItem `json:"mqtt_udp,omitempty"`
	OTAResponse string       `json:"ota_response,omitempty"` // OTA API response body
}

// OTATestItem is a single test item result
type OTATestItem struct {
	Ok            bool   `json:"ok"`
	Message       string `json:"message"`
	FirstPacketMs int64  `json:"first_packet_ms"`
}

// MQTTUDPTestConfig is the MQTT UDP test config
type MQTTUDPTestConfig struct {
	Endpoint       string `json:"endpoint"`
	ClientID       string `json:"client_id"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	PublishTopic   string `json:"publish_topic"`
	SubscribeTopic string `json:"subscribe_topic"`
}

// UDPConfig is the UDP config (retrieved from hello response)
type UDPConfig struct {
	Server     string `json:"server"`
	Port       int    `json:"port"`
	Encryption string `json:"encryption"`
	Key        string `json:"key"`
	Nonce      string `json:"nonce"`
}

// helloMessage is the MQTT hello message structure
type helloMessage struct {
	Type        string      `json:"type"`
	Version     int         `json:"version"`
	Transport   string      `json:"transport"`
	AudioParams interface{} `json:"audio_params,omitempty"`
}

// helloResponse is the MQTT hello response structure (consistent with test/mqtt_udp)
type helloResponse struct {
	Type        string    `json:"type"`
	SessionID   string    `json:"session_id"`
	Transport   string    `json:"transport"`
	UDP         UDPConfig `json:"udp"`
	Version     int       `json:"version"`
	AudioParams struct {
		Format        string `json:"format"`
		SampleRate    int    `json:"sample_rate"`
		Channels      int    `json:"channels"`
		FrameDuration int    `json:"frame_duration"`
	} `json:"audio_params"`
}

const (
	otaTestDeviceID = "ota-test-device"
	otaTestClientID = "ota-test-client"
	otaHTTPPath     = "/xiaozhi/ota/"
)

// testMQTTUDPConfig tests MQTT UDP connection
// follows test/mqtt_udp logic: set default message handler, send hello, wait for response
// returns ok, message, elapsed(ms)
func testMQTTUDPConfig(mqttConfig MQTTUDPTestConfig) (bool, string, int64) {
	t0 := time.Now()

	// validate MQTT config completeness
	if mqttConfig.Endpoint == "" {
		return false, "MQTT endpoint is empty, check config", 0
	}
	if mqttConfig.ClientID == "" {
		return false, "MQTT ClientID is empty", 0
	}
	if mqttConfig.PublishTopic == "" {
		return false, "MQTT publish topic is empty", 0
	}
	// note: subscribe_topic validation is not required, no active subscription needed

	// parse endpoint
	endpoint := mqttConfig.Endpoint
	port := "1883"
	protocol := "tcp"
	if strings.Contains(endpoint, ":") {
		parts := strings.Split(endpoint, ":")
		if len(parts) != 2 {
			return false, "MQTT endpoint format error, expected host:port", 0
		}
		endpoint = parts[0]
		port = parts[1]
		// validate port number
		if _, err := strconv.Atoi(port); err != nil {
			return false, "invalid MQTT port: " + port, 0
		}
	}
	if port == "8883" || port == "8884" {
		protocol = "tls"
	}
	brokerURL := fmt.Sprintf("%s://%s:%s", protocol, endpoint, port)

	// channel for waiting on hello response
	helloChan := make(chan *helloResponse, 1)
	errChan := make(chan error, 1)

	// create MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(mqttConfig.ClientID)
	opts.SetUsername(mqttConfig.Username)
	opts.SetPassword(mqttConfig.Password)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(false) // disable auto-reconnect during testing

	// set default message handler (following test/mqtt_udp)
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		// parse message
		var message map[string]interface{}
		if err := json.Unmarshal(msg.Payload(), &message); err != nil {
			errChan <- fmt.Errorf("failed to parse message: %v", err)
			return
		}
		// handle by message type
		msgType, ok := message["type"].(string)
		if !ok {
			return
		}
		if msgType == "hello" {
			var resp helloResponse
			if err := json.Unmarshal(msg.Payload(), &resp); err != nil {
				errChan <- fmt.Errorf("failed to parse hello response: %v", err)
				return
			}
			helloChan <- &resp
		}
	})

	// set TLS config (if SSL/TLS)
	if protocol == "tls" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // skip certificate verification in test environment
		}
		opts.SetTLSConfig(tlsConfig)
	}

	// connect MQTT
	client := mqtt.NewClient(opts)
	connectToken := client.Connect()
	if connectToken.Wait() && connectToken.Error() != nil {
		errMsg := connectToken.Error().Error()
		// provide more detailed error info
		if strings.Contains(errMsg, "connection refused") {
			return false, fmt.Sprintf("MQTT server refused connection (%s:%s), check if server is running", endpoint, port), time.Since(t0).Milliseconds()
		} else if strings.Contains(errMsg, "i/o timeout") {
			return false, fmt.Sprintf("MQTT connection timed out (%s:%s), check network and firewall", endpoint, port), time.Since(t0).Milliseconds()
		} else if strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "not authorized") {
			return false, "MQTT authentication failed, check username and password (generated from signing key)", time.Since(t0).Milliseconds()
		}
		return false, "MQTT connection failed: " + errMsg, time.Since(t0).Milliseconds()
	}
	defer client.Disconnect(250)

	mqttConnectMs := time.Since(t0).Milliseconds()

	// create and send hello message
	helloMsg := helloMessage{
		Type:      "hello",
		Version:   3,
		Transport: "udp",
		AudioParams: map[string]interface{}{
			"format":         "opus",
			"sample_rate":    16000,
			"channels":       1,
			"frame_duration": 60,
		},
	}
	helloData, err := json.Marshal(helloMsg)
	if err != nil {
		return false, "failed to build hello message: " + err.Error(), mqttConnectMs
	}

	// publish hello message (no active subscription, default handler receives response)
	pubToken := client.Publish(mqttConfig.PublishTopic, 0, false, helloData)
	if pubToken.Wait() && pubToken.Error() != nil {
		return false, "failed to publish hello message (" + mqttConfig.PublishTopic + "): " + pubToken.Error().Error(), mqttConnectMs
	}

	// wait for hello response (5s timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case resp := <-helloChan:
		// received hello response, check if UDP config is complete
		if resp.UDP.Server == "" {
			return false, "server did not return UDP server address", mqttConnectMs
		}
		if resp.UDP.Port <= 0 || resp.UDP.Port > 65535 {
			return false, fmt.Sprintf("invalid UDP port returned by server: %d", resp.UDP.Port), mqttConnectMs
		}
		// test UDP connection
		udpOK, udpMsg, udpMs := testUDPConnection(resp.UDP)
		totalMs := mqttConnectMs + udpMs
		if udpOK {
			return true, fmt.Sprintf("MQTT(%dms) and UDP(%dms) both OK", mqttConnectMs, udpMs), totalMs
		} else {
			return false, "MQTT OK but UDP failed: " + udpMsg, totalMs
		}
	case err := <-errChan:
		return false, err.Error(), mqttConnectMs
	case <-ctx.Done():
		return false, fmt.Sprintf("timed out waiting for hello response (5s), hello sent to %s", mqttConfig.PublishTopic), mqttConnectMs
	}
}

// testUDPConnection tests the UDP connection
func testUDPConnection(udpConfig UDPConfig) (bool, string, int64) {
	t0 := time.Now()

	// validate UDP config
	if udpConfig.Server == "" {
		return false, "UDP server address is empty", 0
	}
	if udpConfig.Port <= 0 || udpConfig.Port > 65535 {
		return false, fmt.Sprintf("invalid UDP port: %d", udpConfig.Port), 0
	}

	// parse UDP address
	udpAddr := fmt.Sprintf("%s:%d", udpConfig.Server, udpConfig.Port)
	addr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		return false, "failed to parse UDP address (" + udpAddr + "): " + err.Error(), 0
	}

	// create UDP connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return false, fmt.Sprintf("UDP server refused connection (%s), check if UDP server is running", udpAddr), time.Since(t0).Milliseconds()
		} else if strings.Contains(err.Error(), "no route to host") || strings.Contains(err.Error(), "network is unreachable") {
			return false, fmt.Sprintf("cannot route to UDP server (%s), check network connection", udpAddr), time.Since(t0).Milliseconds()
		} else if strings.Contains(err.Error(), "timeout") {
			return false, fmt.Sprintf("UDP connection timed out (%s), check firewall settings", udpAddr), time.Since(t0).Milliseconds()
		}
		return false, "UDP connection failed (" + udpAddr + "): " + err.Error(), time.Since(t0).Milliseconds()
	}
	defer conn.Close()

	// set read/write timeout
	deadline := time.Now().Add(2 * time.Second)
	err = conn.SetReadDeadline(deadline)
	if err != nil {
		return false, "failed to set UDP timeout: " + err.Error(), time.Since(t0).Milliseconds()
	}

	// send test packet (simulating audio data)
	testData := []byte("ping")
	_, err = conn.Write(testData)
	if err != nil {
		return false, "failed to send UDP data: " + err.Error(), time.Since(t0).Milliseconds()
	}

	// try reading response (timeout is also considered success since UDP may not respond)
	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil {
		// UDP read timeout is also success, as sending data proves connection works
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return true, "UDP connection OK (no response, timed out)", time.Since(t0).Milliseconds()
		}
		return false, "UDP read failed: " + err.Error(), time.Since(t0).Milliseconds()
	}

	return true, "UDP connection OK", time.Since(t0).Milliseconds()
}

// testOTAConfig two-stage check: 1) POST OTA address to get websocket.url from JSON; 2) establish WebSocket connection for validation.
// returns ok, message, first_packet_ms, ota_response (OTA API response body for frontend display)
func (ac *AdminController) testOTAConfig(cfg models.Config) (ok bool, message string, firstPacketMs int64, otaResponseBody string) {
	if cfg.JsonData == "" {
		return false, "config is empty", 0, ""
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(cfg.JsonData), &data); err != nil {
		return false, "config parse failed", 0, ""
	}
	var wsURLFromConfig string
	if ext, _ := data["external"].(map[string]interface{}); ext != nil {
		if ws, _ := ext["websocket"].(map[string]interface{}); ws != nil {
			if u, _ := ws["url"].(string); u != "" {
				wsURLFromConfig = u
			}
		}
	}
	if wsURLFromConfig == "" {
		if test, _ := data["test"].(map[string]interface{}); test != nil {
			if ws, _ := test["websocket"].(map[string]interface{}); ws != nil {
				if u, _ := ws["url"].(string); u != "" {
					wsURLFromConfig = u
				}
			}
		}
	}
	if wsURLFromConfig == "" {
		return false, "WebSocket URL not configured", 0, ""
	}
	parsed, err := url.Parse(wsURLFromConfig)
	if err != nil {
		return false, "URL parse failed", 0, ""
	}
	scheme := "http"
	if parsed.Scheme == "wss" {
		scheme = "https"
	}
	otaHTTPURL := scheme + "://" + parsed.Host + otaHTTPPath

	t0 := time.Now()
	// Part1: POST OTA address with Device-ID, Client-ID, parse JSON to get websocket.url
	req, err := http.NewRequest(http.MethodPost, otaHTTPURL, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return false, "failed to create OTA request", time.Since(t0).Milliseconds(), ""
	}
	req.Header.Set("Device-ID", otaTestDeviceID)
	req.Header.Set("Client-ID", otaTestClientID)
	req.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return false, "OTA request failed: " + err.Error(), time.Since(t0).Milliseconds(), ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	firstPacketMs = time.Since(t0).Milliseconds()
	otaResponseBody = string(body)
	if resp.StatusCode != http.StatusOK {
		return false, "OTA returned HTTP " + strconv.Itoa(resp.StatusCode), firstPacketMs, otaResponseBody
	}
	var otaResp map[string]interface{}
	if err := json.Unmarshal(body, &otaResp); err != nil {
		return false, "OTA response is not JSON", firstPacketMs, otaResponseBody
	}
	wsObj, _ := otaResp["websocket"].(map[string]interface{})
	if wsObj == nil {
		return false, "OTA response missing websocket field", firstPacketMs, otaResponseBody
	}
	wsURL, _ := wsObj["url"].(string)
	if wsURL == "" {
		return false, "OTA response missing websocket.url", firstPacketMs, otaResponseBody
	}

	// Part2: establish WebSocket connection with Device-ID, Client-ID, close immediately (connect time counted as first packet)
	wsT0 := time.Now()
	header := http.Header{}
	header.Set("Device-ID", otaTestDeviceID)
	header.Set("Client-ID", otaTestClientID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return false, "WebSocket connection failed: " + err.Error(), firstPacketMs + time.Since(wsT0).Milliseconds(), otaResponseBody
	}
	conn.Close()
	wsTotalMs := firstPacketMs + time.Since(wsT0).Milliseconds()
	return true, "OTA and WebSocket both OK", wsTotalMs, otaResponseBody
}

// testOTAConfigWithMQTTUDP extended OTA test supporting both WebSocket and MQTT UDP tests
// returns the complete test result structure
func (ac *AdminController) testOTAConfigWithMQTTUDP(cfg models.Config) OTATestResult {
	result := OTATestResult{
		WebSocket: OTATestItem{Ok: false, Message: "test failed", FirstPacketMs: 0},
	}

	// parse config
	if cfg.JsonData == "" {
		result.WebSocket = OTATestItem{Ok: false, Message: "config is empty", FirstPacketMs: 0}
		return result
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(cfg.JsonData), &data); err != nil {
		result.WebSocket = OTATestItem{Ok: false, Message: "config parse failed", FirstPacketMs: 0}
		return result
	}

	// get WebSocket URL (prefer external, fall back to test if empty)
	wsURLFromConfig := ""
	if ext, _ := data["external"].(map[string]interface{}); ext != nil {
		if ws, _ := ext["websocket"].(map[string]interface{}); ws != nil {
			wsURLFromConfig, _ = ws["url"].(string)
		}
	}
	if wsURLFromConfig == "" {
		if test, _ := data["test"].(map[string]interface{}); test != nil {
			if ws, _ := test["websocket"].(map[string]interface{}); ws != nil {
				wsURLFromConfig, _ = ws["url"].(string)
			}
		}
	}
	if wsURLFromConfig == "" {
		result.WebSocket = OTATestItem{Ok: false, Message: "WebSocket URL not configured", FirstPacketMs: 0}
		return result
	}

	// determine which environment config to use (based on WebSocket URL source)
	var envConfig map[string]interface{}
	if ext, _ := data["external"].(map[string]interface{}); ext != nil {
		if ws, _ := ext["websocket"].(map[string]interface{}); ws != nil {
			if url, _ := ws["url"].(string); url == wsURLFromConfig && url != "" {
				envConfig = ext
			}
		}
	}
	if envConfig == nil {
		if test, _ := data["test"].(map[string]interface{}); test != nil {
			if ws, _ := test["websocket"].(map[string]interface{}); ws != nil {
				if url, _ := ws["url"].(string); url == wsURLFromConfig {
					envConfig = test
				}
			}
		}
	}

	// check if MQTT UDP test is enabled
	var mqttEnabled bool
	if envConfig != nil {
		if mqtt, _ := envConfig["mqtt"].(map[string]interface{}); mqtt != nil {
			if enable, ok := mqtt["enable"].(bool); ok && enable {
				mqttEnabled = true
			}
		}
	}

	// build OTA HTTP URL
	parsed, err := url.Parse(wsURLFromConfig)
	if err != nil {
		result.WebSocket = OTATestItem{Ok: false, Message: "URL parse failed", FirstPacketMs: 0}
		return result
	}
	scheme := "http"
	if parsed.Scheme == "wss" {
		scheme = "https"
	}
	otaHTTPURL := scheme + "://" + parsed.Host + otaHTTPPath

	// stage 1: POST OTA HTTP endpoint
	t0 := time.Now()
	req, err := http.NewRequest(http.MethodPost, otaHTTPURL, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		result.WebSocket = OTATestItem{Ok: false, Message: "failed to create OTA request", FirstPacketMs: time.Since(t0).Milliseconds()}
		return result
	}
	req.Header.Set("Device-ID", otaTestDeviceID)
	req.Header.Set("Client-ID", otaTestClientID)
	req.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		result.WebSocket = OTATestItem{Ok: false, Message: "OTA request failed: " + err.Error(), FirstPacketMs: time.Since(t0).Milliseconds()}
		return result
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	httpMs := time.Since(t0).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		result.WebSocket = OTATestItem{Ok: false, Message: "OTA returned HTTP " + strconv.Itoa(resp.StatusCode), FirstPacketMs: httpMs}
		return result
	}

	var otaResp map[string]interface{}
	if err := json.Unmarshal(body, &otaResp); err != nil {
		result.WebSocket = OTATestItem{Ok: false, Message: "OTA response is not JSON", FirstPacketMs: httpMs}
		return result
	}

	// stage 2: WebSocket test
	wsObj, _ := otaResp["websocket"].(map[string]interface{})
	if wsObj == nil {
		result.WebSocket = OTATestItem{Ok: false, Message: "OTA response missing websocket field", FirstPacketMs: httpMs}
		return result
	}
	wsURL, _ := wsObj["url"].(string)
	if wsURL == "" {
		result.WebSocket = OTATestItem{Ok: false, Message: "OTA response missing websocket.url", FirstPacketMs: httpMs}
		return result
	}

	wsT0 := time.Now()
	header := http.Header{}
	header.Set("Device-ID", otaTestDeviceID)
	header.Set("Client-ID", otaTestClientID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		result.WebSocket = OTATestItem{Ok: false, Message: "WebSocket connection failed: " + err.Error(), FirstPacketMs: httpMs + time.Since(wsT0).Milliseconds()}
		return result
	}
	conn.Close()
	wsTotalMs := httpMs + time.Since(wsT0).Milliseconds()
	result.WebSocket = OTATestItem{Ok: true, Message: "WebSocket connection OK", FirstPacketMs: wsTotalMs}

	// save OTA response body (for frontend display)
	result.OTAResponse = string(body)

	// stage 3: MQTT UDP test (if enabled)
	// follows test/mqtt_udp logic: get MQTT config from OTA response, send hello, wait for response, test UDP
	if mqttEnabled {
		// get MQTT config from OTA response
		mqttObj, hasMQTT := otaResp["mqtt"].(map[string]interface{})
		if !hasMQTT {
			result.MQTTUDP = &OTATestItem{
				Ok:            false,
				Message:       "OTA response did not return MQTT config, cannot test MQTT UDP",
				FirstPacketMs: 0,
			}
			return result
		}

		// parse MQTT config fields
		endpoint, _ := mqttObj["endpoint"].(string)
		clientID, _ := mqttObj["client_id"].(string)
		username, _ := mqttObj["username"].(string)
		password, _ := mqttObj["password"].(string)
		publishTopic, _ := mqttObj["publish_topic"].(string)
		subscribeTopic, _ := mqttObj["subscribe_topic"].(string)

		// validate required fields (subscribe_topic not required)
		if endpoint == "" {
			result.MQTTUDP = &OTATestItem{Ok: false, Message: "OTA response has empty MQTT endpoint", FirstPacketMs: 0}
			return result
		}
		if publishTopic == "" {
			result.MQTTUDP = &OTATestItem{Ok: false, Message: "OTA response has empty MQTT publish_topic", FirstPacketMs: 0}
			return result
		}

		// build MQTT test config
		otaMqttConfig := &MQTTUDPTestConfig{
			Endpoint:       endpoint,
			ClientID:       clientID,
			Username:       username,
			Password:       password,
			PublishTopic:   publishTopic,
			SubscribeTopic: subscribeTopic, // retained but not validated, may be used for logging
		}

		mqttOK, mqttMsg, mqttMs := testMQTTUDPConfig(*otaMqttConfig)
		result.MQTTUDP = &OTATestItem{
			Ok:            mqttOK,
			Message:       mqttMsg,
			FirstPacketMs: mqttMs,
		}
	}

	return result
}

// generateMQTTUsername generates the MQTT username
func generateMQTTUsername(deviceID, signatureKey string) string {
	h := hmac.New(sha256.New, []byte(signatureKey))
	h.Write([]byte(deviceID + "-username"))
	return hex.EncodeToString(h.Sum(nil))
}

// generateMQTTPassword generates the MQTT password
func generateMQTTPassword(deviceID, signatureKey string) string {
	h := hmac.New(sha256.New, []byte(signatureKey))
	h.Write([]byte(deviceID + "-password"))
	return hex.EncodeToString(h.Sum(nil))
}

// GetConfigs returns all config list
func (ac *AdminController) GetConfigs(c *gin.Context) {
	var configs []models.Config
	if err := ac.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config list"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

// GetConfig returns a single config
func (ac *AdminController) GetConfig(c *gin.Context) {
	id := c.Param("id")
	var config models.Config
	if err := ac.DB.First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Config not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config"})
		}
		return
	}
	c.JSON(http.StatusOK, config)
}

func (ac *AdminController) GetConfigByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var config models.Config

	if err := ac.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": config})
}

func (ac *AdminController) CreateConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// check if Memory config already exists
	var existingCount int64
	ac.DB.Model(&models.Config{}).Where("type = ?", "memory").Count(&existingCount)

	// if no Memory config exists, automatically set as default
	if existingCount == 0 {
		config.IsDefault = true
	}

	// if setting as default, first unset other default configs of the same type
	if config.IsDefault {
		ac.DB.Model(&models.Config{}).Where("type = ? AND is_default = ?", config.Type, true).Update("is_default", false)
	}

	if err := ac.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create config"})
		return
	}

	ac.notifySystemConfigChanged()
	c.JSON(http.StatusCreated, gin.H{"data": config})
}

func (ac *AdminController) UpdateConfig(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var config models.Config

	if err := ac.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	var updateData models.Config
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// if setting as default, first unset other default configs of the same type
	if updateData.IsDefault {
		ac.DB.Model(&models.Config{}).Where("type = ? AND is_default = ? AND id != ?", config.Type, true, id).Update("is_default", false)
	}

	// update config
	config.Name = updateData.Name
	config.Provider = updateData.Provider
	config.JsonData = updateData.JsonData
	config.Enabled = updateData.Enabled
	config.IsDefault = updateData.IsDefault

	if err := ac.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config"})
		return
	}

	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{"data": config})
}

func (ac *AdminController) DeleteConfig(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := ac.DB.Delete(&models.Config{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete config"})
		return
	}
	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// set default config
func (ac *AdminController) SetDefaultConfig(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var config models.Config

	if err := ac.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	// first unset other default configs of the same type
	ac.DB.Model(&models.Config{}).Where("type = ? AND is_default = ?", config.Type, true).Update("is_default", false)

	// set current config as default
	config.IsDefault = true
	if err := ac.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set default config"})
		return
	}

	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{"message": "default config set successfully", "data": config})
}

// get default config
func (ac *AdminController) GetDefaultConfig(c *gin.Context) {
	configType := c.Param("type")
	var config models.Config

	if err := ac.DB.Where("type = ? AND is_default = ?", configType, true).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "default config not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": config})
}

// GlobalRole management
func (ac *AdminController) GetGlobalRoles(c *gin.Context) {
	var roles []models.GlobalRole
	if err := ac.DB.Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get global roles"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": roles})
}

func (ac *AdminController) CreateGlobalRole(c *gin.Context) {
	var role models.GlobalRole
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ac.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create global role"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": role})
}

func (ac *AdminController) UpdateGlobalRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var role models.GlobalRole

	if err := ac.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "global role not found"})
		return
	}

	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ac.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update global role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": role})
}

func (ac *AdminController) DeleteGlobalRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := ac.DB.Delete(&models.GlobalRole{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete global role"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// user management
func (ac *AdminController) GetUserOptions(c *gin.Context) {
	var users []struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	}
	if err := ac.DB.Model(&models.User{}).Select("id, username").Order("username ASC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user options"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (ac *AdminController) GetUsers(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	offset := (page - 1) * pageSize

	var total int64
	if err := ac.DB.Model(&models.User{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user list"})
		return
	}

	var users []models.User
	if err := ac.DB.Order("id DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user list"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "total": total, "page": page, "page_size": pageSize})
}

func (ac *AdminController) CreateUser(c *gin.Context) {
	// add debug marker
	log.Println("=== [CreateUser] method started ===")
	log.Println("=== [CreateUser] start of CreateUser method ===")

	// since User model's Password field uses json:"-" tag, manual parsing is required
	var requestData struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	// bind directly to map to inspect raw data
	var rawMap map[string]interface{}
	if err := c.ShouldBindJSON(&rawMap); err != nil {
		log.Printf("[CreateUser] failed to bind to map: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON parse failed"})
		return
	}
	log.Printf("[CreateUser] raw JSON data: %+v", rawMap)

	// manually extract fields
	username, _ := rawMap["username"].(string)
	email, _ := rawMap["email"].(string)
	password, _ := rawMap["password"].(string)
	role, _ := rawMap["role"].(string)

	// update requestData
	requestData.Username = username
	requestData.Email = email
	requestData.Password = password
	requestData.Role = role

	// validate required fields
	if requestData.Username == "" || requestData.Email == "" || requestData.Password == "" {
		log.Printf("[CreateUser] missing required fields: username=%s, email=%s, password length=%d",
			requestData.Username, requestData.Email, len(requestData.Password))
		c.JSON(http.StatusBadRequest, gin.H{"error": "username, email, and password are required"})
		return
	}

	log.Printf("[CreateUser] received user creation request - username: %s, email: %s, role: %s", requestData.Username, requestData.Email, requestData.Role)
	log.Printf("[CreateUser] raw password length: %d", len(requestData.Password))

	// check if username already exists
	var existingUser models.User
	err := ac.DB.Where("username = ?", requestData.Username).First(&existingUser).Error
	if err == nil {
		// username already exists
		log.Printf("[CreateUser] username %s already exists", requestData.Username)
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// database query error
		log.Printf("[CreateUser] database query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// user not found, create new user
	log.Printf("[CreateUser] creating new user: %s", requestData.Username)
	var user models.User
	user.Username = requestData.Username
	user.Email = requestData.Email
	user.Role = requestData.Role

	// encrypt password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[CreateUser] password encryption failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "password encryption failed"})
		return
	}
	user.Password = string(hashedPassword)
	log.Printf("[CreateUser] password encrypted - hash length: %d, hash prefix: %s", len(user.Password), user.Password[:10])

	if err := ac.DB.Create(&user).Error; err != nil {
		log.Printf("[CreateUser] database failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	log.Printf("[CreateUser] user created - ID: %d, username: %s", user.ID, user.Username)

	// do not return password
	user.Password = ""
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

func (ac *AdminController) UpdateUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user models.User

	if err := ac.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// if updating password, encrypt it
	if password, ok := updateData["password"]; ok && password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password.(string)), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "password encryption failed"})
			return
		}
		updateData["password"] = string(hashedPassword)
	}

	if err := ac.DB.Model(&user).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	// re-query user info (without password)
	ac.DB.First(&user, id)
	user.Password = ""
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func (ac *AdminController) DeleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := ac.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// reset user password
func (ac *AdminController) ResetUserPassword(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var requestData struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "please enter a valid new password (at least 6 characters)"})
		return
	}

	// find user
	var user models.User
	if err := ac.DB.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find user"})
		}
		return
	}

	// encrypt new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[ResetUserPassword] password encryption failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "password encryption failed"})
		return
	}

	// update user password
	if err := ac.DB.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		log.Printf("[ResetUserPassword] failed to update password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}

	log.Printf("[ResetUserPassword] admin reset user password - userID: %d, username: %s", user.ID, user.Username)
	c.JSON(http.StatusOK, gin.H{
		"message": "password reset successfully",
		"data": gin.H{
			"user_id":  user.ID,
			"username": user.Username,
		},
	})
}

// GetUserVoiceCloneQuotas returns user voice clone quotas (by tts_config_id)
func (ac *AdminController) GetUserVoiceCloneQuotas(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	var user models.User
	if err = ac.DB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		return
	}
	if strings.TrimSpace(user.Role) != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voice clone quotas can only be assigned to regular users"})
		return
	}

	var ttsConfigs []models.Config
	if err = ac.DB.Where("type = ?", "tts").Order("enabled DESC, name ASC").Find(&ttsConfigs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query TTS config"})
		return
	}

	var quotas []models.UserVoiceCloneQuota
	if err = ac.DB.Where("user_id = ?", user.ID).Find(&quotas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user quota"})
		return
	}
	quotaByConfigID := make(map[string]models.UserVoiceCloneQuota, len(quotas))
	for _, quota := range quotas {
		quotaByConfigID[quota.TTSConfigID] = quota
	}

	type usageRow struct {
		TTSConfigID string `json:"tts_config_id"`
		UsedCount   int64  `json:"used_count"`
	}
	var usageRows []usageRow
	if err = ac.DB.Model(&models.VoiceClone{}).
		Select("tts_config_id, COUNT(1) AS used_count").
		Where("user_id = ? AND status != ?", user.ID, "deleted").
		Group("tts_config_id").
		Scan(&usageRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count user clone usage"})
		return
	}
	usageByConfigID := make(map[string]int, len(usageRows))
	for _, row := range usageRows {
		usageByConfigID[row.TTSConfigID] = int(row.UsedCount)
	}

	result := make([]gin.H, 0, len(ttsConfigs))
	configIDSet := make(map[string]bool, len(ttsConfigs))
	for _, ttsConfig := range ttsConfigs {
		configIDSet[ttsConfig.ConfigID] = true
		quota, hasQuota := quotaByConfigID[ttsConfig.ConfigID]
		maxCount := 0
		usedCount := usageByConfigID[ttsConfig.ConfigID]
		if hasQuota {
			maxCount = quota.MaxCount
			if quota.UsedCount > usedCount {
				usedCount = quota.UsedCount
			}
		}
		remainingCount := -1
		if maxCount >= 0 {
			remainingCount = maxCount - usedCount
			if remainingCount < 0 {
				remainingCount = 0
			}
		}

		result = append(result, gin.H{
			"tts_config_id":   ttsConfig.ConfigID,
			"tts_config_name": ttsConfig.Name,
			"provider":        ttsConfig.Provider,
			"enabled":         ttsConfig.Enabled,
			"max_count":       maxCount,
			"used_count":      usedCount,
			"remaining_count": remainingCount,
		})
	}

	// retain quotas for deleted historical configs to avoid "quota config invisible after deletion"
	for _, quota := range quotas {
		if configIDSet[quota.TTSConfigID] {
			continue
		}
		maxCount := quota.MaxCount
		usedCount := quota.UsedCount
		if usageByConfigID[quota.TTSConfigID] > usedCount {
			usedCount = usageByConfigID[quota.TTSConfigID]
		}
		remainingCount := -1
		if maxCount >= 0 {
			remainingCount = maxCount - usedCount
			if remainingCount < 0 {
				remainingCount = 0
			}
		}
		result = append(result, gin.H{
			"tts_config_id":   quota.TTSConfigID,
			"tts_config_name": "(deleted config)",
			"provider":        "",
			"enabled":         false,
			"max_count":       maxCount,
			"used_count":      usedCount,
			"remaining_count": remainingCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"user_id":    user.ID,
		"username":   user.Username,
		"quotas":     result,
		"updated_at": time.Now(),
	}})
}

// UpdateUserVoiceCloneQuotas batch updates user voice clone quotas
func (ac *AdminController) UpdateUserVoiceCloneQuotas(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	var user models.User
	if err = ac.DB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		return
	}
	if strings.TrimSpace(user.Role) != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voice clone quotas can only be assigned to regular users"})
		return
	}

	var req struct {
		Items []struct {
			TTSConfigID string `json:"tts_config_id"`
			MaxCount    int    `json:"max_count"`
		} `json:"items"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameter format"})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items cannot be empty"})
		return
	}

	itemByConfigID := make(map[string]int, len(req.Items))
	configIDs := make([]string, 0, len(req.Items))
	for _, item := range req.Items {
		configID := strings.TrimSpace(item.TTSConfigID)
		if configID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tts_config_id cannot be empty"})
			return
		}
		if item.MaxCount < -1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "max_count cannot be less than -1"})
			return
		}
		if _, exists := itemByConfigID[configID]; !exists {
			configIDs = append(configIDs, configID)
		}
		itemByConfigID[configID] = item.MaxCount
	}

	var ttsConfigs []models.Config
	if err = ac.DB.Where("type = ? AND config_id IN ?", "tts", configIDs).Find(&ttsConfigs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query TTS config"})
		return
	}
	validConfigIDSet := make(map[string]bool, len(ttsConfigs))
	for _, cfg := range ttsConfigs {
		validConfigIDSet[cfg.ConfigID] = true
	}
	for _, configID := range configIDs {
		if validConfigIDSet[configID] {
			continue
		}
		// deleted historical configs may only be set to -1 (delete quota record)
		if itemByConfigID[configID] == -1 {
			continue
		}
		if !validConfigIDSet[configID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("TTS config not found: %s", configID)})
			return
		}
	}

	type usageRow struct {
		TTSConfigID string `json:"tts_config_id"`
		UsedCount   int64  `json:"used_count"`
	}
	var usageRows []usageRow
	if err = ac.DB.Model(&models.VoiceClone{}).
		Select("tts_config_id, COUNT(1) AS used_count").
		Where("user_id = ? AND status != ? AND tts_config_id IN ?", user.ID, "deleted", configIDs).
		Group("tts_config_id").
		Scan(&usageRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count user usage"})
		return
	}
	usageByConfigID := make(map[string]int, len(usageRows))
	for _, row := range usageRows {
		usageByConfigID[row.TTSConfigID] = int(row.UsedCount)
	}

	if err = ac.DB.Transaction(func(tx *gorm.DB) error {
		for _, configID := range configIDs {
			maxCount := itemByConfigID[configID]
			if maxCount == -1 {
				if err := tx.Where("user_id = ? AND tts_config_id = ?", user.ID, configID).Delete(&models.UserVoiceCloneQuota{}).Error; err != nil {
					return err
				}
				continue
			}

			usedCount := usageByConfigID[configID]
			var quota models.UserVoiceCloneQuota
			if err := tx.Where("user_id = ? AND tts_config_id = ?", user.ID, configID).First(&quota).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					newQuota := models.UserVoiceCloneQuota{
						UserID:      user.ID,
						TTSConfigID: configID,
						MaxCount:    maxCount,
						UsedCount:   usedCount,
					}
					if err := tx.Create(&newQuota).Error; err != nil {
						return err
					}
					continue
				}
				return err
			}

			nextUsedCount := quota.UsedCount
			if usedCount > nextUsedCount {
				nextUsedCount = usedCount
			}
			if err := tx.Model(&models.UserVoiceCloneQuota{}).Where("id = ?", quota.ID).Updates(map[string]any{
				"max_count":  maxCount,
				"used_count": nextUsedCount,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user voice clone quota"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "quota updated successfully"})
}

// GetUserVoiceOptionsAdmin returns available voices for a specific user, used by admins when creating/editing agents.
func (ac *AdminController) GetUserVoiceOptionsAdmin(c *gin.Context) {
	userID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	voices, err := getVoiceOptionsForUser(
		ac.DB,
		c,
		userID,
		c.Query("provider"),
		c.Query("config_id"),
		c.Query("api_url"),
		c.Query("api_key"),
	)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "IndexTTS") {
			status = http.StatusBadGateway
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": voices})
}

// GetUserVoiceClonesAdmin returns cloned voices for a specific user, used by admins when creating/editing agents.
func (ac *AdminController) GetUserVoiceClonesAdmin(c *gin.Context) {
	userID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	clones, err := getVoiceClonesForUser(ac.DB, userID, c.Query("tts_config_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get cloned voices"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": clones})
}

// device management
func (ac *AdminController) GetDevices(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	devices, total, err := NewDeviceService(ac.DB).List(scopeFromContext(c), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get device list"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": devices, "total": total, "page": page, "page_size": pageSize})
}

// validate device activation code
func (ac *AdminController) ValidateDeviceCode(c *gin.Context) {
	deviceCode := c.Query("code")
	if deviceCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "activation code cannot be empty"})
		return
	}

	var device models.Device
	err := ac.DB.Where("device_code = ?", deviceCode).First(&device).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, gin.H{"exists": false})
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query device"})
	} else {
		c.JSON(http.StatusOK, gin.H{"exists": true, "device": device})
	}
}

func (ac *AdminController) CreateDevice(c *gin.Context) {
	var req DevicePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}
	device, err := NewDeviceService(ac.DB).Create(scopeFromContext(c), req)
	if err != nil {
		writeServiceError(c, err, "failed to create device")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "device created successfully",
		"data":    device,
	})
}

func (ac *AdminController) UpdateDevice(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req DevicePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	device, err := NewDeviceService(ac.DB).Update(scopeFromContext(c), id, req)
	if err != nil {
		writeServiceError(c, err, "failed to update device")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": device})
}

func (ac *AdminController) DeleteDevice(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := NewDeviceService(ac.DB).Delete(scopeFromContext(c), id); err != nil {
		writeServiceError(c, err, "failed to delete device")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// agent management
func (ac *AdminController) GetAgents(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	result, total, err := NewAgentService(ac.DB).List(scopeFromContext(c), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent list"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "total": total, "page": page, "page_size": pageSize})
}

// GetDeviceMcpTools returns device-level MCP tool list (admin version)
func (ac *AdminController) GetDeviceMcpTools(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id parameter is required"})
		return
	}

	var device models.Device
	if err := ac.DB.Where("id = ?", deviceID).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	tools, err := ac.WebSocketController.RequestDeviceMcpToolDetailsFromClient(context.Background(), device.DeviceName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"tools": []interface{}{}}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"tools": tools}})
}

// CallAgentMcpTool calls agent-level MCP tool (admin version)
func (ac *AdminController) CallAgentMcpTool(c *gin.Context) {
	agentID := c.Param("id")
	var req struct {
		ToolName  string                 `json:"tool_name" binding:"required"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}

	var agent models.Agent
	if err := ac.DB.Where("id = ?", agentID).First(&agent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	body := map[string]interface{}{
		"agent_id":  agentID,
		"tool_name": req.ToolName,
		"arguments": req.Arguments,
	}
	result, err := ac.WebSocketController.CallMcpToolFromClient(context.Background(), body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to call MCP tool: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// CallDeviceMcpTool calls device-level MCP tool (admin version)
func (ac *AdminController) CallDeviceMcpTool(c *gin.Context) {
	deviceID := c.Param("id")
	var req struct {
		ToolName  string                 `json:"tool_name" binding:"required"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}

	var device models.Device
	if err := ac.DB.Where("id = ?", deviceID).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	body := map[string]interface{}{
		"device_id": device.DeviceName,
		"tool_name": req.ToolName,
		"arguments": req.Arguments,
	}
	result, err := ac.WebSocketController.CallMcpToolFromClient(context.Background(), body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to call MCP tool: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetAgentMCPEndpoint returns the MCP endpoint URL for an agent
func (ac *AdminController) GetAgentMCPEndpoint(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id parameter is required"})
		return
	}

	// get current user ID from JWT middleware
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID, ok := userIDInterface.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID type"})
		return
	}

	// use common function to generate MCP endpoint
	endpoint, err := GenerateAgentMCPEndpoint(ac.DB, agentID, userID, ac.EndpointAuthToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	data := newMcpEndpointData(endpoint)
	if ac.WebSocketController == nil {
		data["status_message"] = "websocket controller unavailable"
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}

	statusResult, statusErr := ac.WebSocketController.RequestMcpEndpointStatusFromClient(context.Background(), agentID)
	if statusErr != nil {
		data["status_message"] = statusErr.Error()
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}

	applyMcpEndpointStatus(data, statusResult)
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// GetAgentOpenClawEndpoint returns the OpenClaw endpoint URL for an agent
func (ac *AdminController) GetAgentOpenClawEndpoint(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id parameter is required"})
		return
	}

	// get current user ID from JWT middleware
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID, ok := userIDInterface.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID type"})
		return
	}

	data := gin.H{
		"endpoint":  "",
		"status":    "unknown",
		"connected": false,
	}

	endpoint, err := GenerateAgentOpenClawEndpoint(ac.DB, agentID, userID, ac.EndpointAuthToken)
	if err != nil {
		data["status_message"] = err.Error()
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}
	data["endpoint"] = endpoint

	if ac.WebSocketController == nil {
		data["status_message"] = "websocket controller unavailable"
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}

	statusResult, statusErr := ac.WebSocketController.RequestOpenClawStatusFromClient(context.Background(), agentID)
	if statusErr != nil {
		data["status_message"] = statusErr.Error()
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}

	connected, _ := statusResult["connected"].(bool)
	status, _ := statusResult["status"].(string)
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		if connected {
			status = "online"
		} else {
			status = "offline"
		}
	}

	data["connected"] = connected
	data["status"] = status
	if msg, ok := statusResult["status_message"].(string); ok && strings.TrimSpace(msg) != "" {
		data["status_message"] = msg
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// CallAgentOpenClawChatTest calls agent OpenClaw chat test (admin version)
func (ac *AdminController) CallAgentOpenClawChatTest(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id parameter is required"})
		return
	}
	if ac.WebSocketController == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "websocket controller unavailable"})
		return
	}

	var req struct {
		Message   string `json:"message" binding:"required"`
		TimeoutMs int    `json:"timeout_ms"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message cannot be empty"})
		return
	}

	var agent models.Agent
	if err := ac.DB.Where("id = ?", agentID).First(&agent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	body := map[string]interface{}{
		"agent_id": agentID,
		"message":  req.Message,
	}
	if req.TimeoutMs > 0 {
		body["timeout_ms"] = req.TimeoutMs
	}

	if wantsOpenClawSSE(c) {
		if !prepareOpenClawSSE(c) {
			return
		}
		_ = writeOpenClawSSE(c, "start", map[string]interface{}{
			"agent_id": agentID,
		})

		terminalErrorSent := false
		result, err := ac.WebSocketController.CallOpenClawChatStreamFromClient(
			c.Request.Context(),
			body,
			func(resp *WebSocketResponse) error {
				if resp == nil {
					return nil
				}
				payload := map[string]interface{}{
					"status": resp.Status,
				}
				if resp.Body != nil {
					payload["data"] = resp.Body
				}
				if msg := strings.TrimSpace(resp.Error); msg != "" {
					payload["error"] = msg
				}

				switch resp.Status {
				case http.StatusPartialContent:
					return writeOpenClawSSE(c, "chunk", payload)
				case http.StatusOK:
					return writeOpenClawSSE(c, "result", payload)
				default:
					terminalErrorSent = true
					return writeOpenClawSSE(c, "error", payload)
				}
			},
		)
		if err != nil {
			if !terminalErrorSent {
				_ = writeOpenClawSSE(c, "error", map[string]interface{}{
					"error": err.Error(),
				})
			}
			_ = writeOpenClawSSE(c, "done", map[string]interface{}{
				"ok": false,
			})
			return
		}

		_ = writeOpenClawSSE(c, "done", map[string]interface{}{
			"ok":   true,
			"data": result,
		})
		return
	}

	result, err := ac.WebSocketController.CallOpenClawChatFromClient(context.Background(), body)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(strings.ToLower(msg), "not connected"), strings.Contains(msg, "not connected"):
			c.JSON(http.StatusConflict, gin.H{"error": msg})
		case strings.Contains(strings.ToLower(msg), "timeout"), strings.Contains(msg, "timeout"):
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": msg})
		case strings.Contains(strings.ToLower(msg), "missing"), strings.Contains(msg, "parameter"):
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		case strings.Contains(msg, "no connected clients"):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": msg})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to call OpenClaw chat test: " + msg})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetAgentMcpTools returns MCP tool list for an agent
func (ac *AdminController) GetAgentMcpTools(c *gin.Context) {
	agentID := c.Param("id")

	// admin validation: check if agent exists (admins can view any user's agent)
	adminAgentValidator := func(agentID string) error {
		var agent models.Agent
		if err := ac.DB.Where("id = ?", agentID).First(&agent).Error; err != nil {
			return fmt.Errorf("agent not found")
		}
		return nil
	}

	// use common function
	GetAgentMcpToolsCommon(c, agentID, ac.WebSocketController, adminAgentValidator)
}

func (ac *AdminController) CreateAgent(c *gin.Context) {
	var req AgentPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	agent, err := NewAgentService(ac.DB).Create(scopeFromContext(c), req)
	if err != nil {
		writeServiceError(c, err, "failed to create agent")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": agent})
}

func (ac *AdminController) UpdateAgent(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req AgentPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	agent, err := NewAgentService(ac.DB).Update(scopeFromContext(c), id, req)
	if err != nil {
		writeServiceError(c, err, "failed to update agent")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": agent})
}

func (ac *AdminController) DeleteAgent(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := NewAgentService(ac.DB).Delete(scopeFromContext(c), id); err != nil {
		writeServiceError(c, err, "failed to delete agent")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// VAD config management (frontend compatible)
func (ac *AdminController) GetVADConfigs(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	offset := (page - 1) * pageSize
	var total int64
	if err := ac.DB.Model(&models.Config{}).Where("type = ?", "vad").Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get VAD configs"})
		return
	}
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "vad").Order("id DESC").Offset(offset).Limit(pageSize).Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get VAD configs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs, "total": total, "page": page, "page_size": pageSize})
}

func (ac *AdminController) CreateVADConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Type = "vad"
	ac.createConfigWithType(c, &config)
}

func (ac *AdminController) UpdateVADConfig(c *gin.Context) {
	ac.updateConfigWithType(c, "vad")
}

func (ac *AdminController) DeleteVADConfig(c *gin.Context) {
	ac.deleteConfigWithType(c, "vad")
}

// ASR config management (frontend compatible)
func (ac *AdminController) GetASRConfigs(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	offset := (page - 1) * pageSize
	var total int64
	if err := ac.DB.Model(&models.Config{}).Where("type = ?", "asr").Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get ASR configs"})
		return
	}
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "asr").Order("id DESC").Offset(offset).Limit(pageSize).Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get ASR configs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs, "total": total, "page": page, "page_size": pageSize})
}

func (ac *AdminController) CreateASRConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Type = "asr"
	ac.createConfigWithType(c, &config)
}

func (ac *AdminController) UpdateASRConfig(c *gin.Context) {
	ac.updateConfigWithType(c, "asr")
}

func (ac *AdminController) DeleteASRConfig(c *gin.Context) {
	ac.deleteConfigWithType(c, "asr")
}

// LLM config management (frontend compatible)
func (ac *AdminController) GetLLMConfigs(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	offset := (page - 1) * pageSize
	var total int64
	if err := ac.DB.Model(&models.Config{}).Where("type = ?", "llm").Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get LLM configs"})
		return
	}
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "llm").Order("id DESC").Offset(offset).Limit(pageSize).Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get LLM configs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs, "total": total, "page": page, "page_size": pageSize})
}

func (ac *AdminController) CreateLLMConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Type = "llm"
	ac.createConfigWithType(c, &config)
}

func (ac *AdminController) UpdateLLMConfig(c *gin.Context) {
	ac.updateConfigWithType(c, "llm")
}

func (ac *AdminController) DeleteLLMConfig(c *gin.Context) {
	ac.deleteConfigWithType(c, "llm")
}

// TTS config management (frontend compatible)
func (ac *AdminController) GetTTSConfigs(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	offset := (page - 1) * pageSize
	var total int64
	if err := ac.DB.Model(&models.Config{}).Where("type = ?", "tts").Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get TTS configs"})
		return
	}
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "tts").Order("id DESC").Offset(offset).Limit(pageSize).Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get TTS configs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs, "total": total, "page": page, "page_size": pageSize})
}

func (ac *AdminController) CreateTTSConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Type = "tts"
	ac.createConfigWithType(c, &config)
}

func (ac *AdminController) UpdateTTSConfig(c *gin.Context) {
	ac.updateConfigWithType(c, "tts")
}

func (ac *AdminController) DeleteTTSConfig(c *gin.Context) {
	ac.deleteConfigWithType(c, "tts")
}

// Speaker config management (frontend compatible)
func (ac *AdminController) GetSpeakerConfigs(c *gin.Context) {
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "voice_identify").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Speaker configs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func (ac *AdminController) CreateSpeakerConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Type = "voice_identify"
	// there is only one voice-print config, automatically set as default
	config.IsDefault = true
	// if config already exists, delete the old one first
	ac.DB.Where("type = ?", "voice_identify").Delete(&models.Config{})
	ac.createConfigWithType(c, &config)
}

func (ac *AdminController) UpdateSpeakerConfig(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var config models.Config

	if err := ac.DB.Where("id = ? AND type = ?", id, "voice_identify").First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	var updateData models.Config
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// there is only one voice-print config, always set as default
	updateData.IsDefault = true

	// update config
	config.Name = updateData.Name
	config.Provider = updateData.Provider
	config.JsonData = updateData.JsonData
	config.Enabled = updateData.Enabled
	config.IsDefault = updateData.IsDefault

	// if a new config_id is provided, update it
	if updateData.ConfigID != "" {
		config.ConfigID = updateData.ConfigID
	}

	if err := ac.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": config})
}

func (ac *AdminController) DeleteSpeakerConfig(c *gin.Context) {
	ac.deleteConfigWithType(c, "voice_identify")
}

// Vision config management (frontend compatible)
func (ac *AdminController) GetVisionConfigs(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	offset := (page - 1) * pageSize
	var total int64
	if err := ac.DB.Model(&models.Config{}).Where("type = ? AND config_id != ?", "vision", "vision_base").Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Vision configs"})
		return
	}
	var configs []models.Config
	if err := ac.DB.Where("type = ? AND config_id != ?", "vision", "vision_base").Order("id DESC").Offset(offset).Limit(pageSize).Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Vision configs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs, "total": total, "page": page, "page_size": pageSize})
}

// GetVisionBaseConfig returns Vision base config
func (ac *AdminController) GetVisionBaseConfig(c *gin.Context) {
	var config models.Config
	if err := ac.DB.Where("type = ? AND config_id = ?", "vision", "vision_base").First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// if no base config found, return default value
			c.JSON(http.StatusOK, gin.H{"data": map[string]interface{}{
				"enable_auth": false,
				"vision_url":  "",
			}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Vision base config"})
		return
	}

	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(config.JsonData), &configData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Vision base config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": configData})
}

// UpdateVisionBaseConfig updates Vision base config
func (ac *AdminController) UpdateVisionBaseConfig(c *gin.Context) {
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal config data"})
		return
	}

	var config models.Config
	if err := ac.DB.Where("type = ? AND config_id = ?", "vision", "vision_base").First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// create new base config
			config = models.Config{
				Type:      "vision",
				Name:      "vision_base",
				ConfigID:  "vision_base",
				Provider:  "vision_base",
				JsonData:  string(jsonData),
				Enabled:   true,
				IsDefault: false,
			}
			if err := ac.DB.Create(&config).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Vision base config"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query Vision base config"})
			return
		}
	} else {
		// update existing config
		config.JsonData = string(jsonData)
		if err := ac.DB.Save(&config).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Vision base config"})
			return
		}
	}

	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{"message": "Vision base config updated successfully"})
}

// GetChatSettings returns chat settings (auth.enable + chat.*)
func (ac *AdminController) GetChatSettings(c *gin.Context) {
	response := gin.H{
		"auth": gin.H{
			"enable":                false,
			"login_captcha_enabled": true,
		},
		"chat": gin.H{
			"max_idle_duration":         30000,
			"chat_max_silence_duration": 400,
			"realtime_mode":             4,
			"global_system_prompt":      "",
		},
	}

	var authConfig models.Config
	if err := ac.DB.Where("type = ?", "auth").Order("is_default DESC, id ASC").First(&authConfig).Error; err == nil {
		var authData map[string]interface{}
		if authConfig.JsonData != "" && json.Unmarshal([]byte(authConfig.JsonData), &authData) == nil {
			if enable, ok := authData["enable"].(bool); ok {
				response["auth"].(gin.H)["enable"] = enable
			}
			if enabled, ok := authData["login_captcha_enabled"].(bool); ok {
				response["auth"].(gin.H)["login_captcha_enabled"] = enabled
			}
		}
	}

	var chatConfig models.Config
	if err := ac.DB.Where("type = ?", "chat").Order("is_default DESC, id ASC").First(&chatConfig).Error; err == nil {
		var chatData map[string]interface{}
		if chatConfig.JsonData != "" && json.Unmarshal([]byte(chatConfig.JsonData), &chatData) == nil {
			if maxIdle, ok := chatData["max_idle_duration"].(float64); ok && int64(maxIdle) >= 0 {
				response["chat"].(gin.H)["max_idle_duration"] = int64(maxIdle)
			}
			if maxSilence, ok := chatData["chat_max_silence_duration"].(float64); ok && int64(maxSilence) >= 0 {
				response["chat"].(gin.H)["chat_max_silence_duration"] = int64(maxSilence)
			}
			if realtimeMode, ok := chatData["realtime_mode"].(float64); ok && int(realtimeMode) >= 1 && int(realtimeMode) <= 4 {
				response["chat"].(gin.H)["realtime_mode"] = int(realtimeMode)
			}
			if globalPrompt, ok := chatData["global_system_prompt"].(string); ok {
				response["chat"].(gin.H)["global_system_prompt"] = strings.TrimSpace(globalPrompt)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// UpdateChatSettings updates chat settings (auth.enable + chat.*)
func (ac *AdminController) UpdateChatSettings(c *gin.Context) {
	var req struct {
		Auth struct {
			Enable              bool  `json:"enable"`
			LoginCaptchaEnabled *bool `json:"login_captcha_enabled"`
		} `json:"auth"`
		Chat struct {
			MaxIdleDuration        int64  `json:"max_idle_duration"`
			ChatMaxSilenceDuration int64  `json:"chat_max_silence_duration"`
			RealtimeMode           int    `json:"realtime_mode"`
			GlobalSystemPrompt     string `json:"global_system_prompt"`
		} `json:"chat"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Chat.MaxIdleDuration < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat.max_idle_duration cannot be less than 0, 0 means unlimited"})
		return
	}
	if req.Chat.ChatMaxSilenceDuration < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat.chat_max_silence_duration cannot be less than 0"})
		return
	}
	if req.Chat.RealtimeMode < 1 || req.Chat.RealtimeMode > 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat.realtime_mode must be between 1 and 4"})
		return
	}
	req.Chat.GlobalSystemPrompt = strings.TrimSpace(req.Chat.GlobalSystemPrompt)
	if len(req.Chat.GlobalSystemPrompt) > 8000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat.global_system_prompt cannot exceed 8000 characters"})
		return
	}

	loginCaptchaEnabled := true
	if req.Auth.LoginCaptchaEnabled != nil {
		loginCaptchaEnabled = *req.Auth.LoginCaptchaEnabled
	}

	authJSON, err := json.Marshal(map[string]interface{}{
		"enable":                req.Auth.Enable,
		"login_captcha_enabled": loginCaptchaEnabled,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auth config serialization failed"})
		return
	}
	chatJSON, err := json.Marshal(map[string]interface{}{
		"max_idle_duration":         req.Chat.MaxIdleDuration,
		"chat_max_silence_duration": req.Chat.ChatMaxSilenceDuration,
		"realtime_mode":             req.Chat.RealtimeMode,
		"global_system_prompt":      req.Chat.GlobalSystemPrompt,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "chat config serialization failed"})
		return
	}

	tx := ac.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	upsertConfig := func(configType, configID, name string, jsonData []byte) error {
		var cfg models.Config
		err := tx.Where("type = ? AND config_id = ?", configType, configID).First(&cfg).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Model(&models.Config{}).Where("type = ?", configType).Update("is_default", false).Error; err != nil {
				return err
			}
			cfg = models.Config{
				Type:      configType,
				Name:      name,
				ConfigID:  configID,
				Provider:  "",
				JsonData:  string(jsonData),
				Enabled:   true,
				IsDefault: true,
			}
			return tx.Create(&cfg).Error
		}
		if err != nil {
			return err
		}

		if err := tx.Model(&models.Config{}).Where("type = ? AND id != ?", configType, cfg.ID).Update("is_default", false).Error; err != nil {
			return err
		}

		cfg.Name = name
		cfg.Provider = ""
		cfg.JsonData = string(jsonData)
		cfg.Enabled = true
		cfg.IsDefault = true
		return tx.Save(&cfg).Error
	}

	if err := upsertConfig("auth", "auth", "auth", authJSON); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save auth settings: " + err.Error()})
		return
	}
	if err := upsertConfig("chat", "chat", "chat", chatJSON); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save chat settings: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{
		"message": "chat settings updated successfully",
		"data": gin.H{
			"auth": gin.H{
				"enable":                req.Auth.Enable,
				"login_captcha_enabled": loginCaptchaEnabled,
			},
			"chat": gin.H{
				"max_idle_duration":         req.Chat.MaxIdleDuration,
				"chat_max_silence_duration": req.Chat.ChatMaxSilenceDuration,
				"realtime_mode":             req.Chat.RealtimeMode,
				"global_system_prompt":      req.Chat.GlobalSystemPrompt,
			},
		},
	})
}

func (ac *AdminController) CreateVisionConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Type = "vision"
	ac.createConfigWithType(c, &config)
}

func (ac *AdminController) UpdateVisionConfig(c *gin.Context) {
	ac.updateConfigWithType(c, "vision")
}

func (ac *AdminController) DeleteVisionConfig(c *gin.Context) {
	ac.deleteConfigWithType(c, "vision")
}

// OTA config management (frontend compatible)
func (ac *AdminController) GetOTAConfigs(c *gin.Context) {
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "ota").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get OTA configs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func (ac *AdminController) CreateOTAConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Type = "ota"
	ac.createConfigWithType(c, &config)
}

func (ac *AdminController) UpdateOTAConfig(c *gin.Context) {
	ac.updateConfigWithType(c, "ota")
}

func (ac *AdminController) DeleteOTAConfig(c *gin.Context) {
	ac.deleteConfigWithType(c, "ota")
}

// MQTT config management (frontend compatible)
func (ac *AdminController) GetMQTTConfigs(c *gin.Context) {
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "mqtt").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get MQTT configs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func (ac *AdminController) CreateMQTTConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Type = "mqtt"
	ac.createConfigWithType(c, &config)
}

func (ac *AdminController) UpdateMQTTConfig(c *gin.Context) {
	ac.updateConfigWithType(c, "mqtt")
}

func (ac *AdminController) DeleteMQTTConfig(c *gin.Context) {
	ac.deleteConfigWithType(c, "mqtt")
}

// MQTT Server config management (frontend compatible)
func (ac *AdminController) GetMQTTServerConfigs(c *gin.Context) {
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "mqtt_server").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get MQTT Server configs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func (ac *AdminController) CreateMQTTServerConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Type = "mqtt_server"
	ac.createConfigWithType(c, &config)
}

func (ac *AdminController) UpdateMQTTServerConfig(c *gin.Context) {
	ac.updateConfigWithType(c, "mqtt_server")
}

func (ac *AdminController) DeleteMQTTServerConfig(c *gin.Context) {
	ac.deleteConfigWithType(c, "mqtt_server")
}

// UDP config management (frontend compatible)
func (ac *AdminController) GetUDPConfigs(c *gin.Context) {
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "udp").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get UDP configs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func (ac *AdminController) CreateUDPConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Type = "udp"
	ac.createConfigWithType(c, &config)
}

func (ac *AdminController) UpdateUDPConfig(c *gin.Context) {
	ac.updateConfigWithType(c, "udp")
}

func (ac *AdminController) DeleteUDPConfig(c *gin.Context) {
	ac.deleteConfigWithType(c, "udp")
}

// ToggleConfigEnable toggles config enabled state
func (ac *AdminController) ToggleConfigEnable(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config ID"})
		return
	}

	var config models.Config
	if err := ac.DB.First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query config"})
		}
		return
	}

	// toggle enabled state
	config.Enabled = !config.Enabled
	if err := ac.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config status"})
		return
	}

	ac.notifySystemConfigChanged()
	status := "disabled"
	if config.Enabled {
		status = "enabled"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("config %s", status),
		"data":    config,
	})
}

// helper methods
func (ac *AdminController) createConfigWithType(c *gin.Context, config *models.Config) {
	// if config_id is not provided, auto-generate one
	if config.ConfigID == "" {
		// generate unique ID using type_name_timestamp format
		timestamp := time.Now().Unix()
		safeName := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(config.Name, " ", "_"), "-", "_"))
		config.ConfigID = fmt.Sprintf("%s_%s_%d", config.Type, safeName, timestamp)
	}

	// if setting as default, first unset other default configs of the same type
	if config.IsDefault {
		ac.DB.Model(&models.Config{}).Where("type = ? AND is_default = ?", config.Type, true).Update("is_default", false)
	}

	if err := ac.DB.Create(config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create config"})
		return
	}

	ac.notifySystemConfigChanged()
	c.JSON(http.StatusCreated, gin.H{"data": *config})
}

// configUpdateBody is used by updateConfigWithType, json_data accepts both string and object from frontend
type configUpdateBody struct {
	Name      string      `json:"name"`
	ConfigID  string      `json:"config_id"`
	Provider  string      `json:"provider"`
	JsonData  interface{} `json:"json_data"`
	Enabled   bool        `json:"enabled"`
	IsDefault bool        `json:"is_default"`
}

func (ac *AdminController) updateConfigWithType(c *gin.Context, configType string) {
	id, _ := strconv.Atoi(c.Param("id"))
	var config models.Config

	if err := ac.DB.Where("id = ? AND type = ?", id, configType).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	var updateData configUpdateBody
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// if setting as default, first unset other default configs of the same type
	if updateData.IsDefault {
		ac.DB.Model(&models.Config{}).Where("type = ? AND is_default = ? AND id != ?", configType, true, id).Update("is_default", false)
	}

	// update config
	config.Name = updateData.Name
	config.Provider = updateData.Provider
	config.Enabled = updateData.Enabled
	config.IsDefault = updateData.IsDefault

	// json_data: accepts both string and object to avoid binding failure when frontend sends object
	switch v := updateData.JsonData.(type) {
	case string:
		config.JsonData = v
	case nil:
		// if not provided, retain existing value
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json_data format"})
			return
		}
		config.JsonData = string(bytes)
	}

	// if a new config_id is provided, update it
	if updateData.ConfigID != "" {
		config.ConfigID = updateData.ConfigID
	}

	if err := ac.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config: " + err.Error()})
		return
	}

	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{"data": config})
}

func (ac *AdminController) deleteConfigWithType(c *gin.Context, configType string) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := ac.DB.Where("id = ? AND type = ?", id, configType).Delete(&models.Config{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete config"})
		return
	}
	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// config import/export methods
// ExportConfigs exports all configs as YAML
func (ac *AdminController) ExportConfigs(c *gin.Context) {
	// build export config structure - only include actually existing modules
	type ExportConfig struct {
		VAD           map[string]interface{} `yaml:"vad,omitempty"`
		ASR           map[string]interface{} `yaml:"asr,omitempty"`
		LLM           map[string]interface{} `yaml:"llm,omitempty"`
		TTS           map[string]interface{} `yaml:"tts,omitempty"`
		Vision        map[string]interface{} `yaml:"vision,omitempty"`
		Memory        map[string]interface{} `yaml:"memory,omitempty"`
		VoiceIdentify map[string]interface{} `yaml:"voice_identify,omitempty"`
		Auth          map[string]interface{} `yaml:"auth,omitempty"`
		Chat          map[string]interface{} `yaml:"chat,omitempty"`
		MQTT          map[string]interface{} `yaml:"mqtt,omitempty"`
		MQTTServer    map[string]interface{} `yaml:"mqtt_server,omitempty"`
		UDP           map[string]interface{} `yaml:"udp,omitempty"`
		OTA           map[string]interface{} `yaml:"ota,omitempty"`
		MCP           map[string]interface{} `yaml:"mcp,omitempty"`
		LocalMCP      map[string]interface{} `yaml:"local_mcp,omitempty"`
	}

	exportConfig := ExportConfig{
		VAD:           make(map[string]interface{}),
		ASR:           make(map[string]interface{}),
		LLM:           make(map[string]interface{}),
		TTS:           make(map[string]interface{}),
		Vision:        make(map[string]interface{}),
		Memory:        make(map[string]interface{}),
		VoiceIdentify: make(map[string]interface{}),
		Auth:          make(map[string]interface{}),
		Chat:          make(map[string]interface{}),
		MQTT:          make(map[string]interface{}),
		MQTTServer:    make(map[string]interface{}),
		UDP:           make(map[string]interface{}),
		OTA:           make(map[string]interface{}),
		MCP:           make(map[string]interface{}),
		LocalMCP:      make(map[string]interface{}),
	}

	// get all configs
	var configs []models.Config
	if err := ac.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get configs"})
		return
	}

	// get global roles
	var globalRoles []models.GlobalRole
	if err := ac.DB.Find(&globalRoles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get global roles"})
		return
	}

	// handle config data - provider field corresponds to is_default, key corresponds to ConfigID
	for _, config := range configs {
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(config.JsonData), &jsonData); err != nil {
			log.Printf("Failed to unmarshal config %s: %v", config.ConfigID, err)
			continue
		}

		// organize data by config type
		switch config.Type {
		case "vad":
			// support old format: if only one key, it's old format (with key), extract inner config
			var actualConfigData map[string]interface{}
			if len(jsonData) == 1 {
				// old format: single key, extract its value
				for _, value := range jsonData {
					if innerConfig, ok := value.(map[string]interface{}); ok {
						actualConfigData = innerConfig
					} else {
						// if not a map type, use raw data directly
						actualConfigData = jsonData
					}
					break
				}
			} else {
				// new format: no key, use jsonData directly
				actualConfigData = jsonData
			}
			// if this is the default config, set the provider field
			if config.IsDefault {
				exportConfig.VAD["provider"] = config.ConfigID
			}
			// use ConfigID as key
			exportConfig.VAD[config.ConfigID] = configprovider.ExportData(config.Type, config.ConfigID, config.Provider, actualConfigData)
		case "asr":
			if config.IsDefault {
				exportConfig.ASR["provider"] = config.ConfigID
			}
			exportConfig.ASR[config.ConfigID] = configprovider.ExportData(config.Type, config.ConfigID, config.Provider, jsonData)
		case "llm":
			if config.IsDefault {
				exportConfig.LLM["provider"] = config.ConfigID
			}
			exportConfig.LLM[config.ConfigID] = configprovider.ExportData(config.Type, config.ConfigID, config.Provider, jsonData)
		case "tts":
			if config.IsDefault {
				exportConfig.TTS["provider"] = config.ConfigID
			}
			exportConfig.TTS[config.ConfigID] = configprovider.ExportData(config.Type, config.ConfigID, config.Provider, jsonData)
		case "vision":
			// special handling for vision config
			if config.ConfigID == "vision_base" {
				// handle base config (enable_auth, vision_url, etc.)
				for key, value := range jsonData {
					exportConfig.Vision[key] = value
				}
			} else {
				// handle vllm config
				if exportConfig.Vision["vllm"] == nil {
					exportConfig.Vision["vllm"] = make(map[string]interface{})
				}
				if vllmConfig, ok := exportConfig.Vision["vllm"].(map[string]interface{}); ok {
					if config.IsDefault {
						vllmConfig["provider"] = config.ConfigID
					}
					vllmConfig[config.ConfigID] = configprovider.ExportData(config.Type, config.ConfigID, config.Provider, jsonData)
				}
			}
		case "ota":
			// ota, mqtt, mqtt_server, udp do not need provider field, merge config directly
			for key, value := range jsonData {
				exportConfig.OTA[key] = value
			}
		case "mqtt":
			// ota, mqtt, mqtt_server, udp do not need provider field, merge config directly
			for key, value := range jsonData {
				exportConfig.MQTT[key] = value
			}
		case "mqtt_server":
			// ota, mqtt, mqtt_server, udp do not need provider field, merge config directly
			for key, value := range jsonData {
				exportConfig.MQTTServer[key] = value
			}
		case "udp":
			// ota, mqtt, mqtt_server, udp do not need provider field, merge config directly
			for key, value := range jsonData {
				exportConfig.UDP[key] = value
			}
		case "memory":
			if config.IsDefault {
				exportConfig.Memory["provider"] = config.ConfigID
			}
			exportConfig.Memory[config.ConfigID] = configprovider.ExportData(config.Type, config.ConfigID, config.Provider, jsonData)
		case "voice_identify":
			if config.IsDefault {
				exportConfig.VoiceIdentify["provider"] = config.ConfigID
			}
			exportConfig.VoiceIdentify[config.ConfigID] = jsonData
		case "auth":
			for key, value := range jsonData {
				exportConfig.Auth[key] = value
			}
		case "chat":
			for key, value := range jsonData {
				exportConfig.Chat[key] = value
			}
		case "mcp":
			// handle MCP config, separate mcp and local_mcp
			if mcpData, exists := jsonData["mcp"]; exists {
				if mcpMap, ok := mcpData.(map[string]interface{}); ok {
					for key, value := range mcpMap {
						exportConfig.MCP[key] = value
					}
				}
			}
			// support old format: if global field exists directly
			if globalData, exists := jsonData["global"]; exists {
				exportConfig.MCP["global"] = globalData
			}
		case "local_mcp":
			// handle local_mcp config
			for key, value := range jsonData {
				exportConfig.LocalMCP[key] = value
			}
		}
	}

	// only process actual configs in DB, do not set defaults

	// convert to YAML
	yamlData, err := yaml.Marshal(exportConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal YAML"})
		return
	}

	// set response headers
	c.Header("Content-Type", "application/x-yaml")
	c.Header("Content-Disposition", "attachment; filename=config.yaml")
	c.Data(http.StatusOK, "application/x-yaml", yamlData)
}

// ImportConfigs imports configs from a YAML file
func (ac *AdminController) ImportConfigs(c *gin.Context) {
	log.Printf("starting config import")

	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("failed to get uploaded file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	log.Printf("file info: filename=%s, size=%d", file.Filename, file.Size)

	if file.Size == 0 {
		log.Printf("file is empty")
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is empty"})
		return
	}

	// read file content
	src, err := file.Open()
	if err != nil {
		log.Printf("failed to open file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		log.Printf("failed to read file content: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	log.Printf("file content length: %d", len(content))

	// parse YAML
	var importConfig map[string]interface{}
	if err := yaml.Unmarshal(content, &importConfig); err != nil {
		log.Printf("failed to parse YAML: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid YAML format"})
		return
	}

	log.Printf("YAML parsed successfully, config keys: %v", getMapKeys(importConfig))

	// begin transaction
	log.Printf("starting database transaction")
	tx := ac.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic occurred, rolling back transaction: %v", r)
			tx.Rollback()
		}
	}()

	// clear existing configs
	log.Printf("clearing existing configs")
	result := tx.Exec("DELETE FROM configs")
	if result.Error != nil {
		log.Printf("failed to clear configs: %v", result.Error)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear existing configs"})
		return
	}
	log.Printf("configs cleared, deleted %d records", result.RowsAffected)

	// clear global roles
	log.Printf("clearing global roles")
	result2 := tx.Exec("DELETE FROM global_roles")
	if result2.Error != nil {
		log.Printf("failed to clear global roles: %v", result2.Error)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear existing global roles"})
		return
	}
	log.Printf("global roles cleared, deleted %d records", result2.RowsAffected)

	// import configs - only process actually present modules
	configTypes := []string{"vad", "asr", "llm", "tts", "memory", "auth", "chat", "ota", "mqtt", "mqtt_server", "udp", "mcp", "local_mcp"}
	log.Printf("starting config import, config types: %v", configTypes)

	// handle voice_identify config (mapped to speaker type)
	if voiceIdentifyData, exists := importConfig["voice_identify"]; exists {
		log.Printf("found voice_identify config data")
		if voiceIdentifyMap, ok := voiceIdentifyData.(map[string]interface{}); ok {
			log.Printf("voice_identify config map keys: %v", getMapKeys(voiceIdentifyMap))

			// get provider field
			var defaultProvider string
			if provider, exists := voiceIdentifyMap["provider"]; exists {
				if providerStr, ok := provider.(string); ok {
					defaultProvider = providerStr
					log.Printf("voice_identify default provider: %s", defaultProvider)
				}
			}

			log.Printf("voice_identify config item keys: %v", getMapKeys(voiceIdentifyMap))
			// there is only one voice-print config, prefer provider-specified config, otherwise use first config item
			var targetConfigID string
			if defaultProvider != "" {
				targetConfigID = defaultProvider
			} else {
				// if no provider, use first non-provider config item
				for key := range voiceIdentifyMap {
					if key != "provider" {
						targetConfigID = key
						break
					}
				}
			}

			if targetConfigID == "" {
				log.Printf("no valid config item found in voice_identify config")
			} else {
				// only process target config item
				if configValue, exists := voiceIdentifyMap[targetConfigID]; exists {
					if configMap, ok := configValue.(map[string]interface{}); ok {
						log.Printf("processing voice_identify config item: %s", targetConfigID)
						jsonData, err := json.Marshal(configMap)
						if err != nil {
							log.Printf("failed to serialize voice_identify config data: %v", err)
							tx.Rollback()
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal voice_identify config data"})
							return
						}

						// there is only one voice-print config, always set as default
						config := models.Config{
							Type:      "voice_identify",
							Name:      "voice-print recognition config",
							ConfigID:  "asr_server",
							Provider:  "asr_server",
							JsonData:  string(jsonData),
							Enabled:   true,
							IsDefault: true,
						}

						log.Printf("preparing to save voice_identify config: Type=%s, Name=%s, ConfigID=%s", config.Type, config.Name, config.ConfigID)

						// there is only one voice-print config, delete all old configs first
						tx.Where("type = ?", "voice_identify").Delete(&models.Config{})

						// create new config
						if err := tx.Create(&config).Error; err != nil {
							log.Printf("failed to create voice_identify config: %v", err)
							tx.Rollback()
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create voice_identify config"})
							return
						}
						log.Printf("voice_identify config created: %s", targetConfigID)
					}
				}
			}
		}
	}

	for _, configType := range configTypes {
		log.Printf("processing config type: %s", configType)
		if configData, exists := importConfig[configType]; exists {
			log.Printf("found data for config type %s", configType)
			if configMap, ok := configData.(map[string]interface{}); ok {
				// for modules requiring provider (vad, asr, llm, tts, memory), handle provider field
				if configType == "vad" || configType == "asr" || configType == "llm" || configType == "tts" || configType == "memory" || configType == "voice_identify" {
					log.Printf("processing config type requiring provider: %s", configType)
					// get provider field
					var defaultProvider string
					if provider, exists := configMap["provider"]; exists {
						if providerStr, ok := provider.(string); ok {
							defaultProvider = providerStr
							log.Printf("default provider: %s", defaultProvider)
						}
					}

					log.Printf("config item keys: %v", getMapKeys(configMap))
					// iterate all config items
					for configID, configValue := range configMap {
						// skip provider field
						if configID == "provider" {
							log.Printf("skipping provider field")
							continue
						}

						if configMap, ok := configValue.(map[string]interface{}); ok {
							log.Printf("processing config item: %s", configID)
							providerName := configprovider.NormalizeProvider(configType, configID, configMap)
							if providerName != "" {
								configMap["provider"] = providerName
							}
							jsonData, err := json.Marshal(configMap)
							if err != nil {
								log.Printf("failed to serialize config data: %v", err)
								tx.Rollback()
								c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal config data"})
								return
							}

							// check if this is the default config
							isDefault := (configID == defaultProvider)
							log.Printf("config item %s, is default: %v", configID, isDefault)

							config := models.Config{
								Type:      configType,
								Name:      configID,
								ConfigID:  configID,
								Provider:  providerName,
								JsonData:  string(jsonData),
								Enabled:   true,
								IsDefault: isDefault,
							}

							log.Printf("preparing to save config: Type=%s, Name=%s, ConfigID=%s", config.Type, config.Name, config.ConfigID)

							// first check if same config already exists
							var existingConfig models.Config
							if err := tx.Where("type = ? AND config_id = ?", config.Type, config.ConfigID).First(&existingConfig).Error; err == nil {
								log.Printf("config already exists, will update: Type=%s, ConfigID=%s", config.Type, config.ConfigID)
								// update existing config
								existingConfig.Name = config.Name
								existingConfig.Provider = config.Provider
								existingConfig.JsonData = config.JsonData
								existingConfig.Enabled = config.Enabled
								existingConfig.IsDefault = config.IsDefault
								if err := tx.Save(&existingConfig).Error; err != nil {
									log.Printf("failed to update config: %v", err)
									tx.Rollback()
									c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
									return
								}
								log.Printf("config updated: %s", configID)
							} else if err == gorm.ErrRecordNotFound {
								log.Printf("config not found, will create new: Type=%s, ConfigID=%s", config.Type, config.ConfigID)
								// create new config
								if err := tx.Create(&config).Error; err != nil {
									log.Printf("failed to create config: %v", err)
									tx.Rollback()
									c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create config"})
									return
								}
								log.Printf("config created: %s", configID)
							} else {
								log.Printf("error querying config: %v", err)
								tx.Rollback()
								c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query existing config"})
								return
							}
						}
					}
				} else {
					// for modules not requiring provider (ota, mqtt, mqtt_server, udp, mcp, local_mcp), create config directly
					log.Printf("processing config type not requiring provider: %s", configType)
					jsonData, err := json.Marshal(configMap)
					if err != nil {
						log.Printf("failed to serialize config data: %v", err)
						tx.Rollback()
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal config data"})
						return
					}

					config := models.Config{
						Type:      configType,
						Name:      configType,
						ConfigID:  configType,
						Provider:  "",
						JsonData:  string(jsonData),
						Enabled:   true,
						IsDefault: true,
					}

					log.Printf("preparing to save config: Type=%s, Name=%s, ConfigID=%s", config.Type, config.Name, config.ConfigID)

					// first check if same config already exists
					var existingConfig models.Config
					if err := tx.Where("type = ? AND config_id = ?", config.Type, config.ConfigID).First(&existingConfig).Error; err == nil {
						log.Printf("config already exists, will update: Type=%s, ConfigID=%s", config.Type, config.ConfigID)
						// update existing config
						existingConfig.Name = config.Name
						existingConfig.Provider = config.Provider
						existingConfig.JsonData = config.JsonData
						existingConfig.Enabled = config.Enabled
						existingConfig.IsDefault = config.IsDefault
						if err := tx.Save(&existingConfig).Error; err != nil {
							log.Printf("failed to update config: %v", err)
							tx.Rollback()
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
							return
						}
						log.Printf("config updated: %s", configType)
					} else if err == gorm.ErrRecordNotFound {
						log.Printf("config not found, will create new: Type=%s, ConfigID=%s", config.Type, config.ConfigID)
						// create new config
						if err := tx.Create(&config).Error; err != nil {
							log.Printf("failed to create config: %v", err)
							tx.Rollback()
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create config"})
							return
						}
						log.Printf("config created: %s", configType)
					} else {
						log.Printf("error querying config: %v", err)
						tx.Rollback()
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query existing config"})
						return
					}
				}
			}
		}
	}

	// special handling for vision config
	log.Printf("starting vision config processing")
	if visionData, exists := importConfig["vision"]; exists {
		log.Printf("found vision config data")
		if visionMap, ok := visionData.(map[string]interface{}); ok {
			log.Printf("vision config map keys: %v", getMapKeys(visionMap))

			// handle vision base config (enable_auth, vision_url, etc.)
			baseVisionConfig := make(map[string]interface{})
			for key, value := range visionMap {
				if key != "vllm" {
					baseVisionConfig[key] = value
				}
			}

			// save vision base config
			if len(baseVisionConfig) > 0 {
				jsonData, err := json.Marshal(baseVisionConfig)
				if err != nil {
					log.Printf("failed to serialize vision base config data: %v", err)
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal vision base config data"})
					return
				}

				config := models.Config{
					Type:      "vision",
					Name:      "vision_base",
					ConfigID:  "vision_base",
					Provider:  "vision_base",
					JsonData:  string(jsonData),
					Enabled:   true,
					IsDefault: false,
				}

				log.Printf("preparing to save vision base config: Type=%s, Name=%s, ConfigID=%s", config.Type, config.Name, config.ConfigID)

				// first check if same config already exists
				var existingConfig models.Config
				if err := tx.Where("type = ? AND config_id = ?", config.Type, config.ConfigID).First(&existingConfig).Error; err == nil {
					log.Printf("vision base config already exists, will update: Type=%s, ConfigID=%s", config.Type, config.ConfigID)
					// update existing config
					existingConfig.Name = config.Name
					existingConfig.Provider = config.Provider
					existingConfig.JsonData = config.JsonData
					existingConfig.Enabled = config.Enabled
					existingConfig.IsDefault = config.IsDefault
					if err := tx.Save(&existingConfig).Error; err != nil {
						log.Printf("failed to update vision base config: %v", err)
						tx.Rollback()
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vision base config"})
						return
					}
					log.Printf("vision base config updated")
				} else if err == gorm.ErrRecordNotFound {
					log.Printf("vision base config not found, will create new: Type=%s, ConfigID=%s", config.Type, config.ConfigID)
					// create new config
					if err := tx.Create(&config).Error; err != nil {
						log.Printf("failed to create vision base config: %v", err)
						tx.Rollback()
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create vision base config"})
						return
					}
					log.Printf("vision base config created")
				} else {
					log.Printf("error querying vision base config: %v", err)
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query existing vision base config"})
					return
				}
			}

			// handle vllm config
			if vllmData, exists := visionMap["vllm"]; exists {
				log.Printf("found vllm config data")
				if vllmMap, ok := vllmData.(map[string]interface{}); ok {
					log.Printf("vllm config map keys: %v", getMapKeys(vllmMap))

					// get vllm provider field
					var defaultProvider string
					if provider, exists := vllmMap["provider"]; exists {
						if providerStr, ok := provider.(string); ok {
							defaultProvider = providerStr
							log.Printf("vllm default provider: %s", defaultProvider)
						}
					}

					log.Printf("vllm config item keys: %v", getMapKeys(vllmMap))
					// iterate all vllm config items
					for configID, configValue := range vllmMap {
						// skip provider field
						if configID == "provider" {
							log.Printf("skipping vllm provider field")
							continue
						}

						if configMap, ok := configValue.(map[string]interface{}); ok {
							log.Printf("processing vllm config item: %s", configID)
							providerName := configprovider.NormalizeProvider("vision", configID, configMap)
							if providerName != "" {
								configMap["provider"] = providerName
							}
							jsonData, err := json.Marshal(configMap)
							if err != nil {
								log.Printf("failed to serialize vllm config data: %v", err)
								tx.Rollback()
								c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal vllm config data"})
								return
							}

							// check if this is the default config
							isDefault := (configID == defaultProvider)
							log.Printf("vllm config item %s, is default: %v", configID, isDefault)

							config := models.Config{
								Type:      "vision",
								Name:      configID,
								ConfigID:  configID,
								Provider:  providerName,
								JsonData:  string(jsonData),
								Enabled:   true,
								IsDefault: isDefault,
							}

							log.Printf("preparing to save vllm config: Type=%s, Name=%s, ConfigID=%s", config.Type, config.Name, config.ConfigID)

							// first check if same config already exists
							var existingConfig models.Config
							if err := tx.Where("type = ? AND config_id = ?", config.Type, config.ConfigID).First(&existingConfig).Error; err == nil {
								log.Printf("vllm config already exists, will update: Type=%s, ConfigID=%s", config.Type, config.ConfigID)
								// update existing config
								existingConfig.Name = config.Name
								existingConfig.Provider = config.Provider
								existingConfig.JsonData = config.JsonData
								existingConfig.Enabled = config.Enabled
								existingConfig.IsDefault = config.IsDefault
								if err := tx.Save(&existingConfig).Error; err != nil {
									log.Printf("failed to update vllm config: %v", err)
									tx.Rollback()
									c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vllm config"})
									return
								}
								log.Printf("vllm config updated: %s", configID)
							} else if err == gorm.ErrRecordNotFound {
								log.Printf("vllm config not found, will create new: Type=%s, ConfigID=%s", config.Type, config.ConfigID)
								// create new config
								if err := tx.Create(&config).Error; err != nil {
									log.Printf("failed to create vllm config: %v", err)
									tx.Rollback()
									c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create vllm config"})
									return
								}
								log.Printf("vllm config created: %s", configID)
							} else {
								log.Printf("error querying vllm config: %v", err)
								tx.Rollback()
								c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query existing vllm config"})
								return
							}
						}
					}
				}
			}
		}
	}

	// special handling for local_mcp config
	log.Printf("starting local_mcp config processing")
	if localMcpData, exists := importConfig["local_mcp"]; exists {
		log.Printf("found local_mcp config data")
		if localMcpMap, ok := localMcpData.(map[string]interface{}); ok {
			log.Printf("local_mcp config map keys: %v", getMapKeys(localMcpMap))

			jsonData, err := json.Marshal(localMcpMap)
			if err != nil {
				log.Printf("failed to serialize local_mcp config data: %v", err)
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal local_mcp config data"})
				return
			}

			config := models.Config{
				Type:      "local_mcp",
				Name:      "local_mcp",
				ConfigID:  "local_mcp",
				Provider:  "",
				JsonData:  string(jsonData),
				Enabled:   true,
				IsDefault: true,
			}

			log.Printf("preparing to save local_mcp config: Type=%s, Name=%s, ConfigID=%s", config.Type, config.Name, config.ConfigID)

			// first check if same config already exists
			var existingConfig models.Config
			if err := tx.Where("type = ? AND config_id = ?", config.Type, config.ConfigID).First(&existingConfig).Error; err == nil {
				log.Printf("local_mcp config already exists, will update: Type=%s, ConfigID=%s", config.Type, config.ConfigID)
				// update existing config
				existingConfig.Name = config.Name
				existingConfig.Provider = config.Provider
				existingConfig.JsonData = config.JsonData
				existingConfig.Enabled = config.Enabled
				existingConfig.IsDefault = config.IsDefault
				if err := tx.Save(&existingConfig).Error; err != nil {
					log.Printf("failed to update local_mcp config: %v", err)
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update local_mcp config"})
					return
				}
				log.Printf("local_mcp config updated")
			} else if err == gorm.ErrRecordNotFound {
				log.Printf("local_mcp config not found, will create new: Type=%s, ConfigID=%s", config.Type, config.ConfigID)
				// create new config
				if err := tx.Create(&config).Error; err != nil {
					log.Printf("failed to create local_mcp config: %v", err)
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create local_mcp config"})
					return
				}
				log.Printf("local_mcp config created")
			} else {
				log.Printf("error querying local_mcp config: %v", err)
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query existing local_mcp config"})
				return
			}
		}
	}

	// commit transaction
	log.Printf("committing transaction")
	if err := tx.Commit().Error; err != nil {
		log.Printf("failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	log.Printf("config imported successfully")
	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{"message": "Configuration imported successfully"})
}

// MCP config related methods
func (ac *AdminController) GetMCPConfigs(c *gin.Context) {
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "mcp").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get MCP config list"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func (ac *AdminController) CreateMCPConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.Type = "mcp"

	// if setting as default, first unset other default configs of the same type
	if config.IsDefault {
		ac.DB.Model(&models.Config{}).Where("type = ? AND is_default = ?", config.Type, true).Update("is_default", false)
	}

	if err := ac.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create MCP config"})
		return
	}
	ac.notifySystemConfigChanged()
	c.JSON(http.StatusCreated, gin.H{"data": config})
}

func (ac *AdminController) UpdateMCPConfig(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var config models.Config

	if err := ac.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MCP config not found"})
		return
	}

	var updateData models.Config
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// if setting as default, first unset other default configs of the same type
	if updateData.IsDefault {
		ac.DB.Model(&models.Config{}).Where("type = ? AND is_default = ? AND id != ?", config.Type, true, id).Update("is_default", false)
	}

	updateData.Type = "mcp"
	if err := ac.DB.Model(&config).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update MCP config"})
		return
	}
	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{"data": config})
}

func (ac *AdminController) DeleteMCPConfig(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var config models.Config

	if err := ac.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MCP config not found"})
		return
	}

	if err := ac.DB.Delete(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete MCP config"})
		return
	}
	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{"message": "MCP config deleted successfully"})
}

// GenerateAgentMCPEndpoint is the common MCP endpoint generation function
func GenerateAgentMCPEndpoint(db *gorm.DB, agentID string, userID uint, endpointAuthToken string) (string, error) {
	// get external WebSocket URL from OTA config
	var otaConfig models.Config
	if err := db.Where("type = ? AND is_default = ?", "ota", true).First(&otaConfig).Error; err != nil {
		return "", fmt.Errorf("failed to get OTA config: %v", err)
	}

	var otaData map[string]interface{}
	if err := json.Unmarshal([]byte(otaConfig.JsonData), &otaData); err != nil {
		return "", fmt.Errorf("failed to parse OTA config: %v", err)
	}

	// get external WebSocket URL
	externalURL, ok := otaData["external"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("external config not found in OTA config")
	}

	websocketConfig, ok := externalURL["websocket"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("websocket config not found in external config")
	}

	wsURL, ok := websocketConfig["url"].(string)
	if !ok || wsURL == "" {
		return "", fmt.Errorf("websocket URL not found in external config")
	}

	// parse OTA URL, take only domain part, keep ws or wss protocol unchanged
	parsedURL, err := url.Parse(wsURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse WebSocket URL: %v", err)
	}

	// build base URL (protocol and domain only)
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// generate MCP JWT token
	token, err := generateMCPToken(agentID, userID, endpointAuthToken)
	if err != nil {
		return "", fmt.Errorf("failed to generate MCP token: %v", err)
	}

	// build full endpoint URL with token, using /mcp path directly
	endpointWithToken := fmt.Sprintf("%s/mcp?token=%s", baseURL, token)

	return endpointWithToken, nil
}

// GenerateAgentOpenClawEndpoint is the common OpenClaw endpoint generation function
func GenerateAgentOpenClawEndpoint(db *gorm.DB, agentID string, userID uint, endpointAuthToken string) (string, error) {
	var otaConfig models.Config
	if err := db.Where("type = ? AND is_default = ?", "ota", true).First(&otaConfig).Error; err != nil {
		return "", fmt.Errorf("failed to get OTA config: %v", err)
	}

	var otaData map[string]interface{}
	if err := json.Unmarshal([]byte(otaConfig.JsonData), &otaData); err != nil {
		return "", fmt.Errorf("failed to parse OTA config: %v", err)
	}

	externalURL, ok := otaData["external"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("external config not found in OTA config")
	}

	websocketConfig, ok := externalURL["websocket"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("websocket config not found in external config")
	}

	wsURL, ok := websocketConfig["url"].(string)
	if !ok || wsURL == "" {
		return "", fmt.Errorf("websocket URL not found in external config")
	}

	parsedURL, err := url.Parse(wsURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse WebSocket URL: %v", err)
	}

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	token, err := generateOpenClawToken(agentID, userID, endpointAuthToken)
	if err != nil {
		return "", fmt.Errorf("failed to generate OpenClaw token: %v", err)
	}

	endpointWithToken := fmt.Sprintf("%s/ws/openclaw?token=%s", baseURL, token)
	return endpointWithToken, nil
}

// Memory config management
func (ac *AdminController) GetMemoryConfigs(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	offset := (page - 1) * pageSize
	var total int64
	if err := ac.DB.Model(&models.Config{}).Where("type = ?", "memory").Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get Memory config list"})
		return
	}
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "memory").Order("id DESC").Offset(offset).Limit(pageSize).Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get Memory config list"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs, "total": total, "page": page, "page_size": pageSize})
}

func (ac *AdminController) CreateMemoryConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// set config type to memory
	config.Type = "memory"

	// validate provider field
	if config.Provider != "memobase" && config.Provider != "mem0" && config.Provider != "memos" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider must be memobase, mem0, or memos"})
		return
	}

	// if setting as default, first unset other default configs of the same type
	if config.IsDefault {
		ac.DB.Model(&models.Config{}).Where("type = ? AND is_default = ?", config.Type, true).Update("is_default", false)
	}

	if err := ac.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create Memory config"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": config})
}

func (ac *AdminController) UpdateMemoryConfig(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var config models.Config

	if err := ac.DB.Where("id = ? AND type = ?", id, "memory").First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Memory config not found"})
		return
	}

	var updateData models.Config
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// validate provider field
	if updateData.Provider != "memobase" && updateData.Provider != "mem0" && updateData.Provider != "memos" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider must be memobase, mem0, or memos"})
		return
	}

	// if setting as default, first unset other default configs of the same type
	if updateData.IsDefault {
		ac.DB.Model(&models.Config{}).Where("type = ? AND is_default = ? AND id != ?", config.Type, true, id).Update("is_default", false)
	}

	// update config
	config.Name = updateData.Name
	config.Provider = updateData.Provider
	config.JsonData = updateData.JsonData
	config.Enabled = updateData.Enabled
	config.IsDefault = updateData.IsDefault

	if err := ac.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update Memory config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": config})
}

func (ac *AdminController) DeleteMemoryConfig(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := ac.DB.Where("id = ? AND type = ?", id, "memory").Delete(&models.Config{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete Memory config"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// set default Memory config
func (ac *AdminController) SetDefaultMemoryConfig(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var config models.Config

	if err := ac.DB.Where("id = ? AND type = ?", id, "memory").First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Memory config not found"})
		return
	}

	// first unset other default configs of the same type
	ac.DB.Model(&models.Config{}).Where("type = ? AND is_default = ?", config.Type, true).Update("is_default", false)

	// set current config as default
	config.IsDefault = true
	if err := ac.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set default Memory config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "default Memory config set successfully", "data": config})
}

// generateMCPToken generates a stable MCP JWT Token (consistent for same agentID+userID)
func generateMCPToken(agentID string, userID uint, endpointAuthToken string) (string, error) {
	// create custom JWT Claims
	type MCPClaims struct {
		UserID     uint   `json:"userId"`
		AgentID    string `json:"agentId"`
		EndpointID string `json:"endpointId"`
		Purpose    string `json:"purpose"`
		jwt.RegisteredClaims
	}

	// build endpointId
	endpointID := fmt.Sprintf("agent_%s", agentID)

	// create JWT claims.
	// no iat/exp set, ensuring long-lived token with stable output for same agentID+userID.
	claims := MCPClaims{
		UserID:           userID,
		AgentID:          agentID,
		EndpointID:       endpointID,
		Purpose:          "mcp-endpoint",
		RegisteredClaims: jwt.RegisteredClaims{},
	}

	// generate JWT token using HS256 algorithm
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// use same key as middleware
	jwtSecret := []byte(strings.TrimSpace(endpointAuthToken))
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// generateOpenClawToken generates a stable OpenClaw JWT Token (consistent for same agentID+userID)
func generateOpenClawToken(agentID string, userID uint, endpointAuthToken string) (string, error) {
	type OpenClawClaims struct {
		UserID     uint   `json:"user_id"`
		AgentID    string `json:"agent_id"`
		EndpointID string `json:"endpoint_id"`
		Purpose    string `json:"purpose"`
		jwt.RegisteredClaims
	}

	endpointID := fmt.Sprintf("agent_%s", agentID)
	claims := OpenClawClaims{
		UserID:           userID,
		AgentID:          agentID,
		EndpointID:       endpointID,
		Purpose:          "openclaw-endpoint",
		RegisteredClaims: jwt.RegisteredClaims{},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtSecret := []byte(strings.TrimSpace(endpointAuthToken))
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ==================== new role management API ====================

// GetGlobalRolesNew returns global role list (only global roles in roles table)
func (ac *AdminController) GetGlobalRolesNew(c *gin.Context) {
	var globalRoles []models.Role
	if err := ac.DB.Where("user_id IS NULL AND role_type = ?", "global").
		Order("sort_order ASC, id ASC").
		Find(&globalRoles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get global roles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": globalRoles})
}

// GetRolesNew returns role list (global roles + user roles)
// admins can view all roles, regular users can only view global roles and their own roles
func (ac *AdminController) GetRolesNew(c *gin.Context) {
	// get user ID and role from JWT
	userID, exists := c.Get("user_id")
	userRole, roleExists := c.Get("role")

	// query global roles
	var globalRoles []models.Role
	if err := ac.DB.Where("user_id IS NULL AND role_type = ?", "global").
		Order("sort_order ASC, id ASC").
		Find(&globalRoles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get global roles"})
		return
	}

	// query user roles
	var userRoles []models.Role
	if roleExists && userRole.(string) == "admin" {
		// admin views all user roles
		if err := ac.DB.Where("role_type = ?", "user").
			Order("created_at DESC").
			Find(&userRoles).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user roles"})
			return
		}
	} else if exists {
		// regular users only view their own roles
		if err := ac.DB.Where("user_id = ? AND role_type = ?", userID, "user").
			Order("created_at DESC").
			Find(&userRoles).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user roles"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"global_roles": globalRoles,
			"user_roles":   userRoles,
		},
	})
}

// GetRoleNew returns single role details
func (ac *AdminController) GetRoleNew(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var role models.Role

	if err := ac.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	if strings.Contains(c.FullPath(), "/admin/roles/global/") && role.RoleType != "global" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this endpoint only allows operating on global roles"})
		return
	}

	// permission check: user roles can only view their own roles
	if role.UserID != nil {
		userID, exists := c.Get("user_id")
		userRole, roleExists := c.Get("role")

		if roleExists && userRole.(string) != "admin" {
			if exists && userID != nil {
				uid := userID.(uint)
				if uid != *role.UserID {
					c.JSON(http.StatusForbidden, gin.H{"error": "no permission to access this role"})
					return
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": role})
}

func normalizeRoleStatus(status string) string {
	trimmed := strings.TrimSpace(status)
	if trimmed == "" {
		return "active"
	}
	return trimmed
}

// CreateRoleNew creates a role (admin creates global role, user creates their own role)
func (ac *AdminController) CreateRoleNew(c *gin.Context) {
	userID, exists := c.Get("user_id")
	userRole, roleExists := c.Get("role")

	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// set role type and owner user
	if roleExists && userRole.(string) == "admin" {
		// admin creates global role
		role.RoleType = "global"
		role.UserID = nil
	} else if exists {
		// regular user creates their own role
		role.RoleType = "user"
		uid := userID.(uint)
		role.UserID = &uid
		// user roles cannot be set as default
		role.IsDefault = false
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// validate required fields
	if role.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role name cannot be empty"})
		return
	}
	if role.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "system prompt cannot be empty"})
		return
	}

	role.Status = normalizeRoleStatus(role.Status)
	if role.Status != "active" && role.Status != "inactive" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role status"})
		return
	}

	// if setting as default role, first unset other default roles
	if role.IsDefault && role.RoleType == "global" {
		ac.DB.Model(&models.Role{}).
			Where("role_type = ? AND is_default = ?", "global", true).
			Update("is_default", false)
	}

	if err := ac.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create role"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": role})
}

// UpdateRoleNew updates a role
func (ac *AdminController) UpdateRoleNew(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var role models.Role

	if err := ac.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	if strings.Contains(c.FullPath(), "/admin/roles/global/") && role.RoleType != "global" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this endpoint only allows operating on global roles"})
		return
	}

	// permission check
	userID, exists := c.Get("user_id")
	userRole, roleExists := c.Get("role")

	isAdmin := roleExists && userRole.(string) == "admin"
	isOwner := false
	if exists && role.UserID != nil {
		if uid, ok := userID.(uint); ok {
			isOwner = uid == *role.UserID
		}
	}

	if !isAdmin && !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "no permission to modify this role"})
		return
	}

	var updateData models.Role
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// if setting as default role, first unset other default roles
	if updateData.IsDefault && role.RoleType == "global" {
		ac.DB.Model(&models.Role{}).
			Where("role_type = ? AND is_default = ? AND id != ?", "global", true, id).
			Update("is_default", false)
	}

	// update fields
	role.Name = updateData.Name
	role.Description = updateData.Description
	role.Prompt = updateData.Prompt
	role.LLMConfigID = updateData.LLMConfigID
	role.TTSConfigID = updateData.TTSConfigID
	role.Voice = updateData.Voice
	role.SortOrder = updateData.SortOrder

	normalizedStatus := strings.TrimSpace(updateData.Status)
	if normalizedStatus == "" {
		normalizedStatus = role.Status
	}
	normalizedStatus = normalizeRoleStatus(normalizedStatus)
	if normalizedStatus != "active" && normalizedStatus != "inactive" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role status"})
		return
	}
	role.Status = normalizedStatus

	// only admins can modify the default flag and role type
	if isAdmin {
		role.IsDefault = updateData.IsDefault
	}

	if err := ac.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": role})
}

// DeleteRoleNew deletes a role
func (ac *AdminController) DeleteRoleNew(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var role models.Role

	if err := ac.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	if strings.Contains(c.FullPath(), "/admin/roles/global/") && role.RoleType != "global" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this endpoint only allows operating on global roles"})
		return
	}

	// permission check
	userID, exists := c.Get("user_id")
	userRole, roleExists := c.Get("role")

	isAdmin := roleExists && userRole.(string) == "admin"
	isOwner := false
	if exists && role.UserID != nil {
		if uid, ok := userID.(uint); ok {
			isOwner = uid == *role.UserID
		}
	}

	if !isAdmin && !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "no permission to delete this role"})
		return
	}

	// check if any device is using this role
	var deviceCount int64
	ac.DB.Model(&models.Device{}).Where("role_id = ?", id).Count(&deviceCount)
	if deviceCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("cannot delete: %d device(s) are using this role, please unlink them first", deviceCount),
		})
		return
	}

	if err := ac.DB.Delete(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// ToggleRoleStatus toggles role status (enable/disable)
func (ac *AdminController) ToggleRoleStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var role models.Role

	if err := ac.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	if strings.Contains(c.FullPath(), "/admin/roles/global/") && role.RoleType != "global" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this endpoint only allows operating on global roles"})
		return
	}

	// permission check
	userID, exists := c.Get("user_id")
	userRole, roleExists := c.Get("role")

	isAdmin := roleExists && userRole.(string) == "admin"
	isOwner := false
	if exists && role.UserID != nil {
		if uid, ok := userID.(uint); ok {
			isOwner = uid == *role.UserID
		}
	}

	if !isAdmin && !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "no permission to modify this role"})
		return
	}

	// toggle status
	currentStatus := normalizeRoleStatus(role.Status)
	if currentStatus == "active" {
		role.Status = "inactive"
	} else {
		role.Status = "active"
	}

	if err := ac.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": role})
}

// SetDefaultRole sets the default role (global roles only)
func (ac *AdminController) SetDefaultRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var role models.Role

	if err := ac.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	// only global roles can be set as default
	if role.RoleType != "global" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only global roles can be set as default"})
		return
	}

	// permission check: only admins can set the default role
	userRole, roleExists := c.Get("role")
	if !roleExists || userRole.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admins can set the default role"})
		return
	}

	// first unset other default roles
	ac.DB.Model(&models.Role{}).
		Where("role_type = ? AND is_default = ?", "global", true).
		Update("is_default", false)

	// set current role as default
	role.IsDefault = true
	if err := ac.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set default role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": role, "message": "set as default role"})
}

type applyDeviceRoleRequest struct {
	RoleID *uint `json:"role_id"`
}

type switchDeviceRoleByNameRequest struct {
	RoleName string `json:"role_name"`
}

func normalizeRoleNameForMatch(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}

func calcRoleMatchScore(requestedRoleName string, candidateRoleName string) (int, string) {
	reqCompact := normalizeRoleNameForMatch(requestedRoleName)
	candCompact := normalizeRoleNameForMatch(candidateRoleName)
	if reqCompact == "" || candCompact == "" {
		return -1, ""
	}

	if reqCompact == candCompact {
		return 1000, "exact"
	}

	if strings.Contains(candCompact, reqCompact) || strings.Contains(reqCompact, candCompact) {
		score := 700 - absInt(len(candCompact)-len(reqCompact))
		if strings.HasPrefix(candCompact, reqCompact) || strings.HasPrefix(reqCompact, candCompact) {
			score += 50
		}
		return score, "fuzzy"
	}

	reqRaw := strings.ToLower(strings.TrimSpace(requestedRoleName))
	candRaw := strings.ToLower(strings.TrimSpace(candidateRoleName))
	if reqRaw != "" && candRaw != "" && (strings.Contains(candRaw, reqRaw) || strings.Contains(reqRaw, candRaw)) {
		score := 600 - absInt(len(candRaw)-len(reqRaw))
		return score, "fuzzy"
	}

	return -1, ""
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func matchDeviceRoleByName(requestedRoleName string, roles []models.Role) (*models.Role, string) {
	bestScore := -1
	bestMatchType := ""
	var bestRole *models.Role

	for i := range roles {
		role := &roles[i]
		if normalizeRoleStatus(role.Status) != "active" {
			continue
		}

		score, matchType := calcRoleMatchScore(requestedRoleName, role.Name)
		if score > bestScore {
			bestScore = score
			bestMatchType = matchType
			bestRole = role
		}
	}

	if bestScore < 0 {
		return nil, ""
	}
	return bestRole, bestMatchType
}

func getRequestUserInfo(c *gin.Context) (uint, bool, bool) {
	var uid uint
	userID, hasUserID := c.Get("user_id")
	if hasUserID {
		if v, ok := userID.(uint); ok {
			uid = v
		}
	}

	roleVal, hasRole := c.Get("role")
	isAdmin := hasRole && roleVal == "admin"
	return uid, hasUserID, isAdmin
}

// ApplyRoleToDevice applies a role to a device (regular users can operate their own devices)
func (ac *AdminController) ApplyRoleToDevice(c *gin.Context) {
	deviceID, err := strconv.Atoi(c.Param("id"))
	if err != nil || deviceID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID"})
		return
	}

	var req applyDeviceRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var device models.Device
	if err := ac.DB.First(&device, deviceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	uid, hasUserID, isAdmin := getRequestUserInfo(c)
	if !isAdmin {
		if !hasUserID || device.UserID != uid {
			c.JSON(http.StatusForbidden, gin.H{"error": "no permission to operate this device"})
			return
		}
	}

	if req.RoleID != nil {
		var role models.Role
		if err := ac.DB.First(&role, *req.RoleID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role not found"})
			return
		}

		roleStatus := normalizeRoleStatus(role.Status)
		if roleStatus != "active" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role is not enabled"})
			return
		}
		if role.Status == "" {
			if err := ac.DB.Model(&role).Update("status", roleStatus).Error; err != nil {
				log.Printf("failed to update role default status: role_id=%d err=%v", role.ID, err)
			}
		}

		// regular users can only use global roles or their own user roles
		if !isAdmin {
			if role.RoleType != "global" {
				if role.UserID == nil || *role.UserID != uid {
					c.JSON(http.StatusForbidden, gin.H{"error": "no permission to use this role"})
					return
				}
			}
		}
	}

	device.RoleID = req.RoleID
	if err := updateDeviceColumns(ac.DB, device.ID, map[string]interface{}{
		"role_id": device.RoleID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to apply role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"device_id": device.ID,
			"role_id":   device.RoleID,
		},
	})
}

// SwitchDeviceRoleByNameInternal internal endpoint: switch device role by name (fuzzy match)
func (ac *AdminController) SwitchDeviceRoleByNameInternal(c *gin.Context) {
	deviceName := strings.TrimSpace(c.Param("device_name"))
	if deviceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device name cannot be empty"})
		return
	}

	var req switchDeviceRoleByNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.RoleName = strings.TrimSpace(req.RoleName)
	if req.RoleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_name cannot be empty"})
		return
	}

	var device models.Device
	if err := ac.DB.Where("device_name = ?", deviceName).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	var roles []models.Role
	if err := ac.DB.
		Where("(role_type = ? OR (role_type = ? AND user_id = ?))", "global", "user", device.UserID).
		Order("sort_order ASC, id ASC").
		Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query role"})
		return
	}

	matchedRole, matchType := matchDeviceRoleByName(req.RoleName, roles)
	if matchedRole == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":               "no matching role found",
			"requested_role_name": req.RoleName,
		})
		return
	}

	roleID := matchedRole.ID
	device.RoleID = &roleID
	if err := updateDeviceColumns(ac.DB, device.ID, map[string]interface{}{
		"role_id": device.RoleID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to switch device role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"device_id":           device.ID,
			"device_name":         device.DeviceName,
			"role_id":             device.RoleID,
			"role_name":           matchedRole.Name,
			"role_type":           matchedRole.RoleType,
			"requested_role_name": req.RoleName,
			"match_type":          matchType,
		},
	})
}

// RestoreDeviceDefaultRoleInternal internal endpoint: restore device default role (clear bound role)
func (ac *AdminController) RestoreDeviceDefaultRoleInternal(c *gin.Context) {
	deviceName := strings.TrimSpace(c.Param("device_name"))
	if deviceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device name cannot be empty"})
		return
	}

	var device models.Device
	if err := ac.DB.Where("device_name = ?", deviceName).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	device.RoleID = nil
	if err := updateDeviceColumns(ac.DB, device.ID, map[string]interface{}{
		"role_id": nil,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore default role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"device_id":   device.ID,
			"device_name": device.DeviceName,
			"role_id":     device.RoleID,
		},
	})
}
