package llm_memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	i_redis "xiaozhi-esp32-server-golang/internal/db/redis"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/spf13/viper"

	"github.com/redis/go-redis/v9"
)

var (
	memoryInstance *Memory
	once           sync.Once
	configOnce     sync.Once
)

// Memory represents dialogue memory
type Memory struct {
	redisClient *redis.Client
	keyPrefix   string
	sync.RWMutex
}

// Get gets the memory instance
func Get() *Memory {
	if memoryInstance == nil {
		once.Do(func() {
			redisInstance := i_redis.GetClient()

			memoryInstance = &Memory{
				redisClient: redisInstance,
				keyPrefix:   viper.GetString("redis.key_prefix"),
			}
		})
	}
	return memoryInstance
}

// GetWithConfig uses the configuration to get the memory instance (singleton mode)
func GetWithConfig(config map[string]interface{}) (*Memory, error) {
	var initErr error
	configOnce.Do(func() {
		//Read redis related configuration from configuration
		redisConfig, ok := config["redis"]
		if !ok {
			initErr = fmt.Errorf("redis configuration does not exist")
			return
		}

		redisConfigMap, ok := redisConfig.(map[string]interface{})
		if !ok {
			initErr = fmt.Errorf("redis configuration format error")
			return
		}

		//Read key_prefix configuration
		var keyPrefix string
		if keyPrefixInterface, exists := redisConfigMap["key_prefix"]; exists {
			if kp, ok := keyPrefixInterface.(string); ok {
				keyPrefix = kp
			} else {
				initErr = fmt.Errorf("redis.key_prefix must be a string")
				return
			}
		} else {
			keyPrefix = "xiaozhi:" //Default value
		}

		//Obtain the Redis client (the existing Redis client acquisition method is still used here)
		//Because the initialization of the Redis client is relatively complicated, we will keep the current method for the time being.
		redisClient := i_redis.GetClient()
		if redisClient == nil {
			initErr = fmt.Errorf("Unable to get Redis client")
			return
		}

		//Create LLM memory instance
		memoryInstance = &Memory{
			redisClient: redisClient,
			keyPrefix:   keyPrefix,
		}

		log.Log().Infof("LLM memory initialization successful, key_prefix: %s", keyPrefix)
	})

	if initErr != nil {
		return nil, initErr
	}
	return memoryInstance, nil
}

// NewWithConfig creates a new LLM memory instance using configuration
func NewWithConfig(config map[string]interface{}) (*Memory, error) {
	//Read redis related configuration from configuration
	redisConfig, ok := config["redis"]
	if !ok {
		return nil, fmt.Errorf("redis configuration does not exist")
	}

	redisConfigMap, ok := redisConfig.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("redis configuration format error")
	}

	//Read key_prefix configuration
	var keyPrefix string
	if keyPrefixInterface, exists := redisConfigMap["key_prefix"]; exists {
		if kp, ok := keyPrefixInterface.(string); ok {
			keyPrefix = kp
		} else {
			return nil, fmt.Errorf("redis.key_prefix must be a string")
		}
	} else {
		keyPrefix = "xiaozhi:" //Default value
	}

	//Obtain the Redis client (the existing Redis client acquisition method is still used here)
	//Because the initialization of the Redis client is relatively complicated, we will keep the current method for the time being.
	redisClient := i_redis.GetClient()
	if redisClient == nil {
		return nil, fmt.Errorf("Unable to get Redis client")
	}

	//Create LLM memory instance
	llmMemory := &Memory{
		redisClient: redisClient,
		keyPrefix:   keyPrefix,
	}

	log.Log().Infof("LLM memory initialization successful, key_prefix: %s", keyPrefix)
	return llmMemory, nil
}

// NewMemory creates a new memory instance (for testing only)
func NewMemory(redisClient *redis.Client) *Memory {
	return &Memory{
		redisClient: redisClient,
	}
}

// getMemoryKey generates the Redis key corresponding to the device
func (m *Memory) getMemoryKey(deviceID string) string {
	return fmt.Sprintf("%s:llm:%s", m.keyPrefix, deviceID)
}

// getSystemPromptKey generates the Redis key of the system prompt corresponding to the device
func (m *Memory) getSystemPromptKey(deviceID string) string {
	return fmt.Sprintf("%s:llm:system:%s", m.keyPrefix, deviceID)
}

// AddMessage adds a new conversation message to memory
func (m *Memory) AddMessage(ctx context.Context, deviceID string, agentID string, msg schema.Message) error {
	if m.redisClient == nil {
		log.Log().Warn("redis client is nil")
		return nil
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	key := m.getMemoryKey(deviceID)
	//Use nanosecond timestamp as score
	//ZREVRANGE will return the results from largest to smallest scores
	score := float64(time.Now().UnixNano())

	log.Debugf("Add message to memory: %s, %s", key, string(msgBytes))

	return m.redisClient.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: string(msgBytes),
	}).Err()
}

