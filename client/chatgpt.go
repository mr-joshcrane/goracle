package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

const (
	MessageText  = "text"
	MessageImage = "image"
	MessageAudio = "audio"
)

// --- Prompts and Messages
type Prompt interface {
	GetPurpose() string
	GetHistory() ([]string, []string)
	GetQuestion() string
	GetPages() [][]byte
}

type Transform interface {
	GetSource() io.Reader
	GetTarget() io.ReadWriter
}

type Messages []Message

type Message interface {
	GetFormat() string
}

type TextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (t TextMessage) GetFormat() string {
	return MessageText
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

type Artifact struct {
	Contents io.ReadWriter
}

func (a Artifact) Write(p []byte) (int, error) {
	return a.Contents.Write(p)
}

func (a Artifact) Read(p []byte) (int, error) {
	return a.Contents.Read(p)
}

// --- Dummy Client
type Dummy struct {
	FixedResponse string
	Failure       error
	T             Transform
	P             Prompt
}

func NewDummyClient(FixedResponse string, err error) *Dummy {
	return &Dummy{
		FixedResponse: FixedResponse,
		Failure:       err,
	}
}

func (d *Dummy) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	d.P = prompt
	return strings.NewReader(d.FixedResponse), d.Failure
}

func (d *Dummy) Transform(ctx context.Context, transform Transform) error {
	d.T = transform
	_, err := io.Copy(transform.GetTarget(), transform.GetSource())
	return err
}

// --- ChatGPT Client

type ChatGPT struct {
	Token string
	Model string
}

func addDefaultHeaders(token string, r *http.Request) *http.Request {
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+token)
	return r
}
func NewChatGPT(token string) *ChatGPT {
	return &ChatGPT{
		Token: token,
		Model: GPT4,
	}
}

func (c *ChatGPT) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	messages := MessageFromPrompt(prompt)
	for _, message := range messages {
		if message.GetFormat() == MessageImage {
			return c.visionCompletion(ctx, messages)
		}
	}
	return textCompletion(ctx, c.Token, messages)
}

func (c *ChatGPT) Transform(ctx context.Context, transform Transform) error {
	source := transform.GetSource()
	if _, ok := source.(*strings.Reader); ok {
		return c.textToSpeech(ctx, transform)
	}
	if _, ok := source.(*bytes.Buffer); ok {
		return c.speechToText(ctx, transform)
	}
	return fmt.Errorf("unknown transform source, %v", reflect.TypeOf(source).String())
}
