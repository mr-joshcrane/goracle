package vertex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	VertexURI = "https://us-central1-aiplatform.googleapis.com/v1/projects/aisandbox-407405/locations/us-central1/publishers/google/models/text-bison:predict"
	ProjectID = "aisandbox-407405"
)

type VertexAI struct {
	token string
}

func (v VertexAI) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	req, err := CreateVertexTextCompletionRequest(v.token, "", Messages{})
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	answer, err := ParseVertexTextCompletionReponse(*resp)
	if err != nil {
		return nil, err
	}
	return answer, err
}

func (v VertexAI) Transform(ctx context.Context, transform Transform) error {
	return nil
}

func NewVertexAI(token string) *VertexAI {
	return &VertexAI{
		token: token,
	}
}

type Request struct {
	Instances  []Instance `json:"instances"`
	Parameters Parameters `json:"parameters"`
}

type Instance struct {
	Content string `json:"content"`
}

type Parameters struct {
	CandidateCount  int     `json:"candidateCount"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"topP"`
	TopK            int     `json:"topK"`
}

func CreateVertexTextCompletionRequest(token string, model string, messages Messages) (*http.Request, error) {
	body := Request{
		Instances: []Instance{
			{
				Content: "Whats up",
			},
		},
		Parameters: Parameters{
			CandidateCount:  1,
			MaxOutputTokens: 20,
			Temperature:     0.8,
			TopP:            0.95,
			TopK:            40,
		},
	}

	d, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	data := bytes.NewReader(d)

	req, err := http.NewRequest(http.MethodPost, VertexURI, data)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	return req, nil
}

func ParseVertexTextCompletionReponse(resp http.Response) (io.Reader, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	body := struct {
		Predictions []struct {
			Content string `json:"content"`
		} `json:"predictions"`
	}{}

	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	if len(body.Predictions) < 1 {
		return nil, fmt.Errorf("no predictions returned")
	}
	answer := body.Predictions[0].Content
	return strings.NewReader(answer), nil
}
