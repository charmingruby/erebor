package aikit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	genai "github.com/firebase/genkit/go/ai"
	aix "github.com/firebase/genkit/go/ai/exp"
	"github.com/firebase/genkit/go/genkit"
	genkitx "github.com/firebase/genkit/go/genkit/exp"

	"github.com/charmingruby/erebor/o11y/logging"
)

// Agent wraps a Genkit agent with logging and inference error handling.
type Agent struct {
	def  *aix.Agent[struct{}]
	name string
}

// DefineAgent registers an agent with the supplied model, tools, and prompt.
func DefineAgent(
	ak *AIKit,
	name string,
	model genai.ModelRef,
	tools []Tool,
	prompt aix.InlinePrompt,
	opts ...aix.AgentOption[struct{}],
) (*Agent, error) {
	refs := make([]genai.ToolRef, 0, len(tools))
	for _, tool := range tools {
		genkit.RegisterAction(ak.Genkit, tool.Ref)
		refs = append(refs, tool.Ref)
	}

	cfg := aix.InlinePrompt{
		genai.WithTools(refs...),
		genai.WithMaxTurns(8),
		genai.WithModel(model),
		genai.WithUse(
			toolPolicyMiddleware(),
			agentObserverMiddleware(name),
		),
	}

	cfg = append(cfg, prompt...)

	def := genkitx.DefineAgent[struct{}](ak.Genkit, name, cfg, opts...)

	return &Agent{
		def:  def,
		name: name,
	}, nil
}

// Run sends a message to the agent and returns its text response.
func (a *Agent) Run(
	ctx context.Context,
	message string,
	opts ...aix.InvocationOption[struct{}],
) (string, error) {
	log := logging.LoggerFromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	out, err := a.def.RunText(ctx, message, opts...)
	if err != nil {
		log.WarnContext(ctx, "agent invocation failed",
			"agent", a.name,
		)

		return "", fmt.Errorf("%w: %w", ErrInference, err)
	}

	log.InfoContext(ctx, "agent execution finished", "agent", a.name, "outcome", out.FinishReason)

	if out.Usage != nil {
		log.InfoContext(ctx, "agent model usage",
			"agent", a.name,
			"input_tokens", out.Usage.InputTokens,
			"output_tokens", out.Usage.OutputTokens,
		)
	}

	if out.Error != nil {
		return "", fmt.Errorf("%w: %w", ErrInference, out.Error)
	}

	unexpectedFinish := out.FinishReason != aix.AgentFinishReasonStop || out.Message == nil

	if unexpectedFinish {
		return "", fmt.Errorf("%w: agent finish reason %s", ErrInference, out.FinishReason)
	}

	text := out.Message.Text()
	if text == "" {
		return "", fmt.Errorf("%w: empty agent response", ErrInference)
	}

	return text, nil
}

// HandleHTTP serves the agent through Genkit's HTTP handler.
func (a *Agent) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	genkit.Handler(a.def).ServeHTTP(w, r)
}

// Ref returns the underlying agent reference.
func (a *Agent) Ref() aix.AgentRef {
	return a.def.Ref()
}
