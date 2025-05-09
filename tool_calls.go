package aichat

import (
	"encoding/json"
	"fmt"
)

// ToolCall represents a call to an external tool or function
type ToolCall struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`

	Function Function `json:"function"`
}

// RangePendingToolCalls iterates through messages to find and process tool calls that haven't received a response.
// It performs two passes: first to identify which tool calls have responses, then to process pending calls.
// The provided function is called for each pending tool call.
func (chat *Chat) RangePendingToolCalls(fn func(toolCallContext *ToolCallContext) error) error {
	// Create a map to track which tool calls have responses
	responded := make(map[string]bool)

	// First pass: identify which tool calls have responses
	chat.RangeByRole("tool", func(msg *Message) error {
		if msg.ToolCallID != "" {
			responded[msg.ToolCallID] = true
		}
		return nil
	})

	// Second pass: process pending tool calls
	return chat.RangeByRole("assistant", func(msg *Message) error {
		for _, call := range msg.ToolCalls {
			if responded[call.ID] {
				continue
			}
			if err := fn(&ToolCallContext{
				Chat:     chat,
				ToolCall: &call,
			}); err != nil {
				return err
			}
			responded[call.ID] = true
		}
		return nil
	})
}

// ToolCallContext represents a tool call within a chat context, managing the lifecycle
// of a single tool invocation including its execution and response handling.
type ToolCallContext struct {
	ToolCall *ToolCall
	Chat     *Chat
}

// Name returns the name of the function
func (tcc *ToolCallContext) Name() string {
	return tcc.ToolCall.Function.Name
}

// Arguments returns the arguments to the function as a map
func (tcc *ToolCallContext) Arguments() (map[string]any, error) {
	return tcc.ToolCall.Function.ArgumentsMap()
}

// Return sends the result of the function call back to the chat
func (tcc *ToolCallContext) Return(result map[string]any) error {
	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	tcc.Chat.AddToolContent(tcc.Name(), tcc.ToolCall.ID, string(jsonData))
	return nil
}
