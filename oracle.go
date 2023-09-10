package oracle

import (
	"time"

	"github.com/mr-joshcrane/oracle/client"
)

// Prompt is a struct that scaffolds a well formed prompt, designed in a way
// that are ideal for Large Language Models.
type Prompt struct {
	Purpose       string
	ExampleInputs []string
	IdealOutputs  []string
	Question      string
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

// LanguageModel is an interface that abstracts a concrete implementation of are
// language model API call.
type LanguageModel interface {
	Completion(prompt client.Prompt) (string, client.ClientError)
}

type ClientError interface {
	Error() error
	StatusCode() int
	RetryIn() time.Duration
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

func WithDummyClient(fixedResponse string) Option {
	return func(o *Oracle) *Oracle {
		o.client = client.NewDummyClient(fixedResponse)
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
func (o Oracle) Ask(question string) (string, ClientError) {
	prompt := o.GeneratePrompt(question)
	return o.Completion(prompt)
}

func (o Oracle) Completion(prompt Prompt) (string, ClientError) {
	return o.client.Completion(prompt)
}
