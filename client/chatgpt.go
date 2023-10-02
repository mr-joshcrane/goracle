package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type Model string

const (
	GPT35Turbo = Model("gpt-3.5-turbo")
	GPT4       = Model("gpt-4")
)

type ChatGPT struct {
	Token string
	model Model
}

type Dummy struct {
	FixedResponse  string
	FixedHTTPError int
}

func (d *Dummy) Completion(ctx context.Context, prompt Prompt) (string, error) {
	if d.FixedHTTPError == 200 {
		return d.FixedResponse, nil
	}
	response := http.Response{
		Status:     "client error",
		StatusCode: d.FixedHTTPError,
	}
	return "", NewClientError(&response)
}

type Option func(*ChatGPT) *ChatGPT

func WithGPTModel(m Model) Option {
	return func(c *ChatGPT) *ChatGPT {
		c.model = m
		return c
	}
}

func NewChatGPT(token string, opts ...Option) *ChatGPT {
	c := &ChatGPT{
		Token: token,
		model: GPT35Turbo,
	}
	for _, opt := range opts {
		c = opt(c)
	}
	return c
}

func NewDummyClient(fixedResponse string, errorCode int) *Dummy {
	return &Dummy{
		FixedResponse:  fixedResponse,
		FixedHTTPError: errorCode,
	}
}

type Prompt interface {
	GetPurpose() string
	GetExamples() ([]string, []string)
	GetQuestion() string
}

func MessageFromPrompt(prompt Prompt) []Message {
	messages := []Message{}
	messages = append(messages, Message{
		Role:    RoleSystem,
		Content: prompt.GetPurpose(),
	})
	givenInputs, idealOutputs := prompt.GetExamples()
	for i, givenInput := range givenInputs {
		messages = append(messages, Message{
			Role:    RoleUser,
			Content: givenInput,
		})
		messages = append(messages, Message{
			Role:    RoleAssistant,
			Content: idealOutputs[i],
		})
	}
	messages = append(messages, Message{
		Role:    RoleUser,
		Content: prompt.GetQuestion(),
	})
	return messages
}

func (c *ChatGPT) Completion(ctx context.Context, prompt Prompt) (string, error) {
	messages := MessageFromPrompt(prompt)
	req, err := CreateChatGPTRequest(c.Token, c.model, messages)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", NewClientError(resp)
	}
	defer resp.Body.Close()
	return ParseResponse(resp.Body)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model    Model     `json:"model"`
	Messages []Message `json:"messages"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func CreateChatGPTRequest(token string, model Model, messages []Message) (*http.Request, error) {
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

func ParseResponse(r io.Reader) (string, error) {
	resp := ChatCompletionResponse{}
	err := json.NewDecoder(r).Decode(&resp)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) < 1 {
		return "", fmt.Errorf("no choices returned")
	}
	return resp.Choices[0].Message.Content, nil
}
