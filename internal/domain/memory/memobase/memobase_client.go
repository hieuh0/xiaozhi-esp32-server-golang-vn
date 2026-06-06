package memobase

import (
	"context"
	"fmt"
	"strings"
	"sync"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/memodb-io/memobase/src/client/memobase-go/blob"
	"github.com/memodb-io/memobase/src/client/memobase-go/core"
)

var (
	clientInstance *MemobaseClient
	once           sync.Once
	configOnce     sync.Once
	//Use fixed namespace UUID, UUID v5 for generating device IDs
	//This way the same device ID is always mapped to the same UUID
	deviceNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") //DNS namespace
)

// MemobaseClient Memobase client manager
type MemobaseClient struct {
	client *core.MemoBaseClient
	users  sync.Map //Caching user objects
	sync.RWMutex
	EnableSearch    bool
	SearchThreshold float64
	SearchTopk      int
}

// GetWithConfig uses configuration to get the Memobase client instance (singleton mode)
func GetWithConfig(config map[string]interface{}) (*MemobaseClient, error) {
	var initErr error
	configOnce.Do(func() {
		iClient := &MemobaseClient{
			users: sync.Map{},
		}
		//Read memobase related configuration from configuration // Read necessary configuration items
		projectUrlInterface, ok := config["base_url"]
		if !ok {
			initErr = fmt.Errorf("memobase.base_url configuration is missing")
			return
		}
		baseUrl, ok := projectUrlInterface.(string)
		if !ok {
			initErr = fmt.Errorf("memobase.base_url must be a string")
			return
		}

		apiKeyInterface, ok := config["api_key"]
		if !ok {
			initErr = fmt.Errorf("memobase.api_key configuration is missing")
			return
		}
		apiKey, ok := apiKeyInterface.(string)
		if !ok {
			initErr = fmt.Errorf("memobase.api_key must be a string")
			return
		}

		if baseUrl == "" || apiKey == "" {
			initErr = fmt.Errorf("Memobase configuration is incomplete: base_url or api_key is empty")
			log.Log().Errorf("Memobase initialization failed: %v", initErr)
			return
		}

		//Read optional search configuration
		enableSearchInterface, ok := config["enable_search"]
		if ok {
			enableSearch, ok := enableSearchInterface.(bool)
			if ok {
				iClient.EnableSearch = enableSearch
			}
		}

		thresholdInterface, ok := config["search_threshold"]
		if ok {
			threshold, ok := thresholdInterface.(float64)
			if ok {
				iClient.SearchThreshold = threshold
			}
		}

		topKInterface, ok := config["search_topk"]
		if ok {
			topK, ok := topKInterface.(int)
			if ok {
				iClient.SearchTopk = topK
			}
		}

		//Create client
		client, err := core.NewMemoBaseClient(baseUrl, apiKey)
		if err != nil {
			initErr = fmt.Errorf("Failed to create Memobase client: %v", err)
			log.Log().Errorf("Memobase initialization failed: %v", initErr)
			return
		}

		iClient.client = client
		clientInstance = iClient

		log.Log().Infof("Memobase client initialized successfully, project_url: %s", baseUrl)
	})

	if initErr != nil {
		return nil, initErr
	}
	return clientInstance, nil
}

// deviceIDToUUID Convert device ID to UUID v5 format
// Use UUID v5 to ensure the same device ID always generates the same UUID
func deviceIDToUUID(deviceID string) string {
	return uuid.NewSHA1(deviceNamespace, []byte(deviceID)).String()
}

func IsEnableSearch() bool {
	return clientInstance.EnableSearch
}

