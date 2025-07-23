package client

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mr-joshcrane/goracle/client/anthropic"
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
	GetResponseFormat() []string
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
	Model openai.ModelConfig
}

func NewChatGPT(token string) *ChatGPT {
	return &ChatGPT{
		Token: token,
		Model: openai.Models["gpt-4.1"],
	}
}

func (c *ChatGPT) WithModel(model string) error {
	m, ok := openai.Models[model]
	if !ok {
		supportedModels := make([]string, 0, len(openai.Models))
		for k := range openai.Models {
			supportedModels = append(supportedModels, k)
		}
		return fmt.Errorf("model %s not found. Supported models include: %s", model, strings.Join(supportedModels, ", "))
	}
	c.Model = m
	return nil
}

func (c *ChatGPT) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	return openai.Do(ctx, c.Token, c.Model, prompt)
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
	Model     google.ModelConfig
}

func NewVertex() *Vertex {
	return &Vertex{
		Model: google.Models["GeminiPro"],
	}
}

func (v *Vertex) WithModel(model string) error {
	m, ok := google.Models[model]
	if !ok {
		supportedModels := make([]string, 0, len(google.Models))
		for k := range google.Models {
			supportedModels = append(supportedModels, k)
		}
		return fmt.Errorf("model %s not found. Supported models include: %s", model, strings.Join(supportedModels, ", "))
	}
	v.Model = m
	return nil
}

func (v *Vertex) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	if v.ProjectID == "" || v.Token == "" {
		project, token, err := google.Authenticate()
		if err != nil {
			return nil, err
		}
		v.ProjectID = project
		v.Token = token
	}
	return google.Completion(ctx, v.Token, v.ProjectID, v.Model, prompt)
}

// --- Anthropic client

type Anthropic struct {
	Token string
	Model anthropic.ModelConfig
}

func NewAnthropic(token string) *Anthropic {
	return &Anthropic{
		Token: token,
		Model: anthropic.Models["ClaudeSonnet3_7"],
	}
}

func (a *Anthropic) WithModel(model string) error {
	m, ok := anthropic.Models[model]
	if !ok {
		supportedModels := make([]string, 0, len(anthropic.Models))
		for k := range anthropic.Models {
			supportedModels = append(supportedModels, k)
		}
		return fmt.Errorf("model %s not found. Supported models include: %s", model, strings.Join(supportedModels, ", "))
	}
	a.Model = m
	return nil
}

func (a *Anthropic) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	if a.Token == "" {
		token, err := anthropic.Authenticate()
		if err != nil {
			return nil, err
		}
		a.Token = token
	}
	return anthropic.Completion(ctx, a.Token, a.Model, prompt)
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
