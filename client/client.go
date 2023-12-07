package client

import (
	"context"
	"io"
	"strings"
)

const (
	Purpose      = "purpose"
	UserInput    = "user"
	UserProvided = "user-provided"
	Generated    = "generated"
)

// --- Prompts and Messages
type Prompt interface {
	GetPurpose() string
	GetHistory() ([]string, []string)
	GetQuestion() string
	GetPages() [][]byte
}

type Messages []Message

type Message struct {
	Content  string
	Origin   string
	MimeType string
}

func PromptToMessages(p Prompt) Messages {
	messages := Messages{
		Message{
			Content:  p.GetPurpose(),
			Origin:   Purpose,
			MimeType: "text/plain",
		},
	}
	previousInputs, previousOutputs := p.GetHistory()
	for i := range previousInputs {
		messages = append(messages, Message{
			Content:  previousInputs[i],
			Origin:   UserInput,
			MimeType: "text/plain",
		})
		messages = append(messages, Message{
			Content:  previousOutputs[i],
			Origin:   Generated,
			MimeType: "text/plain",
		})
	}
	for _, page := range p.GetPages() {
		messages = append(messages, Message{
			Content:  string(page),
			Origin:   UserProvided,
			MimeType: "text/plain",
		})
	}
	messages = append(messages, Message{
		Content:  p.GetQuestion(),
		Origin:   UserProvided,
		MimeType: "text/plain",
	})
	return messages
}

type Transform interface {
	GetSource() io.Reader
	GetTarget() io.ReadWriter
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

func NewChatGPT(token string) *ChatGPT {
	return &ChatGPT{
		Token: token,
	}
}

func (c *ChatGPT) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	messages := PromptToMessages(prompt)
	return openai.textCompletion(ctx, c.Token, messages)
}

func (c *ChatGPT) Transform(ctx context.Context, transform Transform) error {
	return nil
}