// AddMessage adds a message to Memobase
func (m *MemobaseClient) AddMessage(ctx context.Context, agentID string, msg schema.Message) error {
	memobaseUserID := deviceIDToUUID(agentID)
	//Build message
	messages := []blob.OpenAICompatibleMessage{
		{
			Role:    string(msg.Role),
			Content: msg.Content,
		},
	}

	//If there is a tool call, add it to the message
	if len(msg.ToolCalls) > 0 {
		return nil
		/*for _, toolCall := range msg.ToolCalls {
			messages = append(messages, blob.OpenAICompatibleMessage{
				Role:    "tool",
				Content: fmt.Sprintf("Tool: %s, Args: %v", toolCall.Function.Name, toolCall.Function.Arguments),
			})
		}*/
	}

	//Create ChatBlob
	chatBlob := &blob.ChatBlob{
		BaseBlob: blob.BaseBlob{
			Type: blob.ChatType,
		},
		Messages: messages,
	}

	//Get or create a user instance (using userID in UUID format)
	user, err := m.getUser(memobaseUserID)
	if err != nil {
		log.Log().Errorf("Failed to obtain or create user, agentID: %s, memobaseUserID: %s, error: %v", agentID, memobaseUserID, err)
		return fmt.Errorf("Failed to obtain or create user: %v", err)
	}

	//Insert message (asynchronous)
	blobID, err := user.Insert(chatBlob, false)
	if err != nil {
		log.Log().Errorf("Failed to add message to Memobase, deviceID: %s, error: %v", agentID, err)
		return fmt.Errorf("Failed to add message to Memobase: %v", err)
	}

	//user.Flush(blob.ChatType, false)

	log.Log().Debugf("Successfully added message to Memobase, deviceID: %s, blobID: %s", agentID, blobID)
	return nil
}

func (m *MemobaseClient) Flush(ctx context.Context, agentID string) error {
	memobaseUserID := deviceIDToUUID(agentID)
	user, err := m.getUser(memobaseUserID)
	if err != nil {
		log.Log().Errorf("Failed to refresh user memory, agentID: %s, memobaseUserID: %s, error: %v", agentID, memobaseUserID, err)
		return fmt.Errorf("Failed to refresh user memory: %v", err)
	}
	user.Flush(blob.ChatType, false)
	return nil
}

// GetContext Gets user context
func (m *MemobaseClient) GetContext(ctx context.Context, agentID string, maxToken int) (string, error) {

	//Convert device ID to UUID format (required by Memobase)
	memobaseUserID := deviceIDToUUID(agentID)

	//Get user instance (does not perform HTTP GET request, only creates instance)
	user, err := m.getUser(memobaseUserID)
	if err != nil {
		log.Log().Errorf("Failed to obtain user instance, agentID: %s, memobaseUserID: %s, error: %v", agentID, memobaseUserID, err)
		return "", fmt.Errorf("Failed to obtain user instance: %v", err)
	}

	//Get context, use default options
	context, err := user.Context(&core.ContextOptions{
		MaxTokenSize: maxToken,
	})
	if err != nil {
		log.Log().Errorf("Failed to obtain context from Memobase, agentID: %s, memobaseUserID: %s, error: %v", agentID, memobaseUserID, err)
		return "", fmt.Errorf("Failed to get context from Memobase: %v", err)
	}

	log.Log().Debugf("Successfully obtained context from Memobase, agentID: %s, context length: %d", agentID, len(context))
	return context, nil
}

