package structout

import (
	"fmt"

	"github.com/firebase/genkit/go/ai"
)

// WithSystem builds a system GenerateOption with tool-call instructions appended.
func WithSystem[T any](tool *ai.ToolDef[T, any], text string, args ...any) ai.GenerateOption {
	return ai.WithSystem(text+getInstructionMessage(tool.Name()), args...)
}

// NewSystemTextMessage builds a system *ai.Message with tool-call instructions appended.
func NewSystemTextMessage[T any](tool *ai.ToolDef[T, any], text string) *ai.Message {
	return ai.NewSystemTextMessage(text + getInstructionMessage(tool.Name()))
}

// NewTextPart builds an *ai.Part with tool-call instructions appended.
func NewTextPart[T any](tool *ai.ToolDef[T, any], text string) *ai.Part {
	return ai.NewTextPart(text + getInstructionMessage(tool.Name()))
}

// WithMessages clones messages and injects tool-call instructions into the system message.
func WithMessages[T any](tool *ai.ToolDef[T, any], messages []*ai.Message) ai.GenerateOption {
	return ai.WithMessages(injectSystemPrompt(tool.Name(), messages)...)
}

// ToolCallInstruction returns the raw instruction text for callers composing messages manually.
func ToolCallInstruction[T any](tool *ai.ToolDef[T, any]) string {
	return getInstructionMessage(tool.Name())
}

// injectSystemPrompt shallow clones the caller original messages
// and replaces the system (if exists) appending instructions
// for the llm to use the response_tool
func injectSystemPrompt(toolName string, ogMessages []*ai.Message) []*ai.Message {
	messages := append([]*ai.Message{}, ogMessages...)

	var sysMessage *ai.Message
	for i, m := range messages {
		if m.Role == ai.RoleSystem {
			sysMessage = &ai.Message{
				Role:     m.Role,
				Metadata: m.Metadata,
				Content:  append([]*ai.Part{}, m.Content...),
			}
			messages[i] = sysMessage
			break
		}
	}

	if sysMessage == nil {
		sysMessage = ai.NewSystemTextMessage("")
		messages = append([]*ai.Message{sysMessage}, messages...)
	}

	sysMessage.Content = append(sysMessage.Content,
		ai.NewTextPart(getInstructionMessage(toolName)),
	)

	return messages
}

func getInstructionMessage(toolName string) string {
	return fmt.Sprintf("\n\nProduce your response by calling the %s tool. Populate its arguments with the values requested by the schema.", toolName)
}
