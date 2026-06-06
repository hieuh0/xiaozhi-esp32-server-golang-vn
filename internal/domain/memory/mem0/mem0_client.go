package mem0

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/schema"
	"github.com/hackers365/mem0-go/client"
	"github.com/hackers365/mem0-go/types"

	log "xiaozhi-esp32-server-golang/logger"
)

// Mem0Client implements the MemoryProvider and EnhancedMemoryProvider interfaces
type Mem0Client struct {
	client          *client.MemoryClient
	config          Mem0Config
	mu              sync.RWMutex
	EnableSearch    bool    `mapstructure:"enable_search"`
	SearchThreshold float64 `mapstructure:"search_threshold"`
	SearchTopk      int     `mapstructure:"search_topk"`
}

// Mem0Config configuration structure
type Mem0Config struct {
	APIKey           string `mapstructure:"api_key"`
	BaseUrl          string `mapstructure:"base_url"`
	OrganizationName string `mapstructure:"organization_name"`
	ProjectName      string `mapstructure:"project_name"`
	OrganizationID   string `mapstructure:"organization_id"`
	ProjectID        string `mapstructure:"project_id"`
}

var (
	mem0Instance *Mem0Client
	mem0Once     sync.Once
	configOnce   sync.Once
)

// GetMem0ClientWithConfig uses the configuration to obtain the Mem0 client singleton
func GetMem0ClientWithConfig(config map[string]interface{}) (*Mem0Client, error) {
	var err error
	configOnce.Do(func() {
		var enableSearch bool = true
		var searchThreshold float64 = 0.5
		var searchTopk int = 3
		//Parse configuration into structure
		var mem0Cfg Mem0Config

		if enableSearchInterface, exists := config["enable_search"]; exists {
			if iEnableSearch, ok := enableSearchInterface.(bool); ok {
				enableSearch = iEnableSearch
			}
		}

		if searchThresholdInterface, exists := config["search_threshold"]; exists {
			if iSearchThreshold, ok := searchThresholdInterface.(float64); ok {
				searchThreshold = iSearchThreshold
			}
		}

		if searchTopkInterface, exists := config["search_topk"]; exists {
			if iSearchTopk, ok := searchTopkInterface.(int); ok {
				searchTopk = iSearchTopk
			}
		}

		//Read API Key
		if apiKeyInterface, exists := config["api_key"]; exists {
			if apiKey, ok := apiKeyInterface.(string); ok {
				mem0Cfg.APIKey = apiKey
			} else {
				err = fmt.Errorf("mem0.api_key must be a string")
				return
			}
		}

		//Read Host
		if hostInterface, exists := config["base_url"]; exists {
			if host, ok := hostInterface.(string); ok {
				mem0Cfg.BaseUrl = host
			} else {
				err = fmt.Errorf("mem0.host must be a string")
				return
			}
		}

		//Verify necessary configuration
		if mem0Cfg.APIKey == "" {
			err = fmt.Errorf("mem0.api_key configuration is missing or empty")
			return
		}

		//Set default value
		if mem0Cfg.BaseUrl == "" {
			mem0Cfg.BaseUrl = "https://api.mem0.ai"
		}

		//Create mem0 client
		clientOptions := client.ClientOptions{
			APIKey: mem0Cfg.APIKey,
			/*Host:             mem0Cfg.Host,
			OrganizationName: mem0Cfg.OrganizationName,
			ProjectName:      mem0Cfg.ProjectName,
			OrganizationID:   mem0Cfg.OrganizationID,
			ProjectID:        mem0Cfg.ProjectID,*/
		}

		mem0Client, clientErr := client.NewMemoryClient(clientOptions)
		if clientErr != nil {
			err = fmt.Errorf("failed to create mem0 client: %w", clientErr)
			return
		}

		mem0Instance = &Mem0Client{
			client:          mem0Client,
			config:          mem0Cfg,
			EnableSearch:    enableSearch,
			SearchThreshold: searchThreshold,
			SearchTopk:      searchTopk,
		}

		log.Log().Infof("Mem0 client initialization successful, base_url: %s", mem0Cfg.BaseUrl)
	})

	return mem0Instance, err
}

// Init initializes the client
func (m *Mem0Client) Init() error {
	//The client has been initialized on creation
	log.Log().Info("Mem0 client initialized successfully")
	return nil
}

