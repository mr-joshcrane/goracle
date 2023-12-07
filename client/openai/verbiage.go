package openai

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

type TextCompletionRequest struct {
	Model    string   `json:"model"`
	Messages Messages `json:"messages"`
}

type TextCompletionResponse struct {
	Choices []struct {
		Message TextMessage `json:"message"`
	} `json:"choices"`
}

func textCompletion(ctx context.Context, token string, messages Messages) (io.Reader, error) {
	req, err := CreateTextCompletionRequest(token, GPT4, messages)
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
	answer, err := ParseTextCompletionReponse(resp.Body)
	if err != nil {
		return nil, err
	}
	return answer, err
}

func CreateTextCompletionRequest(token string, model string, messages Messages) (*http.Request, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(TextCompletionRequest{
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

func ParseTextCompletionReponse(r io.Reader) (io.Reader, error) {
	resp := TextCompletionResponse{}
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
