package vertex

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	ProjectID = "aisandbox-407405"
)

type Prompt interface {
	GetPurpose() string
	GetHistory() ([]string, []string)
	GetQuestion() string
	GetPages() [][]byte
}

type Messages struct {
	Context  Context   `json:"context"`
	Examples Examples  `json:"examples"`
	Messages []Message `json:"messages"`
}
type Context string

type Examples []Example

type Example struct {
	Input struct {
		Content string `json:"content"`
	} `json:"input"`
	Output struct {
		Content string `json:"content"`
	} `json:"output"`
}

type Message struct {
	Author  string `json:"author"`
	Content string `json:"content"`
}

func MessagesFromPrompt(token string, prompt Prompt) Messages {
	instance := Messages{
		Examples: []Example{},
		Messages: []Message{},
	}
	instance.Context = Context(prompt.GetPurpose())
	givenInputs, idealOutputs := prompt.GetHistory()
	for i, givenInput := range givenInputs {
		instance.Examples = append(instance.Examples, Example{
			Input: struct {
				Content string `json:"content"`
			}{
				Content: givenInput,
			},
			Output: struct {
				Content string `json:"content"`
			}{
				Content: idealOutputs[i],
			},
		})
	}

	for i, page := range prompt.GetPages() {
		if isPNG(page) {
			var err error
			page, err = visualQuestionAnswering(token, page, prompt)
			if err != nil {
				page = []byte("User attempted to provide an image, but failed")
			}
		}
		instance.Messages = append(instance.Messages, Message{
			Author:  "user",
			Content: string(page),
		})
		instance.Messages = append(instance.Messages, Message{
			Author:  "bot",
			Content: fmt.Sprintf("Thanks. I'll call this REFERENCE %d. ", i+1),
		})
	}
	instance.Messages = append(instance.Messages, Message{
		Author:  "user",
		Content: prompt.GetQuestion(),
	})
	return instance

}

func Completion(ctx context.Context, token string, projectID string, prompt Prompt) (io.Reader, error) {
	messages := MessagesFromPrompt(token, prompt)
	req, err := CreateVertexTextCompletionRequest(token, projectID, messages)
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

type Request struct {
	Instances  []Messages `json:"instances"`
	Parameters Parameters `json:"parameters"`
}

type Parameters struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"topP"`
	TopK            int     `json:"topK"`
}

func CreateVertexTextCompletionRequest(token string, projectID string, messages Messages) (*http.Request, error) {
	URI := fmt.Sprintf("https://us-central1-aiplatform.googleapis.com/v1/projects/%s/locations/us-central1/publishers/google/models/chat-bison:predict", projectID)
	body := Request{
		Instances: []Messages{messages},
		Parameters: Parameters{
			MaxOutputTokens: 1024,
			Temperature:     0.0,
			TopP:            20,
			TopK:            0,
		},
	}
	d, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	data := bytes.NewReader(d)

	req, err := http.NewRequest(http.MethodPost, URI, data)
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
			Candidates []struct {
				Content string `json:"content"`
			} `json:"candidates"`
		} `json:"predictions"`
	}{}

	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	if len(body.Predictions) < 1 {
		return nil, fmt.Errorf("no predictions returned")
	}
	answer := body.Predictions[0].Candidates[0].Content
	answer = strings.Trim(answer, " ")
	return strings.NewReader(answer), nil
}

func isPNG(data []byte) bool {
	return bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47})
}

type VisualQuestionAnsweringRequest struct {
	Prompt string `json:"prompt"`
	Image  struct {
		BytesBase64Encoded string `json:"bytesBase64Encoded"`
	} `json:"image"`
	Parameters struct {
		SampleCount int `json:"sampleCount"`
	} `json:"parameters"`
}

func visualQuestionAnswering(token string, data []byte, prompt Prompt) ([]byte, error) {
	URI := fmt.Sprintf("https://us-central1-aiplatform.googleapis.com/v1/projects/%s/locations/us-central1/publishers/google/models/imagetext:predict", ProjectID)
	question := prompt.GetQuestion()
	purpose := prompt.GetPurpose()
	payload := VisualQuestionAnsweringRequest{
		Parameters: struct {
			SampleCount int `json:"sampleCount"`
		}{
			SampleCount: 1,
		},
	}

	payload.Prompt = fmt.Sprintf("%s\n%s", purpose, question)
	payload.Image.BytesBase64Encoded = base64.StdEncoding.EncodeToString(data)

	d, err := json.Marshal(struct {
		Instances []VisualQuestionAnsweringRequest `json:"instances"`
	}{
		Instances: []VisualQuestionAnsweringRequest{payload},
	},
	)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, URI, bytes.NewReader(d))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body := struct {
		Predictions []string `json:"predictions"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return nil, err
	}

	if len(body.Predictions) < 1 {
		return nil, fmt.Errorf("no predictions returned")
	}
	return []byte(body.Predictions[0]), nil
}
