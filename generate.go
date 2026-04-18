package structout

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

var (
	toolRegistry sync.Map
	toolCounter  atomic.Uint64
)

type genkitStructuredOutput[T any] struct {
	Output T
}

func structuredOutputTool[T any](g *genkit.Genkit, t T) *ai.ToolDef[T, any] {
	k := reflect.TypeOf(t)
	if tool, ok := toolRegistry.Load(k); ok {
		return tool.(*ai.ToolDef[T, any])
	}

	tool := genkit.DefineTool(g,
		fmt.Sprintf("response_tool_%d", toolCounter.Add(1)),
		"REQUIRED: use this tool to conform the structured output as the model response",
		func(ctx *ai.ToolContext, l T) (any, error) {
			return nil, ai.InterruptWith(ctx, genkitStructuredOutput[T]{l})
		},
	)

	toolRegistry.Store(k, tool)
	return tool
}

func GenerateStructuredOutput[T any](ctx context.Context, g *genkit.Genkit, messages []*ai.Message, tools []ai.ToolRef, opts ...ai.GenerateOption) (T, error) {
	var zero T

	responseTool := structuredOutputTool(g, zero)

	opts = append(opts,
		ai.WithTools(append([]ai.ToolRef{responseTool}, tools...)...),
		ai.WithMessages(injectSystemPrompt(responseTool.Name(), messages)...),
	)

	resp, err := genkit.Generate(ctx, g, opts...)
	if err != nil {
		return zero, err
	}

	if resp.FinishReason == ai.FinishReasonInterrupted {
		for _, it := range resp.Interrupts() {
			out, ok := ai.InterruptAs[genkitStructuredOutput[T]](it)
			if !ok {
				continue
			}

			return out.Output, nil
		}
	}

	return zero, errors.New("failed to extract structured output: not interrupted")
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
		ai.NewTextPart(fmt.Sprintf("\n\nProduce your response by calling the %s tool. Populate its arguments with the values requested by the schema.", toolName)),
	)

	return messages
}
