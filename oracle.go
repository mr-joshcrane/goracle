package oracle

import (
	"github.com/mr-joshcrane/oracle/client"
)

type Prompt struct {
	Purpose  string
	Examples Exemplars
	Question string
}

func (p Prompt) GetPurpose() string {
	return p.Purpose
}

func (p Prompt) GetExamples() []struct{ GivenInput, IdealOutput string } {
	examples := []struct{ GivenInput, IdealOutput string }{}
	for _, exemplar := range p.Examples {
		examples = append(examples, struct{ GivenInput, IdealOutput string }{exemplar.GivenInput, exemplar.IdealOutput})
	}
	return examples
}

func (p Prompt) GetQuestion() string {
	return p.Question
}

type Exemplars []struct {
	GivenInput  string
	IdealOutput string
}

type LanguageModel interface {
	Completion(prompt client.Prompt) (string, error)
}

type Oracle struct {
	purpose  string
	examples Exemplars
	client   LanguageModel
}

func NewOracle() *Oracle {
	client := client.NewChatGPT()
	return &Oracle{
		purpose:  "You are a helpful assistant",
		examples: Exemplars{},
		client:   client,
	}
}

func (o Oracle) GetPurpose() string {
	return o.purpose
}

func (o Oracle) GetExamples() Exemplars {
	return o.examples
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
