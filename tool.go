package structout

import (
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

	ErrNotMatch     = errors.New("no match for T")
	ErrNotInterrupt = errors.New("response tool did not interrupt")
)

type genkitStructuredOutput[T any] struct {
	Output T
}

// DefineInterruptTool returns the response tool for type T, reused across calls.
// basic usage when is expected to be last tool call
func DefineInterruptTool[T any](g *genkit.Genkit, t T) *ai.ToolDef[T, any] {
	k := reflect.TypeOf(t)
	if tool, ok := toolRegistry.Load(k); ok {
		return tool.(*ai.ToolDef[T, any])
	}

	tool := genkit.DefineTool(g,
		fmt.Sprintf("response_tool_%d", toolCounter.Add(1)),
		"REQUIRED: call this tool at final response to conform the structured output schema",
		func(ctx *ai.ToolContext, l T) (any, error) {
			return nil, ai.InterruptWith(ctx, genkitStructuredOutput[T]{l})
		},
	)

	toolRegistry.Store(k, tool)

	return tool
}

// FromInterruptTool extracts the structured output T captured by the response tool's interrupt.
func FromInterruptTool[T any](resp *ai.ModelResponse) (T, error) {
	var zero T

	if resp == nil {
		return zero, ErrNotMatch
	}

	if resp.FinishReason == ai.FinishReasonInterrupted {
		for _, it := range resp.Interrupts() {
			out, ok := ai.InterruptAs[genkitStructuredOutput[T]](it)
			if !ok {
				continue
			}

			return out.Output, nil
		}

		return zero, ErrNotMatch
	}

	return zero, ErrNotInterrupt
}
