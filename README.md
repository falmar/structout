# GenKit - Ghool

I dont like how GenKit manages structured output (at the time of writing).

So here is attempt to create create genkit tools out of structs
making the llm call the tool in order to get structured output back.

This approach is used by langchain and works well for the most part,
better than asking the model to conform the schema

Motivation: using local models with genkit is painful
