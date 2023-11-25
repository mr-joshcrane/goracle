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
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func CreateChatGPTRequest(token string, model string, messages []Message) (*http.Request, error) {
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
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

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

type ModelResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Id      string `json:"id"`
		Object  string `json:"object"`
		Created int    `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

func (c *ChatGPT) standardCompletion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	messages := MessageFromPrompt(prompt)
	req, err := CreateChatGPTRequest(c.Token, c.Model, messages)
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
