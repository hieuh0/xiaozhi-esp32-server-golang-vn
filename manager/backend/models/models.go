package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// User model
type User struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Username  string    `json:"username" gorm:"type:varchar(50);uniqueIndex:idx_users_username;not null"`
	Password  string    `json:"-" gorm:"type:varchar(255);not null"`
	Email     string    `json:"email" gorm:"type:varchar(100);uniqueIndex:idx_users_email"`
	Role      string    `json:"role" gorm:"type:varchar(20);not null;default:'user'"` // admin, user
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// APIToken is an external OpenAPI access token (hash stored only, no plaintext)
type APIToken struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	UserID      uint       `json:"user_id" gorm:"not null;index"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null"`
	TokenPrefix string     `json:"token_prefix" gorm:"type:varchar(20);index"`
	TokenHash   string     `json:"-" gorm:"type:char(64);uniqueIndex;not null"`
	IsActive    bool       `json:"is_active" gorm:"default:true;index"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at" gorm:"index"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Device model
type Device struct {
	ID           uint       `json:"id" gorm:"primarykey"`
	UserID       uint       `json:"user_id" gorm:"not null"`
	AgentID      uint       `json:"agent_id" gorm:"not null;default:0"`                                       // agent ID; a device can only belong to one agent
	RoleID       *uint      `json:"role_id" gorm:"index"`                                                     // role ID (optional, overrides agent config)
	NickName     string     `json:"nick_name" gorm:"type:varchar(100)"`                                       // device nickname, user-editable
	DeviceCode   string     `json:"device_code" gorm:"type:varchar(100);uniqueIndex:idx_devices_device_code"` // 6-digit activation code
	DeviceName   string     `json:"device_name" gorm:"type:varchar(100)"`                                     // device identifier / Device-ID reported by firmware
	Challenge    string     `json:"challenge" gorm:"type:varchar(128)"`                                       // activation challenge
	PreSecretKey string     `json:"pre_secret_key" gorm:"type:varchar(128)"`                                  // pre-activation secret key
	Activated    bool       `json:"activated" gorm:"default:false"`                                           // whether the device has been activated
	LastActiveAt *time.Time `json:"last_active_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Agent model
