package client

import (
	"context"
	"image"
	"io"
	"net/http"
	"net/url"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// --- Prompts and Messages
type Prompt interface {
	GetPurpose() string
	GetExamples() ([]string, []string)
	GetQuestion() string
	GetImages() []image.Image
	GetUrls() []url.URL
	GetTarget() io.Writer
	GetSource() io.Reader
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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

// --- Dummy Client
type Dummy struct {
	FixedResponse  string
	FixedHTTPError int
}

func NewDummyClient(fixedResponse string, errorCode int) *Dummy {
	return &Dummy{
		FixedResponse:  fixedResponse,
		FixedHTTPError: errorCode,
	}
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

// --- ChatGPT Client

type ChatGPT struct {
	Token string
	Model string
}

func NewChatGPT(token string) *ChatGPT {
	return &ChatGPT{
		Token: token,
		Model: GPT4,
	}
}

func (c *ChatGPT) Completion(ctx context.Context, prompt Prompt) (string, error) {
	var images []string

	for _, image := range prompt.GetImages() {
		dataURI, err := ImageToDataURI(image)
		if err != nil {
			return "", err
		}
		images = append(images, dataURI)
	}

	for _, url := range prompt.GetUrls() {
		dataURI, err := URLToURI(url)
		if err != nil {
			continue
		}
		images = append(images, dataURI)
	}
	if len(images) > 0 {
		return c.visionCompletion(ctx, prompt.GetQuestion(), images...)
	}
	return c.standardCompletion(ctx, prompt)
}
