package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	GPT35Turbo = "gpt-3.5-turbo-1106"
	GPT4       = "gpt-4-1106-preview"
)

type ChatCompletionRequest struct {
	Model    string   `json:"model"`
	Messages Messages `json:"messages"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message TextMessage `json:"message"`
	} `json:"choices"`
}

func standardCompletion(ctx context.Context, token string, prompt Prompt) (io.Reader, error) {
	messages := MessageFromPrompt(prompt)
	req, err := CreateChatGPTRequest(token, GPT4, messages)

	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, NewClientError(resp)
	}
	defer resp.Body.Close()
	answer, err := ParseResponse(resp.Body)
	if err != nil {
		return nil, err
	}
	return answer, err

}

func CreateChatGPTRequest(token string, model string, messages Messages) (*http.Request, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", buf)
	if err != nil {
		return nil, err
	}
	req = addDefaultHeaders(token, req)
	return req, nil
}

func ParseResponse(r io.Reader) (io.Reader, error) {
	resp := ChatCompletionResponse{}
	err := json.NewDecoder(r).Decode(&resp)
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) < 1 {
		return nil, fmt.Errorf("no choices returned")
	}
	output := strings.NewReader(resp.Choices[0].Message.Content)
	return output, nil
}

func (c *ChatGPT) CompletionSwitchboard(ctx context.Context, prompt Prompt) (io.Reader, error) {
	a, _ := prompt.GetArtifacts()
	if len(a) > 0 {
		return imageRequestPrompt(ctx, c.Token, prompt)
	}
	messages := MessageFromPrompt(prompt)
	for _, message := range messages {
		if message.GetFormat() == MessageImage {
			return c.visionCompletion(ctx, messages)
		}
	}
	return standardCompletion(ctx, c.Token, prompt)
}