type Agent struct {
	ID              uint    `json:"id" gorm:"primarykey"`
	UserID          uint    `json:"user_id" gorm:"not null"`
	Name            string  `json:"name" gorm:"type:varchar(100);not null"`                  // name, used for management identification
	Nickname        string  `json:"nickname" gorm:"type:varchar(100)"`                       // nickname, used in LLM/Prompt
	CustomPrompt    string  `json:"custom_prompt" gorm:"type:text"`                          // role description (prompt)
	LLMConfigID     *string `json:"llm_config_id" gorm:"type:varchar(100)"`                  // LLM config ID
	TTSConfigID     *string `json:"tts_config_id" gorm:"type:varchar(100)"`                  // TTS config ID
	Voice           *string `json:"voice" gorm:"type:varchar(200)"`                          // voice value
	ASRSpeed        string  `json:"asr_speed" gorm:"type:varchar(20);default:'normal'"`      // ASR speed: normal/patient/fast
	MemoryMode      string  `json:"memory_mode" gorm:"type:varchar(20);default:'short'"`     // memory mode: none/short/long
	SpeakerChatMode string  `json:"speaker_chat_mode" gorm:"type:varchar(32);default:'off'"` // voiceprint chat mode: off/identified_only
	MCPServiceNames string  `json:"mcp_service_names" gorm:"type:text"`                      // comma-separated MCP service names; empty = use all enabled global MCP services
	// OpenClaw config as JSON string, structure:
	// {"allowed":true,"enter_keywords":["open openclaw"],"exit_keywords":["exit openclaw"]}
	OpenClawConfig string    `json:"openclaw_config" gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// KnowledgeBase is a per-user knowledge base
type KnowledgeBase struct {
	ID                 uint       `json:"id" gorm:"primarykey"`
	UserID             uint       `json:"user_id" gorm:"not null;index"`
	Name               string     `json:"name" gorm:"type:varchar(100);not null"`
	Description        string     `json:"description" gorm:"type:text"`
	Content            string     `json:"content" gorm:"type:text"`
	RetrievalThreshold *float64   `json:"retrieval_threshold" gorm:"type:double"`         // retrieval threshold (nil = inherit global config)
	ExternalKBID       string     `json:"external_kb_id" gorm:"type:varchar(255);index"`  // external knowledge base ID (Dify dataset_id)
	ExternalDocID      string     `json:"external_doc_id" gorm:"type:varchar(255);index"` // external document ID (Dify document_id)
	AutoDataset        bool       `json:"auto_dataset" gorm:"default:false"`              // whether the dataset was auto-created by the system
	SyncProvider       string     `json:"sync_provider" gorm:"type:varchar(50);index"`    // sync provider (currently dify)
	SyncStatus         string     `json:"sync_status" gorm:"type:varchar(20);default:'pending';index"`
	SyncError          string     `json:"sync_error" gorm:"type:text"`
	LastSyncedAt       *time.Time `json:"last_synced_at"`
	Status             string     `json:"status" gorm:"type:varchar(20);default:'active';index"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// KnowledgeBaseDocument is a document within a knowledge base (one KB can have multiple documents)
type KnowledgeBaseDocument struct {
	ID              uint       `json:"id" gorm:"primarykey"`
	KnowledgeBaseID uint       `json:"knowledge_base_id" gorm:"not null;index"`
	Name            string     `json:"name" gorm:"type:varchar(200);not null"`
	Content         string     `json:"content" gorm:"type:text"`
	ExternalDocID   string     `json:"external_doc_id" gorm:"type:varchar(255);index"` // Dify document_id
	SyncStatus      string     `json:"sync_status" gorm:"type:varchar(20);default:'pending';index"`
	SyncError       string     `json:"sync_error" gorm:"type:text"`
	LastSyncedAt    *time.Time `json:"last_synced_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AgentKnowledgeBase is the many-to-many association between agents and knowledge bases
type AgentKnowledgeBase struct {
	ID              uint      `json:"id" gorm:"primarykey"`
	AgentID         uint      `json:"agent_id" gorm:"not null;index;uniqueIndex:idx_agent_kb_unique,priority:1"`
	KnowledgeBaseID uint      `json:"knowledge_base_id" gorm:"not null;index;uniqueIndex:idx_agent_kb_unique,priority:2"`
	CreatedAt       time.Time `json:"created_at"`
}

// Config is the general configuration model
type Config struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Type      string    `json:"type" gorm:"type:varchar(50);not null;uniqueIndex:type_config_id,priority:1"` // vad, asr, llm, tts, ota, mqtt, udp, mqtt_server, vision
	Name      string    `json:"name" gorm:"type:varchar(100);not null"`
	ConfigID  string    `json:"config_id" gorm:"type:varchar(100);not null;uniqueIndex:type_config_id,priority:2"` // config ID for association
	Provider  string    `json:"provider" gorm:"type:varchar(50)"`                                                  // provider field required for some config types
	JsonData  string    `json:"json_data" gorm:"type:text"`                                                        // JSON config data
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	IsDefault bool      `json:"is_default" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MCPMarketService is an MCP service config imported from the marketplace.
// Manual configs are still stored in Config(type=mcp).json_data; marketplace configs are in this dedicated table.
type MCPMarketService struct {
	ID               uint   `json:"id" gorm:"primarykey"`
	Name             string `json:"name" gorm:"type:varchar(150);not null"`
	Enabled          bool   `json:"enabled" gorm:"default:true;index"`
	Transport        string `json:"transport" gorm:"type:varchar(32);not null"` // sse / streamablehttp
	URL              string `json:"url" gorm:"type:text;not null"`
	URLHash          string `json:"url_hash" gorm:"type:char(64);not null;uniqueIndex:idx_mcp_market_services_url_hash"` // sha256(url) hex
	HeadersJSON      string `json:"headers_json" gorm:"type:text"`
	AllowedToolsJSON string `json:"allowed_tools_json" gorm:"type:text"`

	MarketID    *uint  `json:"market_id" gorm:"index"` // references configs(type=mcp_market).id
	ProviderID  string `json:"provider_id" gorm:"type:varchar(50);index"`
	ServiceID   string `json:"service_id" gorm:"type:varchar(255);index"`
	ServiceName string `json:"service_name" gorm:"type:varchar(255)"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Role model (unified management of global and user roles)
type Role struct {
	ID          uint   `json:"id" gorm:"primarykey"`
	UserID      *uint  `json:"user_id" gorm:"index"` // owning user ID; NULL = global role
	Name        string `json:"name" gorm:"type:varchar(100);not null"`
	Description string `json:"description" gorm:"type:text"`
	Prompt      string `json:"prompt" gorm:"type:text"` // system prompt

	// LLM/TTS config (consistent with Agent fields)
	LLMConfigID *string `json:"llm_config_id" gorm:"type:varchar(100)"` // LLM config ID

	TTSConfigID *string `json:"tts_config_id" gorm:"type:varchar(100)"` // TTS config ID
	Voice       *string `json:"voice" gorm:"type:varchar(200)"`         // voice value

	// Role type and status
	RoleType string `json:"role_type" gorm:"type:varchar(20);default:'user';index"` // global/system/user
	Status   string `json:"status" gorm:"type:varchar(20);default:'active';index"`  // active/inactive

	// Ordering and default
	SortOrder int  `json:"sort_order" gorm:"default:0"`           // display order
	IsDefault bool `json:"is_default" gorm:"default:false;index"` // default role flag (global roles only)

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name
func (Role) TableName() string {
	return "roles"
}

// GlobalRole model (kept for compatibility; can be migrated to Role later)
type GlobalRole struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null"`
	Description string    `json:"description" gorm:"type:text"`
	Prompt      string    `json:"prompt" gorm:"type:text"`
	IsDefault   bool      `json:"is_default" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SpeakerGroup is the voiceprint group model
type SpeakerGroup struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	UserID      uint      `json:"user_id" gorm:"not null;index;uniqueIndex:idx_speaker_groups_user_name,priority:1"`
	AgentID     uint      `json:"agent_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null;uniqueIndex:idx_speaker_groups_user_name,priority:2"`
	Prompt      string    `json:"prompt" gorm:"type:text"`
	Description string    `json:"description" gorm:"type:text"`
	TTSConfigID *string   `json:"tts_config_id" gorm:"type:varchar(100)"` // TTS config ID
	Voice       *string   `json:"voice" gorm:"type:varchar(200)"`         // voice value
	Status      string    `json:"status" gorm:"type:varchar(20);default:'active'"`
	SampleCount int       `json:"sample_count" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SpeakerSample is the voiceprint sample model
type SpeakerSample struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	SpeakerGroupID uint      `json:"speaker_group_id" gorm:"not null;index"`
	UserID         uint      `json:"user_id" gorm:"not null;index"`
	UUID           string    `json:"uuid" gorm:"type:varchar(36);not null;uniqueIndex"`
	FilePath       string    `json:"file_path" gorm:"type:varchar(500);not null"`
	FileName       string    `json:"file_name" gorm:"type:varchar(255)"`
	FileSize       int64     `json:"file_size"`
	Duration       float32   `json:"duration"`
	Status         string    `json:"status" gorm:"type:varchar(20);default:'active'"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// VoiceClone is the cloned voice model
type VoiceClone struct {
	ID                 uint      `json:"id" gorm:"primarykey"`
	UserID             uint      `json:"user_id" gorm:"not null;index"`
	Name               string    `json:"name" gorm:"type:varchar(100);not null"`
	Provider           string    `json:"provider" gorm:"type:varchar(50);not null;index"`
	ProviderVoiceID    string    `json:"provider_voice_id" gorm:"type:varchar(200);not null;index"`
	TTSConfigID        string    `json:"tts_config_id" gorm:"type:varchar(100);not null;index"`
	SharedToAll        bool      `json:"shared_to_all" gorm:"default:false;index"`
	Status             string    `json:"status" gorm:"type:varchar(20);default:'active';index"`
	TranscriptRequired bool      `json:"transcript_required" gorm:"default:false"`
	MetaJSON           string    `json:"meta_json" gorm:"type:json"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// VoiceCloneAudio is the raw audio asset model for voice cloning (stores uploaded/recorded data)
type VoiceCloneAudio struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	VoiceCloneID   *uint     `json:"voice_clone_id" gorm:"index"`
	UserID         uint      `json:"user_id" gorm:"not null;index"`
	SourceType     string    `json:"source_type" gorm:"type:varchar(20);not null"` // upload/record
	FilePath       string    `json:"file_path" gorm:"type:varchar(500);not null"`
	FileName       string    `json:"file_name" gorm:"type:varchar(255)"`
	FileSize       int64     `json:"file_size"`
	ContentType    string    `json:"content_type" gorm:"type:varchar(100)"`
	Transcript     string    `json:"transcript" gorm:"type:text"`
	TranscriptLang string    `json:"transcript_lang" gorm:"type:varchar(20)"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// VoiceCloneTask is the async task model for voice cloning
type VoiceCloneTask struct {
	ID           uint       `json:"id" gorm:"primarykey"`
	TaskID       string     `json:"task_id" gorm:"type:varchar(64);not null;uniqueIndex"`
	UserID       uint       `json:"user_id" gorm:"not null;index"`
	VoiceCloneID uint       `json:"voice_clone_id" gorm:"not null;index"`
	Provider     string     `json:"provider" gorm:"type:varchar(50);not null;index"`
	Status       string     `json:"status" gorm:"type:varchar(20);not null;default:'queued';index"` // queued/processing/succeeded/failed
	Attempts     int        `json:"attempts" gorm:"not null;default:0"`
	LastError    string     `json:"last_error" gorm:"type:text"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	MetaJSON     string     `json:"meta_json" gorm:"type:json"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// UserVoiceCloneQuota tracks voice clone quotas per user per tts_config_id
type UserVoiceCloneQuota struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	UserID      uint      `json:"user_id" gorm:"not null;index;uniqueIndex:idx_user_tts_quota,priority:1"`
	TTSConfigID string    `json:"tts_config_id" gorm:"type:varchar(100);not null;index;uniqueIndex:idx_user_tts_quota,priority:2"`
	MaxCount    int       `json:"max_count" gorm:"not null;default:-1"` // -1 = unlimited, 0 = creation disabled
	UsedCount   int       `json:"used_count" gorm:"not null;default:0"` // incremented each time a clone task is submitted
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ChatMessage is the chat message model
type ChatMessage struct {
	ID        uint   `json:"id" gorm:"primarykey"`
	MessageID string `json:"message_id" gorm:"type:varchar(64);uniqueIndex:idx_chat_messages_message_id;not null"`

	// Association info (no foreign keys)
	DeviceID  string `json:"device_id" gorm:"type:varchar(100);index:idx_device_id;not null"`
	AgentID   string `json:"agent_id" gorm:"type:varchar(64);index:idx_agent_id;not null"`
	UserID    uint   `json:"user_id" gorm:"index:idx_user_id;not null"`
	SessionID string `json:"session_id" gorm:"type:varchar(64);index:idx_session_id"` // grouping marker only

	// Message content
	Role    string `json:"role" gorm:"type:varchar(20);index;not null;comment:user|assistant|system|tool"`
	Content string `json:"content" gorm:"type:text;not null"`

	// Tool call info
	ToolCallID    string  `json:"tool_call_id,omitempty" gorm:"type:varchar(64);index;comment:tool call ID (used by Tool role)"`
	ToolCallsJSON *string `json:"tool_calls_json,omitempty" gorm:"type:json;column:tool_calls;comment:tool call list JSON (used by Assistant role)"`

	// Audio file info (filesystem storage, two-level hash sharding)
	AudioPath     string `json:"audio_path,omitempty" gorm:"type:varchar(512);comment:relative audio file path (two-level hash sharding)"`
	AudioDuration *int   `json:"audio_duration,omitempty" gorm:"comment:milliseconds"`
	AudioSize     *int   `json:"audio_size,omitempty" gorm:"comment:bytes"`
	AudioFormat   string `json:"audio_format,omitempty" gorm:"type:varchar(20);default:'wav';comment:audio format (fixed as wav)"`

	// Metadata
	MetadataJSON string                 `json:"-" gorm:"type:json;column:metadata"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" gorm:"-"`

	// Status
	IsDeleted bool      `json:"is_deleted" gorm:"default:false;index"`
	CreatedAt time.Time `json:"created_at" gorm:"index:idx_created_at"`
}

// TableName specifies the table name
func (ChatMessage) TableName() string {
	return "chat_messages"
}

// BeforeSave GORM hook - serializes metadata
func (m *ChatMessage) BeforeSave(tx *gorm.DB) error {
	if m.Metadata != nil {
		data, err := json.Marshal(m.Metadata)
		if err != nil {
			return err
		}
		m.MetadataJSON = string(data)
	}
	return nil
}

// AfterFind GORM hook - deserializes metadata
func (m *ChatMessage) AfterFind(tx *gorm.DB) error {
	if m.MetadataJSON != "" {
		return json.Unmarshal([]byte(m.MetadataJSON), &m.Metadata)
	}
	return nil
}