func (m *MemobaseClient) Search(ctx context.Context, agentID string, query string, topK int, timeRangeDays int64) (string, error) {
	if !m.EnableSearch {
		return "", nil
	}
	topK = m.SearchTopk
	//Convert device ID to UUID format (required by Memobase)
	memobaseUserID := deviceIDToUUID(agentID)

	//Get user instance (does not perform HTTP GET request, only creates instance)
	user, err := m.getUser(memobaseUserID)
	if err != nil {
		log.Log().Errorf("Failed to obtain user instance, agentID: %s, memobaseUserID: %s, error: %v", agentID, memobaseUserID, err)
		return "", fmt.Errorf("Failed to obtain user instance: %v", err)
	}

	topK = 2

	//Search event
	userEventList, err := user.SearchEvent(query, topK, 0.2, int(timeRangeDays))
	if err != nil {
		log.Log().Errorf("Searching events from Memobase failed, agentID: %s, error: %v", agentID, err)
		return "", fmt.Errorf("Searching events from Memobase failed: %v", err)
	}

	var eventList []string
	for _, event := range userEventList {
		eventList = append(eventList, fmt.Sprintf("- %s: %s", event.CreatedAt, event.EventData.EventTip))
	}

	//Convert to string
	userEventStr := strings.Join(eventList, "\n")

	log.Log().Debugf("Successfully searched events from Memobase, agentID: %s, number of events: %d", agentID, len(eventList))
	return userEventStr, nil
}

// AddBatchMessages adds messages to Memobase in batches
func (m *MemobaseClient) AddBatchMessages(ctx context.Context, userID string, messages []schema.Message) error {
	m.Lock()
	defer m.Unlock()

	if len(messages) == 0 {
		return nil
	}

	//Convert message format
	blobMessages := make([]blob.OpenAICompatibleMessage, 0, len(messages))
	for _, msg := range messages {
		blobMessages = append(blobMessages, blob.OpenAICompatibleMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	//Create ChatBlob
	chatBlob := &blob.ChatBlob{
		BaseBlob: blob.BaseBlob{
			Type: blob.ChatType,
		},
		Messages: blobMessages,
	}

	//Convert device ID to UUID format (required by Memobase)
	memobaseUserID := deviceIDToUUID(userID)

	//Get or create a user instance (using userID in UUID format)
	user, err := m.getUser(userID)
	if err != nil {
		log.Log().Errorf("Add messages in batches: Failed to obtain or create users, deviceID: %s, memobaseUserID: %s, error: %v", userID, memobaseUserID, err)
		return fmt.Errorf("Failed to obtain or create user: %v", err)
	}

	//Insert message (asynchronous)
	blobID, err := user.Insert(chatBlob, false)
	if err != nil {
		log.Log().Errorf("Failed to add messages to Memobase in batches, deviceID: %s, error: %v", userID, err)
		return fmt.Errorf("Failed to add messages to Memobase in batches: %v", err)
	}

	log.Log().Debugf("Successfully added %d messages to Memobase in batches, deviceID: %s, blobID: %s", len(messages), userID, blobID)
	return nil
}

// GetMessages Gets the user's historical messages
// Implement the BaseMemoryProvider interface
// Note: Memobase is mainly used for long-term memory and context enhancement, and does not provide historical message retrieval function.
func (m *MemobaseClient) GetMessages(ctx context.Context, agentID string, count int) ([]*schema.Message, error) {
	return []*schema.Message{}, nil
}

// ResetMemory resets the user's memory
// Implement the MemoryProvider interface
// Note: Memobase's memory reset requires deleting user data through the API
func (m *MemobaseClient) ResetMemory(ctx context.Context, userID string) error {
	//TODO: If Memobase SDK provides an interface for deleting user data, call it here
	//Currently returns nil to indicate successful operation (even if no actual deletion was performed)
	log.Log().Infof("Memobase reset memory request: userID=%s (Note: Memobase does not support direct reset)", userID)
	return nil
}

// Close closes the client (if necessary)
func (m *MemobaseClient) Close() error {
	log.Log().Info("Memobase client is closed")
	return nil
}

// todo plus user object cache
func (m *MemobaseClient) getUser(userID string) (*core.User, error) {
	if user, ok := m.users.Load(userID); ok {
		return user.(*core.User), nil
	}

	memobaseUserID := deviceIDToUUID(userID)
	user, err := m.client.GetOrCreateUser(memobaseUserID)
	if err != nil {
		return nil, fmt.Errorf("Failed to obtain user instance: %v", err)
	}

	m.users.Store(userID, user)
	return user, nil
}
