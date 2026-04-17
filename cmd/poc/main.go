package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/ollama"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	modelName := "ollama/gemma4:26b"

	g := createKit(ctx, modelName)

	err := runOllamGenerate(ctx, g, modelName)
	if err != nil {
		slog.Error("failed to generate", "error", err.Error())
		os.Exit(1)
	}
}

func createKit(
	ctx context.Context,
	model string,
) *genkit.Genkit {
	var plugins []api.Plugin

	o := &ollama.Ollama{
		ServerAddress: "http://127.0.0.1:11434",
	}

	plugins = append(plugins, o)

	g := genkit.Init(ctx,
		genkit.WithPlugins(plugins...),
		genkit.WithDefaultModel(model),
	)

	o.DefineModel(g,
		ollama.ModelDefinition{
			Name: model,
			Type: "generate",
		},
		&ai.ModelOptions{
			Supports: &ai.ModelSupports{
				Multiturn:  true,
				SystemRole: true,
				ToolChoice: true,
				Tools:      true,
				Media:      true,
			},
		},
	)

	return g
}

func runOllamGenerate(ctx context.Context, g *genkit.Genkit, model string) error {
	type Lightsaber struct {
		Color string `json:"color" jsonschema:"enum=blue,enum=green" jsonschema_description:"blade color"`
	}

	unsheatheLightsaber := genkit.DefineTool(g, "response_tool", "Choose a color for your lightsaber",
		func(ctx *ai.ToolContext, l Lightsaber) (any, error) {
			fmt.Println("pop lightsaber!", l.Color)
			return nil, ai.InterruptWith(ctx, l)
		},
	)

	resp, err := genkit.Generate(ctx, g,
		ai.WithSystem("You are General Grievous. Use must call response_tool first before your response."),
		ai.WithPrompt("Obi-Wan Kenobi: Hello there"),
		ai.WithModelName(model),
		ai.WithTools(unsheatheLightsaber),
	)
	if err != nil {
		return err
	}

	if resp.FinishReason == ai.FinishReasonInterrupted {
		for _, it := range resp.Interrupts() {
			out, ok := ai.InterruptAs[Lightsaber](it)
			if !ok {
				continue
			}

			b, err := json.Marshal(out)
			if err != nil {
				return err
			}

			fmt.Println("structured output", string(b))
			return nil
		}
	}

	return errors.New("failed to call tool with interrupt")
}
