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
	GPT4o     = "gpt-4o"
	GPT4oMini = "gpt-4o-mini"
	GPTo1     = "o1-preview"
	GPTo1Mini = "o1-mini"
)

type TextCompletionRequest struct {
	Model          string         `json:"model"`
	Messages       Messages       `json:"messages"`
	ResponseFormat map[string]any `json:"response_format"`
}

type TextCompletionResponse struct {
	Choices []struct {
		Message TextMessage `json:"message"`
	} `json:"choices"`
}

func textCompletion(ctx context.Context, token string, model ModelConfig, messages Messages, format ...string) (io.Reader, error) {
	if !model.SupportsSystemMessages {
		messages = messages[1:]
	}
	req, err := CreateTextCompletionRequest(token, model.Name, messages, format...)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	return ParseTextCompletionRequest(resp)
}

func parseResponseArg(arg string) (string, string) {
	parts := strings.Split(arg, ":")
	if len(parts) < 2 {
		return arg, "A self-explanatory field"
	}
	return parts[0], parts[1]
}

func createFormatResponse(args ...string) map[string]any {
	if len(args) < 1 {
		return nil
	}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"response": map[string]any{
				"type":  "array",
				"items": map[string]any{},
			},
		},
		"additionalProperties": false,
	}

	items := schema["properties"].(map[string]any)["response"].(map[string]any)["items"].(map[string]any)
	for _, arg := range args {
		name, description := parseResponseArg(arg)
		items[name] = map[string]any{
			"description": description,
			"type":        "string",
		}
	}

	return map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":   "response_schema",
			"schema": schema,
		},
	}
}

func CreateTextCompletionRequest(token string, model string, messages Messages, outputs ...string) (*http.Request, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(TextCompletionRequest{
		Model:          model,
		Messages:       messages,
		ResponseFormat: createFormatResponse(outputs...),
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", buf)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func ParseTextCompletionRequest(resp *http.Response) (io.Reader, error) {
	if http.StatusOK != resp.StatusCode {
		return nil, NewClientError(resp)
	}
	defer resp.Body.Close()
	var completion TextCompletionResponse
	err := json.NewDecoder(resp.Body).Decode(&completion)
	if err != nil {
		return nil, err
	}
	if len(completion.Choices) < 1 {
		return nil, fmt.Errorf("no choices returned")
	}
	output := strings.NewReader(completion.Choices[0].Message.Content)
	return output, nil
}
