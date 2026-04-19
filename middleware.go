package structout

import (
	"context"
	"encoding/json"

	"github.com/firebase/genkit/go/ai"
	"github.com/google/uuid"
)

func OutputMiddleware[T any](tool *ai.ToolDef[T, any]) ai.ModelMiddleware {
	return func(next ai.ModelFunc) ai.ModelFunc {
		return func(ctx context.Context, input *ai.ModelRequest, cb ai.ModelStreamCallback) (*ai.ModelResponse, error) {
			resp, err := next(ctx, input, cb)
			if err != nil {
				return resp, err
			}

			// for now only works if its the only tool request call ()
			if len(resp.ToolRequests()) == 1 {
				for _, toolReq := range resp.ToolRequests() {
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
						resp.Usage.OutputCharacters += len(b)

						return resp, nil
					}
				}
			}

			return resp, err
		}
	}
}
