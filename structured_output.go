package structout

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/google/uuid"
)

var (
	toolRegistry sync.Map
	toolCounter  atomic.Uint64
)

// Output bundles the tool, middleware, and system-prompt instruction needed
// to coerce an LLM into producing a value of type T via tool-calling. It is
// the return value of [Define].
//
// The tool's input schema is derived from T; the middleware intercepts the
// model's tool call and rewrites the response to plain JSON text, so
// [genkit.GenerateData] can validate and unmarshal it into T.
//
// Example:
//
//	type Jedi struct {
//	    Name       string `json:"name"`
//	    Lightsaber string `json:"lightsaber" jsonschema:"enum=blue,enum=green,enum=purple"`
//	}
//
//	so := structout.Define[Jedi](g)
//	jedi, resp, err := genkit.GenerateData[Jedi](ctx, g,
//	    ai.WithModelName("ollama/gemma4:e4b"),
//	    ai.WithSystem("You are a Jedi master." + so.Instruction),
//	    ai.WithPrompt("Introduce yourself."),
//	    ai.WithTools(so.Tool),
//	    ai.WithMiddleware(so.Middleware),
//	)
type Output[T any] struct {
	// Tool is the formatter tool exposed to the model. Its input schema is
	// generated from T. The tool body never runs — [Output.Middleware]
	// rewrites the response before genkit's tool loop can dispatch it.
	Tool *ai.ToolDef[T, any]

	// Middleware intercepts the model response after generation. When the
	// model returns the formatter tool as the only tool request in the turn,
	// the middleware marshals its input to JSON text and replaces
	// resp.Message so the tool loop exits cleanly. Pass to the Generate call
	// via ai.WithMiddleware.
	//
	// Caveats:
	//   - Only the single-formatter case is rewritten. If the model emits
	//     the formatter alongside other tool calls in the same turn, the
	//     middleware passes through; the other tools run and the model
	//     typically loops. Prompt the model to complete other tool use
	//     first, then call the formatter on its own turn.
	//   - Streaming is not supported. When a stream callback is present,
	//     the middleware passes through unchanged and the caller must
	//     handle the raw tool-request chunks themselves.
	//   - Do not combine with ai.WithOutputSchema or a non-default output
	//     format on the same request. [genkit.GenerateData] already
	//     validates and unmarshals the synthesized text against T.
	Middleware ai.ModelMiddleware

	// Instruction is a system-prompt fragment that tells the model to call
	// Tool and emit its output JSON verbatim. Append it to a system message
	// of your choice; it references the tool by its registered name.
	Instruction string
}

// Define registers a formatter tool for type T on g and returns an
// [Output] bundling the tool, a response-rewriting middleware, and a
// system-prompt instruction.
//
// T should be a JSON-serializable struct; scalars and maps work but give
// the model a weaker schema to target.
//
// Tools are memoized by reflect.Type: a second call to Define[T] with the
// same T returns the first registration. The memoization is process-global
// and is NOT scoped to a *genkit.Genkit instance — using multiple genkit
// instances in the same process with the same T will share the tool
// registered on whichever instance called Define[T] first.
func Define[T any](g *genkit.Genkit) *Output[T] {
	var zero T
	tool := defineOutputTool(g, zero)
	return &Output[T]{
		Tool:        tool,
		Instruction: getInstructionMessage(tool.Name()),
		Middleware:  outputMiddleware(tool),
	}
}

// defineOutputTool registers (or returns a cached) formatter tool for T.
// The tool body is a no-op identity function [outputMiddleware] short-circuits the tool loop before
// genkit dispatches it.
func defineOutputTool[T any](g *genkit.Genkit, t T) *ai.ToolDef[T, any] {
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

// outputMiddleware returns a model middleware that intercepts a formatter
// tool call and rewrites the response to plain JSON text.
//
// Flow: after the inner model call returns, the middleware inspects
// resp.Message. If it contains exactly one tool request and that request
// targets the formatter, its Input is marshaled to JSON; resp.Message is
// replaced with a text message holding that JSON; resp.Request.Messages is
// rewritten to include the original model turn plus a synthetic tool
// response (for trace fidelity). This leaves resp.ToolRequests() empty so
// genkit's generate loop returns instead of dispatching the tool.
//
// Skipped (pass-through) cases:
//   - next(...) returned an error.
//   - A stream callback is active; rewriting would not match what was
//     already streamed to the caller.
//   - The model returned zero tool requests, more than one tool request,
//     or a single tool request that is not the formatter.
func outputMiddleware[T any](tool *ai.ToolDef[T, any]) ai.ModelMiddleware {
	return func(next ai.ModelFunc) ai.ModelFunc {
		return func(ctx context.Context, input *ai.ModelRequest, cb ai.ModelStreamCallback) (*ai.ModelResponse, error) {
			resp, err := next(ctx, input, cb)
			if err != nil || cb != nil {
				return resp, err
			}

			if len(resp.ToolRequests()) == 1 {
				toolReq := resp.ToolRequests()[0]

				if toolReq.Name == tool.Name() {
					modMessages := make([]*ai.Message, 0, len(input.Messages)+2)
					modMessages = append(modMessages, input.Messages...)

					if toolReq.Ref == "" {
						toolReq.Ref = uuid.New().String()
					}

					b, err := json.Marshal(toolReq.Input)
					if err != nil {
						return resp, err
					}

					// append tool request
					modMessages = append(modMessages, resp.Message)

					// append tool response
					modMessages = append(modMessages, ai.NewMessage(ai.RoleTool, nil, ai.NewToolResponsePart(
						&ai.ToolResponse{
							Ref:    toolReq.Ref, // ensure ref id
							Name:   toolReq.Name,
							Output: toolReq.Input,
						},
					)))

					// replace request messages
					resp.Request.Messages = modMessages

					// replace response message
					resp.Message = ai.NewModelTextMessage(string(b))
					if resp.Usage != nil {
						resp.Usage.OutputCharacters += len(b)
					}

					return resp, nil
				}
			}

			return resp, err
		}
	}
}

func getInstructionMessage(toolName string) string {
	return fmt.Sprintf("\n\nProduce your response by calling the %s tool. Populate its arguments with the values requested by the schema. After the tool responds, emit the tool's output JSON verbatim as your final message — no prose, no markdown fences, no commentary.", toolName)
}
