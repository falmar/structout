package structout

import (
	"fmt"
)

// ToolCallInstruction returns the raw instruction text for callers composing messages manually.
func ToolCallInstruction(toolName string) string {
	return getInstructionMessage(toolName)
}

func getInstructionMessage(toolName string) string {
	return fmt.Sprintf("\n\nProduce your response by calling the %s tool. Populate its arguments with the values requested by the schema. After the tool responds, emit the tool's output JSON verbatim as your final message — no prose, no markdown fences, no commentary.", toolName)
}
