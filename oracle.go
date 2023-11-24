package oracle

import (
	"context"
	"image"
	"io"
	"net/url"

	"github.com/mr-joshcrane/oracle/client"
)

// Prompt is a struct that scaffolds a well formed prompt, designed in a way
// that are ideal for Large Language Models.
type Prompt struct {
	Purpose       string
	ExampleInputs []string
	IdealOutputs  []string
	Question      string
	Images        []image.Image
	Urls          []url.URL
	Target        io.Writer
	Source        io.Reader
}

// GetPurpose returns the purpose of the prompt, which frames the models response.
func (p Prompt) GetPurpose() string {
	return p.Purpose
}

// GetExamples returns a list of examples that are used to guide the Models
// response. Quality of the examples is more important than quantity here.
func (p Prompt) GetExamples() ([]string, []string) {
	return p.ExampleInputs, p.IdealOutputs
}

// GetQuestion returns the question that the user is asking the Model
func (p Prompt) GetQuestion() string {
	return p.Question
}

// GetImages returns the images that the user is asking the Model to compare
func (p Prompt) GetImages() []image.Image {
	return p.Images
}

func (p Prompt) GetUrls() []url.URL {
	return p.Urls
}

func (p Prompt) GetTarget() io.Writer {
	return p.Target
}

func (p Prompt) GetSource() io.Reader {
	return p.Source
}

// LanguageModel is an interface that abstracts a concrete implementation of our
// language model API call.
type LanguageModel interface {
	Completion(ctx context.Context, prompt client.Prompt) (string, error)
}

// Oracle is a struct that scaffolds a well formed Oracle, designed in a way
// that facilitates the asking of one or many questions to an underlying Large
// Language Model.
type Oracle struct {
	purpose       string
	exampleInputs []string
	idealOutputs  []string
	client        LanguageModel
}

// Options is a function that modifies the Oracle.
type Option func(*Oracle) *Oracle

func WithDummyClient(fixedResponse string, responseCode int) Option {
	return func(o *Oracle) *Oracle {
		o.client = client.NewDummyClient(fixedResponse, responseCode)
		return o
	}
}

// NewOracle returns a new Oracle with sensible defaults.
func NewOracle(token string, opts ...Option) *Oracle {
	client := client.NewChatGPT(token)
	o := &Oracle{
		purpose: "You are a helpful assistant",
		client:  client,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type Asset struct {
	Type  string
	Value any
}

func WithImages(images ...image.Image) []Asset {
	assets := []Asset{}
	for _, image := range images {
		assets = append(assets, Asset{
			Type:  "image",
			Value: image,
		})
	}
	return assets
}

func WithURLs(urls ...url.URL) []Asset {
	assets := []Asset{}
	for _, u := range urls {
		assets = append(assets, Asset{
			Type:  "url",
			Value: u,
		})
	}
	return assets
}

func WithTarget(target io.Writer) []Asset {
	return []Asset{
		{
			Type:  "target",
			Value: target,
		},
	}
}

func WithSource(source io.Reader) []Asset {
	return []Asset{
		{
			Type:  "source",
			Value: source,
		},
	}
}

// GeneratePrompt generates a prompt from the Oracle's purpose, examples, and
// question the current question posed by the user.
func (o *Oracle) GeneratePrompt(question string) Prompt {
	return Prompt{
		Purpose:       o.purpose,
		ExampleInputs: o.exampleInputs,
		IdealOutputs:  o.idealOutputs,
		Question:      question,
	}
}

func DescribeImagePrompt(question string, items ...[]Asset) Prompt {
	images := []image.Image{}
	urls := []url.URL{}
	for _, item := range items {
		for _, asset := range item {
			switch asset.Type {
			case "image":
				images = append(images, asset.Value.(image.Image))
			case "url":
				urls = append(urls, asset.Value.(url.URL))
			}
		}
	}
	return Prompt{
		Question: question,
		Images:   images,
		Urls:     urls,
	}
}

func CreateTranscriptPrompt(source io.Reader) Prompt {
	return Prompt{
		Source: source,
	}
}

func CreateAudioPrompt(source io.Reader, target io.Writer) Prompt {
	return Prompt{
		Source: source,
		Target: target,
	}
}

func CreateImagePrompt(question string, target io.Writer) Prompt {
	return Prompt{
		Question: question,
		Target:   target,
	}
}

// SetPurpose sets the purpose of the Oracle, which frames the models response.
func (o *Oracle) SetPurpose(purpose string) {
	o.purpose = purpose
}

// GiveExample adds an example to the list of examples. These examples used to guide the models
// response. Quality of the examples is more important than quantity here.
func (o *Oracle) GiveExample(givenInput string, idealCompletion string) {
	o.exampleInputs = append(o.exampleInputs, givenInput)
	o.idealOutputs = append(o.idealOutputs, idealCompletion)
}

// Ask asks the Oracle a question, and returns the response from the underlying
// Large Language Model.
func (o Oracle) Ask(ctx context.Context, question string) (string, error) {
	prompt := o.GeneratePrompt(question)
	return o.Completion(ctx, prompt)
}

func (o Oracle) DescribeImage(ctx context.Context, question string, asset ...Asset) (string, error) {
	prompt := DescribeImagePrompt(question, asset)
	return o.Completion(ctx, prompt)
}

func (o Oracle) CreateImage(ctx context.Context, question string, target io.Writer) error {
	prompt := CreateImagePrompt(question, target)
	_, err := o.Completion(ctx, prompt)
	return err
}

func (o Oracle) CreateTranscript(ctx context.Context, source io.Reader) (string, error) {
	prompt := CreateTranscriptPrompt(source)
	return o.Completion(ctx, prompt)
}

func (o Oracle) CreateAudio(ctx context.Context, question string, target io.Writer) error {
	prompt := o.GeneratePrompt(question)
	_, err := o.Completion(ctx, prompt)
	return err
}

// Completion is a wrapper around the underlying Large Language Model API call.
func (o Oracle) Completion(ctx context.Context, prompt Prompt) (string, error) {
	return o.client.Completion(ctx, prompt)
}

// Reset clears the Oracle's previous chat history
// Useful for when you hit a context limit
func (o *Oracle) Reset() {
	o.purpose = ""
	o.exampleInputs = []string{}
	o.idealOutputs = []string{}
}
