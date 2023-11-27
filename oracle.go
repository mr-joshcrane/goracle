package oracle

import (
	"bytes"
	"context"
	"image"
	"io"
	"strings"

	"github.com/mr-joshcrane/oracle/client"
)

// Prompt is a struct that scaffolds a well formed prompt, designed in a way
// that are ideal for Large Language Models.
type Prompt struct {
	Purpose       string
	InputHistory  []string
	OutputHistory []string
	References    []io.Reader
	Question      string
}

type Transform struct {
	Source io.Reader
	Target io.ReadWriter
}

func (t Transform) GetSource() io.Reader {
	return t.Source
}

func (t Transform) GetTarget() io.ReadWriter {
	return t.Target
}

// GetPurpose returns the purpose of the prompt, which frames the models response.
func (p Prompt) GetPurpose() string {
	return p.Purpose
}

// GetHistory returns a list of examples that are used to guide the Models
// response. Quality of the examples is more important than quantity here.
func (p Prompt) GetHistory() ([]string, []string) {
	return p.InputHistory, p.OutputHistory
}

// GetQuestion returns the question that the user is asking the Model
func (p Prompt) GetQuestion() string {
	return p.Question
}

func (p Prompt) GetReferences() []io.Reader {
	return p.References
}

// LanguageModel is an interface that abstracts a concrete implementation of our
// language model API call.
type LanguageModel interface {
	Completion(ctx context.Context, prompt client.Prompt) (io.Reader, error)
	Transform(ctx context.Context, transform client.Transform) error
}

// Oracle is a struct that scaffolds a well formed Oracle, designed in a way
// that facilitates the asking of one or many questions to an underlying Large
// Language Model.
type Oracle struct {
	purpose         string
	previousInputs  []string
	previousOutputs []string
	client          LanguageModel
	artifacts       map[string]image.Image
}

// Options is a function that modifies the Oracle.
type Option func(*Oracle) *Oracle

func WithClient(client LanguageModel) Option {
	return func(o *Oracle) *Oracle {
		o.client = client
		return o
	}
}

// NewOracle returns a new Oracle with sensible defaults.
func NewOracle(token string, opts ...Option) *Oracle {
	client := client.NewChatGPT(token)
	o := &Oracle{
		purpose:   "You are a helpful assistant",
		client:    client,
		artifacts: map[string]image.Image{},
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// GeneratePrompt generates a prompt from the Oracle's purpose, examples, and
// question the current question posed by the user.
func (o *Oracle) GeneratePrompt(question string, references ...io.Reader) Prompt {
	p := Prompt{
		Purpose:       o.purpose,
		InputHistory:  o.previousInputs,
		OutputHistory: o.previousOutputs,
		Question:      question,
	}
	p.References = append(p.References, references...)
	return p
}

// SetPurpose sets the purpose of the Oracle, which frames the models response.
func (o *Oracle) SetPurpose(purpose string) {
	o.purpose = purpose
}

// GiveExample adds an example to the list of examples. These examples used to guide the models
// response. Quality of the examples is more important than quantity here.
func (o *Oracle) GiveExample(givenInput string, idealCompletion string) {
	o.previousInputs = append(o.previousInputs, givenInput)
	o.previousOutputs = append(o.previousOutputs, idealCompletion)
}

type Reference interface {
	Read([]byte) (int, error)
}

type DocumentRef struct {
	contents io.Reader
}

func (i DocumentRef) Read(v []byte) (int, error) {
	return i.contents.Read(v)
}

func NewDocument(r io.Reader) DocumentRef {
	return DocumentRef{
		contents: r,
	}
}

// Ask asks the Oracle a question, and returns the response from the underlying
// Large Language Model.
func (o Oracle) Ask(ctx context.Context, question string, references ...io.Reader) (string, error) {
	prompt := o.GeneratePrompt(question, references...)
	data, err := o.Completion(ctx, prompt)
	if err != nil {
		return "", err
	}
	answer, err := io.ReadAll(data)
	if err != nil {
		return "", err
	}
	return string(answer), nil
}

func (o Oracle) SpeechToText(ctx context.Context, speech io.Reader) (string, error) {
	out := new(bytes.Buffer)
	err := o.client.Transform(ctx, Transform{
		Source: speech,
		Target: out,
	})
	return out.String(), err
}

func (o Oracle) TextToSpeech(ctx context.Context, text string) (io.Reader, error) {
	out := new(bytes.Buffer)
	err := o.Transform(ctx, Transform{
		Source: strings.NewReader(text),
		Target: out,
	})
	return out, err
}

// Completion is a wrapper around the underlying Large Language Model API call.
func (o Oracle) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	return o.client.Completion(ctx, prompt)
}

func (o Oracle) Transform(ctx context.Context, transform Transform) error {
	return o.client.Transform(ctx, transform)
}

// Reset clears the Oracle's previous chat history
// Useful for when you hit a context limit
func (o *Oracle) Reset() {
	o.purpose = ""
	o.previousInputs = []string{}
	o.previousOutputs = []string{}
	o.artifacts = make(map[string]image.Image)
}
