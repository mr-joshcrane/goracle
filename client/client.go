package client

import (
	"context"
	"io"
	"strings"

	"github.com/mr-joshcrane/goracle/client/google"
	"github.com/mr-joshcrane/goracle/client/ollama"
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
	fixedResponse string
	Failure       error
	P             Prompt
}

func NewDummyClient(fixedResponse string, err error) *Dummy {
	return &Dummy{
		fixedResponse: fixedResponse,
		Failure:       err,
	}
}

func (d *Dummy) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	d.P = prompt
	return strings.NewReader(d.fixedResponse), d.Failure
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

// --- Ollama client

type Ollama struct {
	Model    string
	Endpoint string
}

func NewOllama(model string, endpoint string) *Ollama {
	return &Ollama{
		Model:    model,
		Endpoint: endpoint,
	}
}

func (o *Ollama) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	answer, err := ollama.DoChatCompletion(o.Model, o.Endpoint, prompt)
	if err != nil {
		return nil, err
	}
	return strings.NewReader(answer), nil
}

func (o *Ollama) GenerateEmbedding(ctx context.Context, prompt Prompt) ([]float64, error) {
	return ollama.GetEmbedding(o.Model, o.Endpoint, prompt)
}
