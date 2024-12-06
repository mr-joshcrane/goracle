package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"
)

type Role string

var (
	User Role = "USER"
	Bot  Role = "ASSISTANT"
)

type Prompt interface {
	GetPurpose() string
	GetHistory() ([]string, []string)
	GetQuestion() string
	GetReferences() [][]byte
}

type ChatMessage struct {
	Role  Role        `json:"role"`
	Parts MessagePart `json:"parts"`
}

type MessagePart struct {
	Text string `json:"text,omitempty"`
}

type Anthropic struct {
	Token string
	Model ModelConfig
}

func NewAnthropic(token string) *Anthropic {
	return &Anthropic{
		Token: token,
		Model: Models["ClaudeSonnet"],
	}
}

func Authenticate() (token string, err error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}
	return key, nil
}

func capabilityCheck(model ModelConfig, prompt Prompt) error {
	if !model.SupportsVision {
		for _, ref := range prompt.GetReferences() {
			kind := detectDataKind(ref)
			if kind == DataKindImage {
				return fmt.Errorf("model %s does not support image references", model.Name)
			}
		}
	}
	return nil
}

func Completion(ctx context.Context, token string, model ModelConfig, prompt Prompt) (io.Reader, error) {
	err := capabilityCheck(model, prompt)
	if err != nil {
		return nil, err
	}
	req, err := createCompletionRequest(ctx, token, model, prompt)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseAnthropicResponse(resp)
}

func createCompletionRequest(ctx context.Context, token string, model ModelConfig, prompt Prompt) (*http.Request, error) {
	messages := createAnthropicMessages(prompt)
	requestBody := map[string]any{
		"model":      model.Name,
		"system":     prompt.GetPurpose(),
		"max_tokens": 1024,
		"messages":   messages,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.anthropic.com/v1/messages",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", token)
	return req, nil
}

type Message struct {
	Role    Role `json:"role"`
	Content any  `json:"content"`
}

type ImagePayload struct {
	Type   string `json:"type"`
	Source struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	} `json:"source"`
}

func createAnthropicMessages(prompt Prompt) []Message {
	messages := []Message{}
	userHistory, assistantHistory := prompt.GetHistory()
	for i := range userHistory {
		messages = append(messages, Message{Role: "user", Content: userHistory[i]})
		messages = append(messages, Message{Role: "assistant", Content: assistantHistory[i]})
	}
	messages = append(messages, Message{Role: "user", Content: prompt.GetQuestion()})

	for _, ref := range prompt.GetReferences() {
		content, err := processReference(ref)
		if err != nil {
			continue
		}
		messages = append(messages, Message{Role: "user", Content: content})
	}
	return messages
}

func parseAnthropicResponse(resp *http.Response) (io.Reader, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d; %s", resp.StatusCode, resp.Status)
	}

	var responseBody struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}
	var completion string
	for _, message := range responseBody.Content {
		completion += message.Text
	}
	return strings.NewReader(completion), nil
}
