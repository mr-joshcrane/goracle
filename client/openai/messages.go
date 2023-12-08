package openai

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

type Prompt interface {
	GetPurpose() string
	GetHistory() ([]string, []string)
	GetQuestion() string
	GetPages() [][]byte
}

type Messages []Message

type Message interface {
	GetFormat() string
}

type TextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (m TextMessage) GetFormat() string {
	return ""
}

func MessageFromPrompt(prompt Prompt) Messages {
	messages := []Message{}
	messages = append(messages, TextMessage{
		Role:    RoleSystem,
		Content: prompt.GetPurpose(),
	})
	givenInputs, idealOutputs := prompt.GetHistory()
	for i, givenInput := range givenInputs {
		messages = append(messages, TextMessage{
			Role:    RoleUser,
			Content: givenInput,
		})
		messages = append(messages, TextMessage{
			Role:    RoleAssistant,
			Content: idealOutputs[i],
		})
	}
	messages = append(messages, TextMessage{
		Role:    RoleUser,
		Content: prompt.GetQuestion(),
	})
	pages := prompt.GetPages()
	for i, page := range pages {
		i++
		if isPNG(page) {
			uri := ConvertPNGToDataURI(page)
			messages = append(messages, VisionMessage{
				Role:    RoleUser,
				Content: []map[string]string{{"type": "image_url", "image_url": uri}},
			})
			continue
		}
		messages = append(messages, TextMessage{
			Role:    RoleUser,
			Content: fmt.Sprintf("Reference %d: %s", i, page),
		})
	}
	return messages
}

func Do(ctx context.Context, token string, prompt Prompt) (io.Reader, error) {
	strategy := textCompletion
	pages := prompt.GetPages()
	for _, page := range pages {
		if isPNG(page) {
			strategy = visionCompletion
		}
	}
	messages := MessageFromPrompt(prompt)
	return strategy(ctx, token, messages)
}

func addDefaultHeaders(token string, r *http.Request) *http.Request {
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+token)
	return r
}
