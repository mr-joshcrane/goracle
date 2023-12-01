package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	artifacts := prompt.GetArtifacts()
	if len(artifacts) > 0 {
		img, err := c.GenerateImage(prompt.GetQuestion(), len(artifacts))
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Get(img.Data[0].Url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		for _, artifact := range artifacts {
			_, err := artifact.Write(data)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
		}
		return strings.NewReader("I drew you a picture!"), nil
	}
	messages := MessageFromPrompt(prompt)
	for _, message := range messages {
		if message.GetFormat() == MessageImage {
			return c.visionCompletion(ctx, messages)
		}
	}
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
