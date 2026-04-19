# structout

> _Slightly_ better structured output on local LLMs used with Genkit

`structout` is a Go library for the [Firebase Genkit](https://github.com/firebase/genkit) ecosystem. It coerces small/local LLMs (Ollama with gemma, qwen, etc.) into producing JSON that conforms to a Go type `T`.

Genkit's default structured-output path appends the JSON schema as text to the prompt and hopes the model honors it. Larger hosted models do; smaller local models frequently don't. `structout` sidesteps the problem by exposing the schema as a **tool** (which local models respect far better) and using a **model middleware** to rewrite the tool call into the final JSON text — without an extra round-trip to the model.

## Install

```bash
go get github.com/falmar/structout
```

## Usage

```go
package main

import (
    "context"
    "fmt"

    "github.com/falmar/structout"
    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/genkit"
    _ "github.com/firebase/genkit/go/plugins/ollama"
)

type Jedi struct {
    Name       string `json:"name"`
    Lightsaber string `json:"lightsaber" jsonschema:"enum=blue,enum=green,enum=purple"`
}

func main() {
    ctx := context.Background()
    g, err := genkit.Init(ctx /* your plugins / options */)
    if err != nil {
        panic(err)
    }

    so := structout.Define[Jedi](g)

    jedi, _, err := genkit.GenerateData[Jedi](ctx, g,
        ai.WithModelName("ollama/gemma4:e4b"),
        ai.WithSystem("You are a Jedi master."+so.Instruction),
        ai.WithPrompt("Introduce yourself."),
        ai.WithTools(so.Tool),
        ai.WithMiddleware(so.Middleware),
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("%s wields a %s lightsaber\n", jedi.Name, jedi.Lightsaber)
}
```

`Define[T]` returns three pieces you wire into your `Generate`/`GenerateData` call:

- `so.Tool` — the formatter tool. Its input schema is generated from `T`. The tool body never runs.
- `so.Middleware` — rewrites the model's tool call into plain JSON text so `GenerateData[T]` can unmarshal it.
- `so.Instruction` — system-prompt fragment telling the model to call the tool and emit its input verbatim. Append to your own system message.

## How it works

1. The model sees a tool whose input schema is `T`.
2. It calls the tool with `T`-shaped arguments.
3. The middleware intercepts the response before Genkit dispatches the tool, marshals the call's input to JSON, and replaces `resp.Message` with that text.
4. `GenerateData[T]` validates and unmarshals the text into `*T`.
5. **Native Integration**: By utilizing Genkit's built-in middleware and tool registration, `structout` maintains the integrity of Genkit's traces and plugins.

No second model call, no interrupt error in traces, and full tool-input validation happens via Genkit's output parser on the way out.

## Caveats

- **The formatter must be the only tool call in its turn.** Other turns (before or after) can call any tools you like. If the model emits the formatter alongside another tool in the same response, the middleware passes through and the model typically loops. Prompt it to finish other tool use first and call the formatter alone.
- **No streaming.** When a stream callback is active the middleware does nothing; the caller has to handle raw tool-request chunks.
- **Don't combine with `ai.WithOutputSchema` / custom output formats.** `GenerateData[T]` already schema-validates the synthesized text; an additional output handler fights with the middleware and produces confusing errors.
- **Process-global tool memoization.** `Define[T]` caches tools by `reflect.Type`. Multiple `*genkit.Genkit` instances in the same process with the same `T` share the tool registered on whichever instance called `Define[T]` first.

## Requirements

- Go 1.24+
- Genkit Go SDK ([`github.com/firebase/genkit/go`](https://github.com/firebase/genkit/tree/main/go))
