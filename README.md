# GenKit - Structured Output

I dont like how GenKit manages structured output (at the time of writing).

So here is attempt to create create genkit tools out of structs schemas
making the llm call the tool in order to get structured output back.

This approach is used by langchain and works well for the most part,
better than asking the model to conform the schema

> NOTE: This is just a prototype to see how reliable it gets to use ollama local models
> with genkit using the aforementioned technique

> NOTE2: ~sadly it relies on the tool interrupt mechanism
> and it is considered "warning/error" traces on the Dev UI~

> NOTE3: ~i've added a just that tool that just returns its own input
> and by changing the instructions for the llm to return that output verbatim
> it does remove the interrupt mechanism but now its double round-trip
> making the llm go for extraction of the tool response and put it back as last response~

NOTE4: i've found out it was possible to use genkit model middleware to intercept
the response, and by replacing/injecting the messages to fake a model response  
we can avoid calling llm for it to extract the json.

Motivation: the lack of tool choice when using local models with genkit is painful
