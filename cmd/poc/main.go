package main

import (
	"context"
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
	text, err := genkit.GenerateText(ctx, g,
		ai.WithSystem("Star Wars: The Clone Wars\n---\nYou are: General Gravious\nCurrently stationed at: Utapau"),
		ai.WithPrompt("Jedi: Hello there!"),
		ai.WithModelName(model),
	)
	if err != nil {
		return err
	}

	fmt.Println(text)
	return nil
}
