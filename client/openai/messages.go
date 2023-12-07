package openai

import (
	"fmt"
	"net/http"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

type Messages []Message

type Message interface {
}

type TextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func addDefaultHeaders(token string, r *http.Request) *http.Request {
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+token)
	return r
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
		messages = append(messages, TextMessage{
			Role:    RoleUser,
			Content: fmt.Sprintf("Reference %d: %s", i, page),
		})
	}
	return messages
}