// Get gets the memory (internal method)
func (m *Mem0Client) Get(userID string) (interface{}, error) {
	//Search all memories of user
	results, err := m.client.Search("", &types.SearchOptions{
		MemoryOptions: types.MemoryOptions{
			UserID: userID,
		},
		Limit: 100, //Get more memories
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search memories for user %s: %w", userID, err)
	}

	return results, nil
}

// AddMessage adds a message to memory
func (m *Mem0Client) AddMessage(ctx context.Context, agentID string, msg schema.Message) error {
	message := types.Message{
		Role:    string(msg.Role),
		Content: msg.Content,
	}
	//add memory
	_, err := m.client.Add([]types.Message{message}, types.MemoryOptions{
		AgentID:   agentID,
		AsyncMode: true,
	})
	if err != nil {
		return fmt.Errorf("failed to add message to mem0 for user %s: %w", agentID, err)
	}

	log.Log().Debugf("Added message to mem0 for user %s: %s", agentID, message)
	return nil
}

// GetMessages Gets the user's message history
func (m *Mem0Client) GetMessages(ctx context.Context, agentID string, count int) ([]*schema.Message, error) {
	var memoryOptions = types.MemoryOptions{
		AgentID: agentID,
	}

	results, err := m.client.GetAll(&types.SearchOptions{
		MemoryOptions: memoryOptions,
		Limit:         count,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get messages for user %s: %w", agentID, err)
	}

	//Convert to schema.Message format
	var messages []*schema.Message
	for _, result := range results {
		//Extract role and content from metadata
		role := schema.Assistant //Default role
		content := result.Memory

		if result.Metadata != nil {
			if r, ok := result.Metadata["role"].(string); ok {
				switch r {
				case "user":
					role = schema.User
				case "assistant":
					role = schema.Assistant
				case "system":
					role = schema.System
				}
			}
			if c, ok := result.Metadata["content"].(string); ok {
				content = c
			}
		}

		messages = append(messages, &schema.Message{
			Role:    role,
			Content: content,
		})
	}

	return messages, nil
}

// ResetMemory resets user memory
func (m *Mem0Client) ResetMemory(ctx context.Context, userID string) error {

	//Delete all user's memories
	err := m.client.DeleteUser(userID)
	if err != nil {
		return fmt.Errorf("failed to reset memory for user %s: %w", userID, err)
	}

	log.Log().Infof("Reset memory for user %s", userID)
	return nil
}

// GetContext Gets the context (implements the EnhancedMemoryProvider interface)
func (m *Mem0Client) GetContext(ctx context.Context, agentID string, maxToken int) (string, error) {
	return "", nil
}

func (m *Mem0Client) IsEnableSearch() bool {
	return m.EnableSearch
}

func (m *Mem0Client) Search(ctx context.Context, agentId string, query string, topK int, timeRangeDays int64) (string, error) {
	if !m.EnableSearch {
		return "", nil
	}
	topK = m.SearchTopk
	results, err := m.actionSearch(ctx, agentId, query, topK, m.SearchThreshold)
	if err != nil {
		return "", err
	}

	//Build context string
	var msgList []string
	for _, result := range results {
		msgList = append(msgList, fmt.Sprintf("- %s [%s]", result.Memory, result.CreatedAt))
	}

	return strings.Join(msgList, "\n"), nil
}

func (m *Mem0Client) Flush(ctx context.Context, agentID string) error {
	return nil
}

func (m *Mem0Client) actionSearch(ctx context.Context, agentID string, query string, topK int, threshold float64) ([]types.Memory, error) {
	//Search related memories
	results, err := m.client.Search(query, &types.SearchOptions{
		MemoryOptions: types.MemoryOptions{
			AgentID: agentID,
		},
		Limit:     topK,      //Get topK memories
		Threshold: threshold, //Set similarity threshold
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get context for user %s: %w", agentID, err)
	}

	log.Log().Debugf("Successfully obtained context from mem0, agentID: %s, results length: %d", agentID, len(results))
	return results, nil
}

// AddBatchMessages Add messages in batches
func (m *Mem0Client) AddBatchMessages(ctx context.Context, agentID string, messages []schema.Message) error {

	//Prepare batch messages
	var batchMessages []string
	for _, msg := range messages {
		message := fmt.Sprintf("%s: %s", msg.Role, msg.Content)
		batchMessages = append(batchMessages, message)
	}

	//Add memories one by one (mem0-go may not support batch addition)
	for _, message := range batchMessages {
		_, err := m.client.Add(message, types.MemoryOptions{
			AgentID: agentID,
			Metadata: map[string]interface{}{
				"source": "xiaozhi-esp32",
				"batch":  true,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to add batch message to mem0 for user %s: %w", agentID, err)
		}
	}

	log.Log().Debugf("Added %d batch messages to mem0 for user %s", len(messages), agentID)
	return nil
}

// Close Close the client
func (m *Mem0Client) Close() error {
	//mem0-go client does not need to be shut down explicitly
	log.Log().Info("Mem0 client closed")
	return nil
}

//Make sure Mem0Client implements the required interfaces
//Note: The memory package cannot be referenced directly here because it will cause a circular import.
//Interface implementation is automatically checked at compile time
