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
	"strings"

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

// Remember [Oracles Oracle] remember the conversation history and keep track
// of the context of the conversation. This is the default behaviour. References
// are not persisted between calls in order to keep the prompt size down.
// If this behaviour is desired, you can pass the references with
// oracle.GiveExample like so:
// oracle.GiveExample(oracle.File("path/to/file", "<your preferred bot response>"))
func (o *Oracle) Remember() *Oracle {
	o.stateful = true
	return o
}

// Forget [Oracles Oracle] forget the conversation history and do not keep
// track of the context of the conversation. This is useful for when you want
// to ask a single question without previous context affecting the answers.
func (o *Oracle) Forget() *Oracle {
	o.Reset()
	o.stateful = false
	return o
}

// Reset clears the Oracle's previous chat history
// Useful for when you hit a context limit
// Doesn't affect the Oracle's purpose, or whether it's stateful or not
func (o *Oracle) Reset() {
	o.previousInputs = []string{}
	o.previousOutputs = []string{}
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
// Calling this method on a stateless Oracle will have no effect.
// This allows for stateless oracles to still benefit from n-shot learning.
func (o *Oracle) GiveExample(givenInput string, idealCompletion string) {
	o.previousInputs = append(o.previousInputs, givenInput)
	o.previousOutputs = append(o.previousOutputs, idealCompletion)
}

// Ask asks the Oracle a question, and returns the response from the underlying
// Large Language Model. Ask massages the query and supporting references into a
// standardised format that is relatively generalisable across models.
func (o *Oracle) Ask(question string, references ...any) (string, error) {
	return o.AskWithContext(context.Background(), question, references...)
}

// AskWithContext is similar to [*Oracle.Ask] but allows for a context to be passed in.
func (o *Oracle) AskWithContext(ctx context.Context, question string, references ...any) (string, error) {
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

// A Reference helper that reads a file from disk and returns the contents as
// an array of bytes. Content will be a snapshot of the file at the time of
// calling. Consider calling inside the Ask method or using a closure to ensure
// lazy evaluation if you're going to be editing read file in place.
func File(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return []byte{}
	}
	return data
}

// A Reference helper that takes a folder in a filesystem and returns the contents
// of all files in that folder as an array of bytes. Content will be a snapshot
// of the files at the time of calling. Call is recursive, so be careful with
// what you include. Consider adding one of more filters to the includeFilter
// such as ".go" to only include certain files or similar globs.
func Folder(root string, includeFilter ...string) []byte {
	contents := []byte{}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		var filter bool
		if len(includeFilter) != 0 {
			filter = true
		}
		for _, f := range includeFilter {
			if strings.Contains(info.Name(), f) {
				filter = false
			}
		}
		if !filter {
			content := append(File(path), byte('\n'))
			contents = append(contents, content...)
		}
		return nil
	})
	return contents
}

// A Reference helper that takes an image and returns it's contents as an array
// of bytes. Currently encodes to PNGs to be passed to the upstream client.
// Content will be a snapshot of the image at the time of calling.
func Image(i image.Image) []byte {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, i)
	if err != nil {
		return []byte{}
	}
	return buf.Bytes()
}

// NewChatGPTOracle takes an OpenAI API token and sets up a new ChatGPT Oracle
// with sensible defaults.
func NewChatGPTOracle(token string) *Oracle {
	return NewOracle(client.NewChatGPT(token))
}

// NewGoogleGeminiOracle uses the
func NewGoogleGeminiOracle() *Oracle {
	return NewOracle(client.NewVertex())
}

func NewAnthropicOracle(token string) *Oracle {
	return NewOracle(client.NewAnthropic(token))
}

func NewOllamaOracle(model string, endpoint string) *Oracle {
	return NewOracle(client.NewOllama(model, endpoint))
}

func (o *Oracle) WithModel(model string) error {
	switch c := o.client.(type) {
	case *client.ChatGPT:
		return c.WithModel(model)
	case *client.Vertex:
		return c.WithModel(model)
	case *client.Anthropic:
		return c.WithModel(model)
	default:
		return fmt.Errorf("model switching not supported for %T", c)
	}
}
