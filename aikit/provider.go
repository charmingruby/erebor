package aikit

import (
	genai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
)

// Provider supplies a Genkit plugin and model and embedder references.
type Provider interface {
	Plugin() api.Plugin
	Model(name string) genai.ModelRef
	Embedder(name string, dimensions int, query bool) (genai.EmbedderRef, error)
}
