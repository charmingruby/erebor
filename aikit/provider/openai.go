package provider

import (
	genai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
)

// OpenAI provides models and embeddings through OpenAI.
type OpenAI struct {
	plugin *openai.OpenAI
}

// NewOpenAI creates an OpenAI provider with the supplied API key.
func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{
		plugin: &openai.OpenAI{
			APIKey: apiKey,
		},
	}
}

// Plugin returns the OpenAI Genkit plugin.
func (p *OpenAI) Plugin() api.Plugin {
	return p.plugin
}

// Model returns a reference to the named OpenAI model.
func (p *OpenAI) Model(name string) genai.ModelRef {
	return openai.ModelRef(name, nil)
}

// Embedder returns an OpenAI embedder with the requested dimensions.
func (p *OpenAI) Embedder(name string, dimensions int, _ bool) (genai.EmbedderRef, error) {
	return openai.NewEmbedderRef(
		name,
		&openai.TextEmbeddingConfig{
			Dimensions: dimensions,
		},
	), nil
}
