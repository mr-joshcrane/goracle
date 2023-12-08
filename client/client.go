package client

import (
	"context"
	"io"
	"strings"

	"github.com/mr-joshcrane/oracle/client/openai"
	"github.com/mr-joshcrane/oracle/client/vertex"
)

// --- Prompts and Messages
type Prompt interface {
	GetPurpose() string
	GetHistory() ([]string, []string)
	GetQuestion() string
	GetPages() [][]byte
}

// --- Dummy Client
type Dummy struct {
	FixedResponse string
	Failure       error
	P             Prompt
}

func NewDummyClient(FixedResponse string, err error) *Dummy {
	return &Dummy{
		FixedResponse: FixedResponse,
		Failure:       err,
	}
}

func (d *Dummy) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	d.P = prompt
	return strings.NewReader(d.FixedResponse), d.Failure
}

// --- ChatGPT Client

type ChatGPT struct {
	Token string
	Model string
}

func NewChatGPT(token string) *ChatGPT {
	return &ChatGPT{
		Token: token,
	}
}

func (c *ChatGPT) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	return openai.Do(ctx, c.Token, prompt)
}

// --- Vertex client

type Vertex struct {
	Token     string
	ProjectID string
}

func NewVertex(token string, projectID string) *Vertex {
	return &Vertex{
		Token:     token,
		ProjectID: projectID,
	}
}
func (v *Vertex) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	return vertex.Completion(ctx, v.Token, vertex.ProjectID, prompt)
}
