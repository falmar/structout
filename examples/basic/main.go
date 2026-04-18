package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/falmar/structout"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/ollama"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	modelName := "ollama/gemma4:26b"

	// setup genkit with ollama plugin
	g := createKit(ctx, modelName)

	// call llm
	resp, err := chooseLightsaberColor(ctx, g, modelName)
	if err != nil {
		slog.Error("failed to generate", "error", err.Error())
		os.Exit(1)
	}

	fmt.Printf("Jedi *%s* has chosen a *%s* lightsaber\n", resp.Name, resp.Lightsaber.Color)
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

type Jedi struct {
	Name       string `json:"name" jsonschema_description:"your name"`
	Lightsaber Lightsaber
}

type Lightsaber struct {
	Color string `json:"color" jsonschema:"enum=blue,enum=green,enum=purple" jsonschema_description:"choose the color of the lightsaber to unsheathe"`
}

func chooseLightsaberColor(ctx context.Context, g *genkit.Genkit, model string) (Jedi, error) {
	var zero Jedi

	resp, err := structout.GenerateStructuredOutput[Jedi](ctx, g,
		// slice of messages is used to inject instruction to call the tool in the system message
		[]*ai.Message{
			ai.NewSystemTextMessage("You are about to become a Jedi in the Star Wars universe.\nFollow the instructions."),
			ai.NewUserTextMessage("Young Padawan what your name and color of choice for your lightsaber?."),
		},

		// if you have more tools pass them here
		nil, // []ai.ToolRef{...}

		// pass regular ai.GeneraetOption
		ai.WithModelName(model),
		ai.WithConfig(ai.GenerationCommonConfig{
			Temperature: 1,
		}),
	)
	if err != nil {
		return zero, err
	}

	return resp, nil
}
