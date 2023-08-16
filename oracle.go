package oracle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	// "github.com/mr-joshcrane/chatproxy"
)


type Prompts []Prompt

type Prompt struct {
	Prompt     string
	Completion string
}

func (p Prompts) String() string {
	s := ""
	for _, prompt := range p {
		s += fmt.Sprintf("USER: %s\nBOT: %s\n", prompt.Prompt, prompt.Completion)
	}
	return s
}

type Oracle struct {
	purpose  string
	examples []Prompt
}

func NewOracle() *Oracle {
	return &Oracle{
		purpose:  "You are a helpful assistant",
		examples: Prompts{},
	}
}

func (o *Oracle) Ask(question string) (string, error) {
	token := os.Getenv("OPENAI_API_KEY")
	req := CreateChatGPTRequest(token, []Message{
		{
			Role:    "user",
			Content: question,
		},
	})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return ParseResponse(resp.Body)
}

func (o *Oracle) SetPurpose(purpose string) {
	o.purpose = purpose
}

func (o *Oracle) GiveExamplePrompt(prompt string, idealCompletion string) {
	o.examples = append(o.examples, Prompt{prompt, idealCompletion})
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
	if len(resp.Choices) < 1 {
		return "", errors.New("No choices returned")
	}
	return resp.Choices[0].Message.Content, nil
}
