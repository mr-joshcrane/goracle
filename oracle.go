package oracle

import (
	"github.com/mr-joshcrane/oracle/client"
)

// Prompt is a struct that scaffolds a well formed prompt, designed in a way
// that are ideal for Large Language Models.
type Prompt struct {
	Purpose  string
	Examples map[string]string
	Question string
}

// GetPurpose returns the purpose of the prompt, which frames the models response.
func (p Prompt) GetPurpose() string {
	return p.Purpose
}

// GetExamples returns a list of examples that are used to guide the Models
// response. Quality of the examples is more important than quantity here.
func (p Prompt) GetExamples() map[string]string {
	return p.Examples
}

// GetQuestion returns the question that the user is asking the Model
func (p Prompt) GetQuestion() string {
	return p.Question
}

// LanguageModel is an interface that abstracts a concrete implementation of are
// language model API call.
type LanguageModel interface {
	Completion(prompt client.Prompt) (string, error)
}

// Oracle is a struct that scaffolds a well formed Oracle, designed in a way
// that facilitates the asking of one or many questions to an underlying Large
// Language Model.
type Oracle struct {
	purpose  string
	examples map[string]string
	client   LanguageModel
}

// NewOracle returns a new Oracle with sensible defaults.
func NewOracle() *Oracle {
	client := client.NewChatGPT()
	return &Oracle{
		purpose:  "You are a helpful assistant",
		examples: map[string]string{},
		client:   client,
	}
}

// GeneratePrompt generates a prompt from the Oracle's purpose, examples, and
// question the current question posed by the user.
func (o *Oracle) GeneratePrompt(question string) Prompt {
	return Prompt{
		Purpose:  o.purpose,
		Examples: o.examples,
		Question: question,
	}
}

// SetPurpose sets the purpose of the Oracle, which frames the models response.
func (o *Oracle) SetPurpose(purpose string) {
	o.purpose = purpose
}

// GiveExample adds an example to the list of examples. These examples used to guide the models
// response. Quality of the examples is more important than quantity here.
func (o *Oracle) GiveExample(givenInput string, idealCompletion string) {
	o.examples[givenInput] = idealCompletion
}

// Ask asks the Oracle a question, and returns the response from the underlying
// Large Language Model.
func (o Oracle) Ask(question string) (string, error) {
	prompt := o.GeneratePrompt(question)
	return o.client.Completion(prompt)
}
