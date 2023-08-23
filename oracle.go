package oracle

import (
	"github.com/mr-joshcrane/oracle/client"
)

// Prompt is a struct that scaffolds a well formed prompt, designed in a way
// that are ideal for Large Language Models.
type Prompt struct {
	Purpose  string
	Examples Exemplars
	Question string
}

// GetPurpose returns the purpose of the prompt, which frames the models response.
func (p Prompt) GetPurpose() string {
	return p.Purpose
}

// GetExamples returns a list of examples that are used to guide the Models
// response. Quality of the examples is more important than quantity here.
func (p Prompt) GetExamples() []struct{ GivenInput, IdealOutput string } {
	examples := []struct{ GivenInput, IdealOutput string }{}
	for _, exemplar := range p.Examples {
		examples = append(examples, struct{ GivenInput, IdealOutput string }{exemplar.GivenInput, exemplar.IdealOutput})
	}
	return examples
}

// GetQuestion returns the question that the user is asking the Model
func (p Prompt) GetQuestion() string {
	return p.Question
}

// Exemplars is a list of zero or more examples that are used to guide the
// Models response. Quality of the examples is more important than quantity
type Exemplars []struct {
	GivenInput  string
	IdealOutput string
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
	examples Exemplars
	client   LanguageModel
}

// NewOracle returns a new Oracle with sensible defaults.
func NewOracle() *Oracle {
	client := client.NewChatGPT()
	return &Oracle{
		purpose:  "You are a helpful assistant",
		examples: Exemplars{},
		client:   client,
	}
}

func (o *Oracle) GeneratePrompt(question string) Prompt {
	return Prompt{
		Purpose:  o.purpose,
		Examples: o.examples,
		Question: question,
	}
}

func (o *Oracle) SetPurpose(purpose string) {
	o.purpose = purpose
}

func (o *Oracle) GiveExample(givenInput string, idealCompletion string) {
	o.examples = append(o.examples, struct{ GivenInput, IdealOutput string }{givenInput, idealCompletion})
}

func (o Oracle) Ask(question string) (string, error) {
	prompt := o.GeneratePrompt(question)
	return o.client.Completion(prompt)
}
