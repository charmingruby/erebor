package aikit

import (
	"context"

	genai "github.com/firebase/genkit/go/ai"

	"github.com/charmingruby/erebor/validator"
)

// Tool wraps a Genkit tool reference.
type Tool struct {
	Ref genai.Tool
}

// ToolFn is a typed tool implementation.
type ToolFn[In, Out any] func(context.Context, In) (Out, error)

// DefineTool creates a tool that validates its input before invoking fn.
func DefineTool[In, Out any](
	name, description string,
	fn ToolFn[In, Out],
) Tool {
	return Tool{
		Ref: genai.NewTool(name, description,
			func(ctx *genai.ToolContext, input In) (Out, error) {
				if err := validator.Validate(input); err != nil {
					var output Out
					return output, err
				}

				return fn(ctx, input)
			},
		),
	}
}
