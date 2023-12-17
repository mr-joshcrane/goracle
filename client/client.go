package client

import (
	"context"
	"io"
	"strings"

	"github.com/mr-joshcrane/goracle/client/google"
	"github.com/mr-joshcrane/goracle/client/openai"
)

// --- Prompts and Messages
type Prompt interface {
	GetPurpose() string
	GetHistory() ([]string, []string)
	GetQuestion() string
	GetReferences() [][]byte
}

// --- Dummy Client
type Dummy struct {
	FixedResponse string
	Failure       error
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
	return openai.Do(ctx, c.Token, prompt)
}

func (c *ChatGPT) CreateImage(ctx context.Context, prompt string) ([]byte, error) {
	return openai.DoImageRequest(ctx, c.Token, prompt)
}

func (c *ChatGPT) CreateTranscript(ctx context.Context, audio []byte) (string, error) {
	return openai.SpeechToText(ctx, c.Token, audio)
}

func (c *ChatGPT) CreateAudio(ctx context.Context, text string) ([]byte, error) {
	return openai.TextToSpeech(ctx, c.Token, text)
}

// --- Vertex client

type Vertex struct {
	Token     string
	ProjectID string
}

func NewVertex() *Vertex {
	return &Vertex{}
}

func (v *Vertex) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	return google.Completion(ctx, prompt)
}
