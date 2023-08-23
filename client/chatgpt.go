package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type ChatGPT struct {
	Token string
}

type GPTOptions func(*ChatGPT)

func WithToken(token string) GPTOptions {
	return func(c *ChatGPT) {
		c.Token = token
	}
}

func NewChatGPT(opts ...GPTOptions) *ChatGPT {
	c := &ChatGPT{
		Token: os.Getenv("OPENAI_API_KEY"),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Prompt interface {
	GetPurpose() string
	GetExamples() []struct{ GivenInput, IdealOutput string }
	GetQuestion() string
}

func MessageFromPrompt(prompt Prompt) []Message {
	messages := []Message{}
	messages = append(messages, Message{
		Role:    RoleSystem,
		Content: prompt.GetPurpose(),
	})
	for _, example := range prompt.GetExamples() {
		messages = append(messages, Message{
			Role:    RoleUser,
			Content: example.GivenInput,
		})
		messages = append(messages, Message{
			Role:    RoleAssistant,
			Content: example.IdealOutput,
		})
	}
	messages = append(messages, Message{
		Role:    RoleUser,
		Content: prompt.GetQuestion(),
	})
	return messages
}

func (c *ChatGPT) Completion(prompt Prompt) (string, error) {
	messages := MessageFromPrompt(prompt)
	req, err := CreateChatGPTRequest(c.Token, messages)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("bad status code from openai" + resp.Status)
	}
	defer resp.Body.Close()
	return ParseResponse(resp.Body)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func CreateChatGPTRequest(token string, messages []Message) (*http.Request, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(ChatCompletionRequest{
		Model:    "gpt-3.5-turbo",
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
		return "", errors.New("no choices returned")
	}
	return resp.Choices[0].Message.Content, nil
}
