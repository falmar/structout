package structout

import (
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

// DefineOutputTool returns the response tool for type T, reused across calls.
// marshals the input as text with instructions for the llm to return it as JSON verbatim
func DefineOutputTool[T any](g *genkit.Genkit, t T) *ai.ToolDef[T, any] {
	k := reflect.TypeOf(t)
	if tool, ok := toolRegistry.Load(k); ok {
		return tool.(*ai.ToolDef[T, any])
	}

	tool := genkit.DefineTool(g,
		fmt.Sprintf("response_formatter_%d", toolCounter.Add(1)),
		"REQUIRED: call this tool to conform the structured output schema",
		func(ctx *ai.ToolContext, t T) (any, error) {
			return t, nil
		},
	)

	toolRegistry.Store(k, tool)

	return tool
}