// GetMessages Gets all conversation memories of the device
func (m *Memory) GetMessages(ctx context.Context, deviceID string, agentID string, count int) ([]*schema.Message, error) {
	if m.redisClient == nil {
		log.Log().Warn("redis client is nil")
		return []*schema.Message{}, nil
	}

	key := m.getMemoryKey(deviceID)

	if count == 0 {
		count = 10
	}

	//Use ZREVRANGE to get the latest N messages
	//The score (timestamp) is larger first, so the order needs to be reversed to ensure that the old message is first.
	startIndex := int64(-(count))
	results, err := m.redisClient.ZRange(ctx, key, startIndex, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("get messages failed: %w", err)
	}

	//pre-allocated slices
	messages := make([]*schema.Message, 0)

	for i := 0; i < len(results); i++ {
		msg := schema.Message{}
		if err := json.Unmarshal([]byte(results[i]), &msg); err != nil {
			return nil, fmt.Errorf("unmarshal message failed: %w", err)
		}

		messages = append(messages, &msg)
	}

	return messages, nil
}

// GetMessagesForLLM Gets message formats for LLM
func (m *Memory) GetMessagesForLLM(ctx context.Context, deviceID string, count int) ([]*schema.Message, error) {
	if m.redisClient == nil {
		log.Log().Warn("redis client is nil")
		return []*schema.Message{}, nil
	}

	//Get historical messages (already in chronological order: old -> new)
	memoryMessages, err := m.GetMessages(ctx, deviceID, "", count)
	if err != nil {
		return nil, err
	}

	return memoryMessages, nil
}

// SetSystemPrompt sets or updates the system prompt of the device
func (m *Memory) SetSystemPrompt(ctx context.Context, deviceID string, prompt string) error {
	if m.redisClient == nil {
		log.Log().Warn("redis client is nil")
		return nil
	}

	key := m.getSystemPromptKey(deviceID)
	return m.redisClient.Set(ctx, key, prompt, 0).Err()
}

// GetSystemPrompt Gets the system prompt of the device
func (m *Memory) GetSystemPrompt(ctx context.Context, deviceID string) (schema.Message, error) {
	if m.redisClient == nil {
		log.Log().Warn("redis client is nil")
		return schema.Message{Role: schema.System, Content: viper.GetString("system_prompt")}, nil
	}

	key := m.getSystemPromptKey(deviceID)

	result, err := m.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return schema.Message{}, nil //Returns an empty message structure
	}
	if err != nil {
		return schema.Message{}, fmt.Errorf("get system prompt failed: %w", err)
	}

	return schema.Message{
		Role:    schema.System,
		Content: result,
	}, nil
}

// ResetMemory resets the device's conversation memory (including system prompts)
func (m *Memory) ResetMemory(ctx context.Context, deviceID string) error {
	if m.redisClient == nil {
		log.Log().Warn("redis client is nil")
		return nil
	}

	//Delete conversation history
	historyKey := m.getMemoryKey(deviceID)
	if err := m.redisClient.Del(ctx, historyKey).Err(); err != nil {
		return fmt.Errorf("delete history failed: %w", err)
	}

	return nil
}

// GetLastNMessages Gets the latest N messages
func (m *Memory) GetLastNMessages(ctx context.Context, deviceID string, n int64) ([]schema.Message, error) {
	if m.redisClient == nil {
		log.Log().Warn("redis client is nil")
		return []schema.Message{}, nil
	}

	key := m.getMemoryKey(deviceID)

	//Get the last N messages
	results, err := m.redisClient.ZRevRange(ctx, key, 0, n-1).Result()
	if err != nil {
		return nil, fmt.Errorf("get last messages failed: %w", err)
	}

	messages := make([]schema.Message, 0, len(results))
	for i := len(results) - 1; i >= 0; i-- { //Reverse order to maintain chronological order
		var msg schema.Message
		if err := json.Unmarshal([]byte(results[i]), &msg); err != nil {
			return nil, fmt.Errorf("unmarshal message failed: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// RemoveOldMessages deletes messages before the specified time
func (m *Memory) RemoveOldMessages(ctx context.Context, deviceID string, before time.Time) error {
	if m.redisClient == nil {
		log.Log().Warn("redis client is nil")
		return nil
	}

	key := m.getMemoryKey(deviceID)
	score := float64(before.UnixNano())

	return m.redisClient.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%f", score)).Err()
}

// Summary Get a summary of the conversation
func (m *Memory) GetSummary(ctx context.Context, deviceID string) (string, error) {
	return "", nil
}

// SetSummary sets the summary of the conversation
func (m *Memory) SetSummary(ctx context.Context, deviceID string, summary string) error {
	return nil
}

// summarize
func (m *Memory) Summary(ctx context.Context, deviceID string, msgList []schema.Message) (string, error) {
	return "", nil
}

func (m *Memory) GetContext(ctx context.Context, deviceID string, agentID string, maxToken int) (string, error) {
	return "", nil
}

func (m *Memory) Search(ctx context.Context, deviceID string, query string, topK int, timeRangeDays int64) (string, error) {
	return "", nil
}
