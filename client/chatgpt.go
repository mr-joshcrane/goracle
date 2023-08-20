package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type ChatGPTClient struct {
	Token string
}

func NewChatGPTClient() *ChatGPTClient {
	token := os.Getenv("OPENAI_API_KEY")
	return &ChatGPTClient{
		Token: token,
	}
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

func (c *ChatGPTClient) Completion(prompt Prompt) (string, error) {
	messages := MessageFromPrompt(prompt)
	req := CreateChatGPTRequest(c.Token, messages)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("bad status code from openai" + resp.Status)
	}
	fmt.Println(resp)
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

func CreateChatGPTRequest(token string, messages []Message) *http.Request {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(ChatCompletionRequest{
		Model:    "gpt-3.5-turbo",
		Messages: messages,
	})
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", buf)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return req
}

func ParseResponse(r io.Reader) (string, error) {
	resp := ChatCompletionResponse{}
	err := json.NewDecoder(r).Decode(&resp)
	if err != nil {
		return "", err
	}
	fmt.Println(resp)
	if len(resp.Choices) < 1 {
		return "", errors.New("no choices returned")
	}
	return resp.Choices[0].Message.Content, nil
}
