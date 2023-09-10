package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type ChatGPT struct {
	Token string
}

type Dummy struct {
	FixedResponse string
}

func (d *Dummy) Completion(prompt Prompt) (string, *ClientError) {
	return d.FixedResponse, nil
}

func NewChatGPT(token string) *ChatGPT {
	return &ChatGPT{
		Token: token,
	}
}

func NewDummyClient(fixedResponse string) *Dummy {
	return &Dummy{
		FixedResponse: fixedResponse,
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

func (c *ChatGPT) Completion(prompt Prompt) (string, *ClientError) {
	messages := MessageFromPrompt(prompt)
	req, err := CreateChatGPTRequest(c.Token, messages)
	if err != nil {
		return "", GenericError(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", GenericError(err)
	}
	if resp.StatusCode == http.StatusBadRequest {
		return "", ErrorBadRequest()
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return "", ErrorUnauthorized()
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", ErrorRateLimitExceeded(*resp)
	}

	if resp.StatusCode != http.StatusOK {
		return "", &ClientError{
			err:        errors.New("unknown error"),
			statusCode: resp.StatusCode,
		}
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

// You can expect to see the following header fields:
// Field	Sample Value	Description
// x-ratelimit-limit-requests	60	The maximum number of requests that are permitted before exhausting the rate limit.
// x-ratelimit-limit-tokens	150000	The maximum number of tokens that are permitted before exhausting the rate limit.
// x-ratelimit-remaining-requests	59	The remaining number of requests that are permitted before exhausting the rate limit.
// x-ratelimit-remaining-tokens	149984	The remaining number of tokens that are permitted before exhausting the rate limit.
// x-ratelimit-reset-requests	1s	The time until the rate limit (based on requests) resets to its initial state.
// x-ratelimit-reset-tokens	6m0s	The time until the rate limit (based on tokens) resets to its initial state.

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

func ParseResponse(r io.Reader) (string, *ClientError) {
	resp := ChatCompletionResponse{}
	err := json.NewDecoder(r).Decode(&resp)
	if err != nil {
		return "", GenericError(err)
	}
	if len(resp.Choices) < 1 {
		return "", GenericError(errors.New("no choices returned"))
	}
	return resp.Choices[0].Message.Content, nil
}
