package llm

import "context"

type OllamaLLM struct{ host, model string }

func NewOllamaLLM(host, model string) LLM { return &OllamaLLM{host: host, model: model} }

func (l *OllamaLLM) Generate(ctx context.Context, prompt string) (string, error) {
	panic("not yet implemented")
}
