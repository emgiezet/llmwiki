package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ollamaHTTPClient is a package-level client with a bounded timeout so that
// hung Ollama servers cannot block the process indefinitely.
var ollamaHTTPClient = &http.Client{
	Timeout: 2 * time.Minute,
}

type OllamaLLM struct {
	host, model string
	maxTokens   int
}

func NewOllamaLLM(host, model string, maxTokens int) LLM {
	return &OllamaLLM{host: host, model: model, maxTokens: maxTokens}
}

func (l *OllamaLLM) Generate(ctx context.Context, prompt string) (string, error) {
	payload := map[string]interface{}{
		"model":  l.model,
		"prompt": prompt,
		"stream": false,
	}
	if l.maxTokens > 0 {
		payload["options"] = map[string]interface{}{"num_predict": l.maxTokens}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.host+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ollamaHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode ollama response: %w", err)
	}
	return strings.TrimSpace(result.Response), nil
}
