package oracle

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
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
	Pages         References
	Artifacts     References
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

func (p Prompt) GetPages() ([]io.Reader, error) {
	references := []io.Reader{}
	errors := []error{}
	for i := range p.Pages {
		page, ok := p.Pages[i].(Page)
		if !ok {
			return nil, fmt.Errorf("error reading page")
		}
		data, err := page.GetContent()
		if err != nil {
			errors = append(errors, err)
			continue
		}
		references = append(references, bytes.NewReader(data))
	}
	if len(errors) > 0 {
		return nil, fmt.Errorf("error reading pages: %v", errors)
	}
	return references, nil
}

func (p Prompt) GetArtifacts() ([]io.ReadWriter, error) {
	artifacts := []io.ReadWriter{}
	errors := []error{}
	for _, artifact := range p.Artifacts {
		artifact, ok := artifact.(Artifact)
		if !ok {
			return nil, fmt.Errorf("error reading artifact")
		}
		_, err := artifact.contents.Write([]byte{}) // write nothing to test if it's writable
		if err != nil {
			errors = append(errors, err)
			continue
		}
		artifacts = append(artifacts, artifact.contents)
	}
	if len(errors) > 0 {
		return nil, fmt.Errorf("error reading artifacts: %v", errors)
	}
	return artifacts, nil
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
func (o *Oracle) GeneratePrompt(question string, references ...References) Prompt {
	var pages References
	var artifacts References
	p := Prompt{
		Purpose:       o.purpose,
		InputHistory:  o.previousInputs,
		OutputHistory: o.previousOutputs,
		Question:      question,
		Pages:         pages,
		Artifacts:     artifacts,
	}
	for _, reference := range references {
		reference.AddTo(&p)
	}

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

const (
	ReadOnlyRef  = "Page"
	ReadWriteRef = "Artifact"
)

type Reference interface {
	Describe() string
}

type References []Reference

func (r References) AddTo(p *Prompt) {
	for _, ref := range r {
		switch ref.Describe() {
		case ReadOnlyRef:
			p.Pages = append(p.Pages, ref)
		case ReadWriteRef:
			p.Artifacts = append(p.Artifacts, ref)
		}
	}
}

type Page interface {
	GetContent() ([]byte, error)
}

type ImagePage struct {
	Image image.Image
}

func (i ImagePage) Describe() string {
	return ReadOnlyRef
}

type Artifacts []Artifact

type Artifact struct {
	contents io.ReadWriter
}

func (a Artifact) Describe() string {
	return ReadWriteRef
}

func (i *ImagePage) GetContent() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, i.Image)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func NewVisuals(image image.Image, images ...image.Image) References {
	refs := []Reference{&ImagePage{Image: image}}
	for _, image := range images {
		refs = append(refs, &ImagePage{Image: image})
	}
	return refs
}

type DocumentPage struct {
	contents io.Reader
}

func (d DocumentPage) Describe() string {
	return ReadOnlyRef
}
func (i DocumentPage) GetContent() ([]byte, error) {
	data, err := io.ReadAll(i.contents)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func NewDocuments(r io.Reader, a ...io.Reader) References {
	refs := []Reference{}
	a = append(a, r)
	for _, doc := range a {
		d, ok := doc.(io.Seeker)
		if ok {
			_, _ = d.Seek(0, io.SeekStart)
		}
		refs = append(refs, DocumentPage{contents: doc})
	}
	return refs
}

func NewArtifacts(artifact io.ReadWriter, a ...io.ReadWriter) References {
	references := References{}
	references = append(references, Artifact{
		contents: artifact,
	})
	for _, artifact := range a {
		references = append(references, Artifact{
			contents: artifact,
		})
	}
	return references
}

// Ask asks the Oracle a question, and returns the response from the underlying
// Large Language Model.
func (o Oracle) Ask(ctx context.Context, question string, references ...References) (string, error) {
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
