package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
	"xiaozhi-esp32-server-golang/internal/components/http"
	"xiaozhi-esp32-server-golang/internal/domain/config/types"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

var (
	defaultManagerOpenClawEnterKeywords = []string{"open openclaw", "enter openclaw"}
	defaultManagerOpenClawExitKeywords  = []string{"close openclaw", "exit openclaw"}
)

func cloneOpenClawKeywords(keywords []string) []string {
	if len(keywords) == 0 {
		return []string{}
	}
	cloned := make([]string, len(keywords))
	copy(cloned, keywords)
	return cloned
}

func normalizeSpeakerChatMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "identified_only":
		return "identified_only"
	default:
		return "off"
	}
}

// ConfigManager configuration manager
// Provide high-level configuration management functions, including caching, hot update, configuration verification, etc.
type ConfigManager struct {
	//HTTP client
	client *http.ManagerClient
}

// NewConfigManager creates a new configuration manager
func NewManagerUserConfigProvider(config map[string]interface{}) (*ConfigManager, error) {
	//Get the base URL of the backend management system from the configuration
	var baseURL string
	if backendUrl := config["backend_url"]; backendUrl != nil {
		baseURL = backendUrl.(string)
	}
	//If not in configuration, use default value
	if baseURL == "" {
		baseURL = "http://localhost:8080" //Default value
	}

	//Create Manager HTTP client
	authToken := util.GetManagerAuthToken()
	if token, ok := config["auth_token"].(string); ok && strings.TrimSpace(token) != "" {
		authToken = strings.TrimSpace(token)
	}
	managerClient := http.NewManagerClient(http.ManagerClientConfig{
		BaseURL:    baseURL,
		AuthToken:  authToken,
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	})

	manager := &ConfigManager{
		client: managerClient,
	}

	//log.Log().Debug("Configuration Manager initialization successful", "backend_url", baseURL)
	return manager, nil
}

