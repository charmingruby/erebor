package provider

import (
	"fmt"
	"math"

	genai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	googleai "google.golang.org/genai"
)

// EmbeddingTask identifies the intended use of an embedding.
type EmbeddingTask string

const (
	// RetrievalDocumentEmbeddingTask embeds documents for retrieval.
	RetrievalDocumentEmbeddingTask = "RETRIEVAL_DOCUMENT"
	// RetrievalQueryEmbeddingTask embeds search queries for retrieval.
	RetrievalQueryEmbeddingTask = "RETRIEVAL_QUERY"
)

// Google provides models and embeddings through Google AI.
type Google struct {
	plugin *googlegenai.GoogleAI
}

// NewGoogle creates a Google AI provider with the supplied API key.
func NewGoogle(apiKey string) *Google {
	return &Google{
		plugin: &googlegenai.GoogleAI{
			APIKey: apiKey,
		},
	}
}

// Plugin returns the Google AI Genkit plugin.
func (p *Google) Plugin() api.Plugin {
	return p.plugin
}

// Model returns a reference to the named Google AI model.
func (p *Google) Model(name string) genai.ModelRef {
	return googlegenai.ModelRef(p.plugin.Name()+"/"+name, nil)
}

// Embedder returns a Google AI embedder configured for documents or queries.
func (p *Google) Embedder(name string, dimensions int, query bool) (genai.EmbedderRef, error) {
	if dimensions < 1 || dimensions > math.MaxInt32 {
		var ref genai.EmbedderRef
		return ref, fmt.Errorf("embedding dimensions must be between 1 and %d: %d", math.MaxInt32, dimensions)
	}

	task := RetrievalDocumentEmbeddingTask
	if query {
		task = RetrievalQueryEmbeddingTask
	}

	return googlegenai.EmbedderRef(p.plugin.Name()+"/"+name, &googleai.EmbedContentConfig{
		TaskType:             task,
		OutputDimensionality: new(int32(dimensions)),
	}), nil
}
