package goracle

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/mr-joshcrane/goracle/client"
)

// Prompt is a struct that scaffolds a well formed prompt, designed in a way
// that are ideal for Large Language Models. This is the abstraction we will pass
// through to the client library so it can be handled appropriately
type Prompt struct {
	Purpose       string
	InputHistory  []string
	OutputHistory []string
	References    [][]byte
	Question      string
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

func (p Prompt) GetReferences() [][]byte {
	return p.References
}

// LanguageModel is an interface that abstracts a concrete implementation of our
// language model API call.
type LanguageModel interface {
	Completion(ctx context.Context, prompt client.Prompt) (io.Reader, error)
}

// Oracle is a struct that scaffolds a well formed Oracle, designed in a way
// that facilitates the asking of one or many questions to an underlying Large
// Language Model.
type Oracle struct {
	purpose         string
	previousInputs  []string
	previousOutputs []string
	client          LanguageModel
	stateful        bool
}

// Options is a function that modifies the Oracle.
type Option func(*Oracle) *Oracle

func Stateful(*Oracle) *Oracle {
	return &Oracle{
		stateful: true,
	}
}

func Stateless(*Oracle) *Oracle {
	return &Oracle{
		previousInputs:  []string{},
		previousOutputs: []string{},
		stateful:        false,
	}
}

// NewOracle returns a new Oracle with sensible defaults.
func NewOracle(client LanguageModel) *Oracle {
	return &Oracle{
		client:   client,
		purpose:  "You are a helpful assistant",
		stateful: true,
	}
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

// Ask asks the Oracle a question, and returns the response from the underlying
// Large Language Model.
func (o *Oracle) Ask(ctx context.Context, question string, references ...any) (string, error) {
	p := Prompt{
		Purpose:       o.purpose,
		InputHistory:  o.previousInputs,
		OutputHistory: o.previousOutputs,
		Question:      question,
	}
	for _, reference := range references {
		switch r := reference.(type) {
		case []byte:
			p.References = append(p.References, r)
		case string:
			p.References = append(p.References, []byte(r))
		case image.Image:
			p.References = append(p.References, Image(r))
		default:
			return "", fmt.Errorf("unprocessable reference type: %T", r)
		}
	}
	data, err := o.completion(ctx, p)
	if err != nil {
		return "", err
	}
	answer, err := io.ReadAll(data)
	if err != nil {
		return "", err
	}
	if o.stateful {
		o.GiveExample(question, string(answer))
	}
	return string(answer), nil
}

// Completion is a wrapper around the underlying Large Language Model API call.
func (o Oracle) completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	return o.client.Completion(ctx, prompt)
}

// Reset clears the Oracle's previous chat history
// Useful for when you hit a context limit
func (o *Oracle) Reset() {
	o.purpose = ""
	o.previousInputs = []string{}
	o.previousOutputs = []string{}
}

func Folder(root string) []byte {
	contents := []byte{}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		content := append(File(path), byte('\n'))
		contents = append(contents, content...)
		return nil
	})
	return contents
}

func File(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return []byte{}
	}
	return data
}

func Image(i image.Image) []byte {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, i)
	if err != nil {
		return []byte{}
	}
	return buf.Bytes()
}