func (c *ConfigManager) GetUserConfig(ctx context.Context, deviceID string) (types.UConfig, error) {
	//Parse response
	var response struct {
		Data struct {
			VAD struct {
				Provider string `json:"provider"`
				JsonData string `json:"json_data"`
			} `json:"vad"`
			ASR struct {
				Provider string `json:"provider"`
				JsonData string `json:"json_data"`
			} `json:"asr"`
			LLM struct {
				Provider string `json:"provider"`
				JsonData string `json:"json_data"`
			} `json:"llm"`
			TTS struct {
				Provider string `json:"provider"`
				JsonData string `json:"json_data"`
			} `json:"tts"`
			Memory struct {
				Provider string `json:"provider"`
				JsonData string `json:"json_data"`
			} `json:"memory"`
			VoiceIdentify map[string]struct {
				ID                 uint     `json:"id"`
				Name               string   `json:"name"`
				Prompt             string   `json:"prompt"`
				Description        string   `json:"description"`
				Uuids              []string `json:"uuids"`
				TTSConfigID        *string  `json:"tts_config_id"`
				Voice              *string  `json:"voice"`
				VoiceModelOverride *string  `json:"voice_model_override"`
			} `json:"voice_identify"`
			KnowledgeBases  []types.KnowledgeBaseRef `json:"knowledge_bases"`
			Prompt          string                   `json:"prompt"`
			AgentId         string                   `json:"agent_id"`
			MemoryMode      string                   `json:"memory_mode"`
			SpeakerChatMode string                   `json:"speaker_chat_mode"`
			MCPServiceNames string                   `json:"mcp_service_names"`
			OpenClaw        struct {
				Allowed       bool     `json:"allowed"`
				EnterKeywords []string `json:"enter_keywords"`
				ExitKeywords  []string `json:"exit_keywords"`
			} `json:"openclaw"`
		} `json:"data"`
	}

	//Send HTTP request
	err := c.client.DoRequest(ctx, http.RequestOptions{
		Method: "GET",
		Path:   "/api/configs",
		QueryParams: map[string]string{
			"device_id": deviceID,
		},
		Response: &response,
	})
	if err != nil {
		log.Log().Error("Failed to obtain user configuration", "error", err, "device_id", deviceID)
		return types.UConfig{}, err
	}

	//Helper functions for parsing JSON configuration data
	parseJsonData := func(jsonStr string) map[string]interface{} {
		var data map[string]interface{}
		if jsonStr != "" {
			if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
				log.Log().Warn("Failed to parse JSON data", "error", err, "json", jsonStr)
				return make(map[string]interface{})
			}
		}
		return data
	}

	//Obtain the voiceprint group information from the device configuration (only obtain the voiceprint group configuration, not the service address)
	//VoiceIdentify is a map, the key is the voiceprint group name, and the value includes prompt, description and uuids
	voiceIdentifyData := make(map[string]types.SpeakerGroupInfo)
	if len(response.Data.VoiceIdentify) > 0 {
		//Convert voiceprint group information in map format to configuration format
		for groupName, groupInfo := range response.Data.VoiceIdentify {
			groupData := types.SpeakerGroupInfo{
				ID:                 groupInfo.ID,
				Name:               groupInfo.Name,
				Prompt:             groupInfo.Prompt,
				Description:        groupInfo.Description,
				Uuids:              groupInfo.Uuids,
				TTSConfigID:        groupInfo.TTSConfigID,
				Voice:              groupInfo.Voice,
				VoiceModelOverride: groupInfo.VoiceModelOverride,
			}
			voiceIdentifyData[groupName] = groupData
		}
	}

	//Build configuration results
	enterKeywords := response.Data.OpenClaw.EnterKeywords
	if len(enterKeywords) == 0 {
		enterKeywords = cloneOpenClawKeywords(defaultManagerOpenClawEnterKeywords)
	}
	exitKeywords := response.Data.OpenClaw.ExitKeywords
	if len(exitKeywords) == 0 {
		exitKeywords = cloneOpenClawKeywords(defaultManagerOpenClawExitKeywords)
	}

	config := types.UConfig{
		SystemPrompt: response.Data.Prompt, //Using custom prompts for agents
		Asr: types.AsrConfig{
			Provider: response.Data.ASR.Provider,
			Config:   parseJsonData(response.Data.ASR.JsonData),
		},
		Tts: types.TtsConfig{
			Provider: response.Data.TTS.Provider,
			Config:   parseJsonData(response.Data.TTS.JsonData),
		},
		Llm: types.LlmConfig{
			Provider: response.Data.LLM.Provider,
			Config:   parseJsonData(response.Data.LLM.JsonData),
		},
		Vad: types.VadConfig{
			Provider: response.Data.VAD.Provider,
			Config:   parseJsonData(response.Data.VAD.JsonData),
		},
		Memory: types.MemoryConfig{
			Provider: response.Data.Memory.Provider,
			Config:   parseJsonData(response.Data.Memory.JsonData),
		},
		KnowledgeBases:  response.Data.KnowledgeBases,
		VoiceIdentify:   voiceIdentifyData,
		MemoryMode:      response.Data.MemoryMode,
		SpeakerChatMode: response.Data.SpeakerChatMode,
		AgentId:         response.Data.AgentId,
		MCPServiceNames: strings.TrimSpace(response.Data.MCPServiceNames),
		OpenClaw: types.OpenClawConfig{
			Allowed:       response.Data.OpenClaw.Allowed,
			EnterKeywords: enterKeywords,
			ExitKeywords:  exitKeywords,
		},
	}
	if strings.TrimSpace(config.MemoryMode) == "" {
		config.MemoryMode = "short"
	}
	config.SpeakerChatMode = normalizeSpeakerChatMode(config.SpeakerChatMode)

	log.Log().Infof("Successfully obtained device configuration: deviceId: %s, config: %+v", deviceID, config)
	return config, nil
}

// Get mqtt, mqtt_server, udp, ota, vision configuration
func (c *ConfigManager) GetSystemConfig(ctx context.Context) (string, error) {
	//Parse response JSON
	var apiResponse struct {
		Data map[string]interface{} `json:"data"`
	}

	//Send HTTP request
	err := c.client.DoRequest(ctx, http.RequestOptions{
		Method:   "GET",
		Path:     "/api/system/configs",
		Response: &apiResponse,
	})
	if err != nil {
		return "", fmt.Errorf("Failed to obtain system configuration: %w", err)
	}

	//Process the voice_identify configuration, making sure to include the threshold field
	if voiceIdentifyData, exists := apiResponse.Data["voice_identify"]; exists {
		if voiceIdentifyMap, ok := voiceIdentifyData.(map[string]interface{}); ok {
			//If the voice_identify configuration exists but does not have a threshold field, add a default value
			if _, hasThreshold := voiceIdentifyMap["threshold"]; !hasThreshold {
				voiceIdentifyMap["threshold"] = 0.4
				log.Log().Info("The voice_identify configuration is missing the threshold field, a default value of 0.4 has been added")
			} else {
				//Validation threshold range
				if thresholdVal, ok := voiceIdentifyMap["threshold"].(float64); ok {
					if thresholdVal < 0 || thresholdVal > 1 {
						log.Log().Warnf("voice_identify.threshold value %.4f is outside the valid range [0.0, 1.0], use default value 0.4", thresholdVal)
						voiceIdentifyMap["threshold"] = 0.4
					}
				}
			}
			//Update configuration data
			apiResponse.Data["voice_identify"] = voiceIdentifyMap
		}
	}
	//log.Debugf("SyObtain system configuration from internal control: %+vion obtained from internal control: %+v", apiResponse.Data)

	//Convert API response to configuration JSON string
	configJSON, err := json.Marshal(apiResponse.Data)
	if err != nil {
		return "", fmt.Errorf("Serialization configuration failed: %w", err)
	}

	return string(configJSON), nil
}

