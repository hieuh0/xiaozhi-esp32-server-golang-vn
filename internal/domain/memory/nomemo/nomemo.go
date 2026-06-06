package nomemo

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// NoMemoProvider empty memory provider implementation
// Used when the user does not need the memory function, all methods are empty implementations
type NoMemoProvider struct{}

// Get gets the NoMemoProvider instance
func Get() *NoMemoProvider {
	return &NoMemoProvider{}
}

// AddMessage adds a message to memory (empty implementation)
func (n *NoMemoProvider) AddMessage(ctx context.Context, agentID string, msg schema.Message) error {
	//Empty implementation, does nothing
	return nil
}

// GetMessages gets the user's historical messages (empty implementation)
func (n *NoMemoProvider) GetMessages(ctx context.Context, agentId string, count int) ([]*schema.Message, error) {
	//Returns an empty message list
	return []*schema.Message{}, nil
}

// GetContext Gets the user's context information (empty implementation)
func (n *NoMemoProvider) GetContext(ctx context.Context, agentId string, maxToken int) (string, error) {
	//Return empty string
	return "", nil
}

// Search searches the user's memory (empty implementation)
func (n *NoMemoProvider) Search(ctx context.Context, agentId string, query string, topK int, timeRangeDays int64) (string, error) {
	//Return empty string
	return "", nil
}

// Flush refreshes the user's memory (empty implementation)
func (n *NoMemoProvider) Flush(ctx context.Context, agentId string) error {
	//Empty implementation, does nothing
	return nil
}

// ResetMemory resets the user's memory (empty implementation)
func (n *NoMemoProvider) ResetMemory(ctx context.Context, agentId string) error {
	//Empty implementation, does nothing
	return nil
}
