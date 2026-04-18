# GenKit - Ghool

I dont like how GenKit manages structured output (at the time of writing).

so here is attempt to create create genkit tools out of structs schemas
making the llm call the tool in order to get structured output back.

This approach is used by langchain and works well for the most part,
better than asking the model to conform the schema

Motivation: the lack of tool choice when using local models with genkit is painful

NOTE: this is just a prototype to see how reliable it gets to use ollama local models
with genkit using the aforementioned technique
