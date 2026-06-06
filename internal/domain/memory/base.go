package memory

import (
	"context"
	"fmt"

	"xiaozhi-esp32-server-golang/internal/domain/memory/mem0"
	"xiaozhi-esp32-server-golang/internal/domain/memory/memobase"
	"xiaozhi-esp32-server-golang/internal/domain/memory/memos"
	"xiaozhi-esp32-server-golang/internal/domain/memory/nomemo"

	"github.com/cloudwego/eino/schema"
)

// MemoryProvider memory provider interface
// Define the core methods that all memory providers need to implement
type MemoryProvider interface {
	//AddMessage adds a message to memory
	AddMessage(ctx context.Context, agentID string, msg schema.Message) error

	//GetMessages Gets the user's historical messages
	GetMessages(ctx context.Context, agentId string, count int) ([]*schema.Message, error)

	//GetContext obtains the user's contextual information to enhance the LLM prompt
	GetContext(ctx context.Context, agentId string, maxToken int) (string, error)

	//Search Search the user's memory
	Search(ctx context.Context, agentId string, query string, topK int, timeRangeDays int64) (string, error)

	//Flush refreshes the user's memory
	Flush(ctx context.Context, agentId string) error

	//ResetMemory resets the user's memory
	ResetMemory(ctx context.Context, agentId string) error
}

// MemoryType memory type
type MemoryType string

const (
	MemoryTypeNone     MemoryType = "nomemo"
	MemoryTypeMemobase MemoryType = "memobase" //Memobase long-term memory
	MemoryTypeMem0     MemoryType = "mem0"     //Mem0 memory service
	MemoryTypeMemOS    MemoryType = "memos"    //MemOS (compatible with Mem0 API)
)

// GetProvider Gets the memory provider of the specified type
func GetProvider(memoryType MemoryType, config map[string]interface{}) (MemoryProvider, error) {
	return GetProviderByType(memoryType, config)
}

// GetProviderByType Gets the memory provider based on type
func GetProviderByType(memoryType MemoryType, config map[string]interface{}) (MemoryProvider, error) {
	if memoryType == "" {
		memoryType = MemoryTypeNone
	}
	switch memoryType {
	case MemoryTypeNone:
		return nomemo.Get(), nil
	case MemoryTypeMemobase:
		return memobase.GetWithConfig(config)
	case MemoryTypeMem0:
		return mem0.GetMem0ClientWithConfig(config)
	case MemoryTypeMemOS:
		return memos.GetWithConfig(config)
	default:
		return nil, fmt.Errorf("unsupported memory type: %v", memoryType)
	}
}