// LoadSystemConfigToViper loads system configuration from backend API and sets it to viper
func (c *ConfigManager) LoadSystemConfigToViper(ctx context.Context) error {
	//Get system configuration JSON string
	configJSON, err := c.GetSystemConfig(ctx)
	if err != nil {
		return fmt.Errorf("Failed to obtain system configuration: %w", err)
	}

	//Set configuration to viper using viper.MergeConfigMap
	//First parse the JSON string into a map
	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
		return fmt.Errorf("Failed to parse configuration JSON: %w", err)
	}

	//Set to viper (need to import viper package)
	// viper.MergeConfigMap(configMap)

	log.Log().Info("System configuration has been successfully loaded into viper", "config_size", len(configJSON))
	return nil
}

// SwitchDeviceRoleByName switches the device role by role name (supports fuzzy matching)
func (c *ConfigManager) SwitchDeviceRoleByName(ctx context.Context, deviceID string, roleName string) (string, error) {
	deviceID = strings.TrimSpace(deviceID)
	roleName = strings.TrimSpace(roleName)
	if deviceID == "" {
		return "", fmt.Errorf("deviceID cannot be empty")
	}
	if roleName == "" {
		return "", fmt.Errorf("roleName cannot be empty")
	}

	var response struct {
		Data struct {
			RoleName string `json:"role_name"`
		} `json:"data"`
		Error string `json:"error"`
	}

	path := fmt.Sprintf("/api/internal/devices/%s/switch-role", url.PathEscape(deviceID))
	err := c.client.DoRequest(ctx, http.RequestOptions{
		Method: "POST",
		Path:   path,
		Body: map[string]string{
			"role_name": roleName,
		},
		Response: &response,
	})
	if err != nil {
		return "", fmt.Errorf("Failed to switch device role: %w", err)
	}
	if response.Error != "" {
		return "", fmt.Errorf(response.Error)
	}
	if strings.TrimSpace(response.Data.RoleName) == "" {
		return "", fmt.Errorf("Failed to switch device role: No matching role returned")
	}
	return response.Data.RoleName, nil
}

// RestoreDeviceDefaultRole Restores the device's default role (clears device-bound roles)
func (c *ConfigManager) RestoreDeviceDefaultRole(ctx context.Context, deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return fmt.Errorf("deviceID cannot be empty")
	}

	var response struct {
		Error string `json:"error"`
	}

	path := fmt.Sprintf("/api/internal/devices/%s/restore-default-role", url.PathEscape(deviceID))
	err := c.client.DoRequest(ctx, http.RequestOptions{
		Method:   "POST",
		Path:     path,
		Response: &response,
	})
	if err != nil {
		return fmt.Errorf("Failed to restore default role: %w", err)
	}
	if response.Error != "" {
		return fmt.Errorf(response.Error)
	}
	return nil
}

// SearchKnowledge uniformly searches the knowledge base through the management background (the console forwards by provider)
func (c *ConfigManager) NotifyDeviceEvent(ctx context.Context, eventType string, eventData map[string]interface{}) {
	_, err := SendDeviceRequest(ctx, eventType, eventData)
	if err != nil {
		log.Log().Error("Failed to send device event", "error", err)
	}
}

func (c *ConfigManager) RegisterMessageEventHandler(ctx context.Context, eventType string, handler types.EventHandler) {
	GetDefaultClient().RegisterMessageHandler(ctx, eventType, handler)
}
