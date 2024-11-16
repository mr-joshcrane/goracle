package google

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

type Role string

var (
	User Role = "USER"
	Bot  Role = "ASSISTANT"
)

type Prompt interface {
	GetPurpose() string
	GetHistory() ([]string, []string)
	GetQuestion() string
	GetReferences() [][]byte
}

type ChatMessage struct {
	Role  Role        `json:"role"`
	Parts MessagePart `json:"parts"`
}

type MessagePart struct {
	Text string `json:"text,omitempty"`
}

// More serious authentication methods left as an exercise to the reader
// https://cloud.google.com/docs/authentication
func Authenticate() (clientID string, token string, err error) {
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	tokenOut, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error getting GCP token: %w", err)
	}
	token = strings.TrimSpace(string(tokenOut))

	cmd = exec.Command("gcloud", "config", "get-value", "project")
	projectOut, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error getting GCP project: %w", err)
	}
	clientID = strings.TrimSpace(string(projectOut))
	return clientID, token, nil
}

func MessagesFromPrompt(prompt Prompt) []ChatMessage {
	messages := []ChatMessage{
		{
			Role:  User,
			Parts: MessagePart{Text: "SYSTEM: USER PROVIDED PURPOSE: " + prompt.GetPurpose()},
		},
		{
			Role:  Bot,
			Parts: MessagePart{Text: "Understood!"},
		},
	}
	idealInputs, idealOutputs := prompt.GetHistory()
	for i, idealInput := range idealInputs {
		messages = append(messages, ChatMessage{
			Role:  User,
			Parts: MessagePart{Text: idealInput},
		})
		messages = append(messages, ChatMessage{
			Role:  Bot,
			Parts: MessagePart{Text: idealOutputs[i]},
		})
	}
	for i, ref := range prompt.GetReferences() {
		if isPNG(ref) {
			messages = append(messages, ChatMessage{
				Role:  User,
				Parts: MessagePart{Text: string(ref)},
			})
			continue
		}
		messages = append(messages, ChatMessage{
			Role:  User,
			Parts: MessagePart{Text: fmt.Sprintf("SYSTEM: USER PROVIDED FILE %d: %s", i+1, string(ref))},
		})
		messages = append(messages, ChatMessage{
			Role:  Bot,
			Parts: MessagePart{Text: "Understood. I will refer to this text in my future answers!"},
		})
	}
	messages = append(messages, ChatMessage{
		Role:  User,
		Parts: MessagePart{Text: prompt.GetQuestion()},
	})
	return messages
}

func textCompletion(ctx context.Context, token string, projectID string, messages []ChatMessage) (io.Reader, error) {
	req, err := CreateVertexTextCompletionRequest(token, projectID, messages)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	answer, err := ParseVertexTextCompletionResponse(*resp)
	if err != nil {
		return nil, err
	}
	return answer, err
}

func visionCompletion(ctx context.Context, token string, projectID string, messages []ChatMessage) (io.Reader, error) {
	URI := fmt.Sprintf("https://us-central1-aiplatform.googleapis.com/v1/projects/%s/locations/us-central1/publishers/google/models/gemini-pro-vision:streamGenerateContent", projectID)
	payload := VisualCompletionRequest{
		GenerationConfig: GenerationConfig{
			MaxOutputTokens: 1024,
			Temperature:     0.0,
			TopP:            0.8,
			TopK:            40,
		},
		Contents: []VisualRequestContents{
			{
				Role:  User,
				Parts: []any{},
			},
		},
	}
	payload.Contents[0].Parts = append(payload.Contents[0].Parts, struct {
		Text string `json:"text"`
	}{
		Text: "",
	})

	var text string
	for _, message := range messages {
		if isPNG([]byte(message.Parts.Text)) {
			payload.Contents[0].Parts = append(payload.Contents[0].Parts, struct {
				InlineData VisualInlineData `json:"inlineData,omitempty"`
			}{
				InlineData: VisualInlineData{
					MimeType: "image/png",
					Data:     base64.StdEncoding.EncodeToString([]byte(message.Parts.Text)),
				},
			})
		} else {
			text += "\n" + message.Parts.Text
		}
	}
	payload.Contents[0].Parts[0] = struct {
		Text string `json:"text"`
	}{Text: text}

	d, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, URI, bytes.NewReader(d))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d; %s", resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()
	body := []struct {
		Candidates []struct {
			Content struct {
				Role  string `json:"role"`
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return nil, err
	}

	if len(body) < 1 {
		return nil, fmt.Errorf("no predictions returned")
	}
	var answer string
	for _, candidate := range body {
		for _, content := range candidate.Candidates {
			answer += content.Content.Parts[0].Text
		}
	}
	answer = strings.Trim(answer, " ")
	return strings.NewReader(answer), nil
}

func Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	project, token, err := Authenticate()
	if err != nil {
		return nil, err
	}
	strategy := textCompletion
	messages := MessagesFromPrompt(prompt)
	for _, message := range messages {
		if isPNG([]byte(message.Parts.Text)) {
			strategy = visionCompletion
			break
		}
	}
	answer, err := strategy(ctx, token, project, messages)
	if err != nil {
		return nil, err
	}
	return answer, nil
}

type TextCompletionRequest struct {
	Contents         []ChatMessage    `json:"contents"`
	GenerationConfig GenerationConfig `json:"generation_config"`
}

type GenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"topP"`
	TopK            int     `json:"topK"`
}

func CreateVertexTextCompletionRequest(token string, projectID string, messages []ChatMessage) (*http.Request, error) {
	URI := fmt.Sprintf("https://us-central1-aiplatform.googleapis.com/v1/projects/%s/locations/us-central1/publishers/google/models/gemini-1.5-pro-002:streamGenerateContent", projectID)
	body := TextCompletionRequest{
		Contents: messages,
		GenerationConfig: GenerationConfig{
			MaxOutputTokens: 8192,
			Temperature:     0.9,
			TopP:            0.8,
			TopK:            40,
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

func ParseVertexTextCompletionResponse(resp http.Response) (io.Reader, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d, %s", resp.StatusCode, resp.Status)
	}

	defer resp.Body.Close()
	body := []struct {
		Candidates []struct {
			Content struct {
				Role  string `json:"role"`
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		}
	}{}
	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	if len(body) < 1 {
		return nil, fmt.Errorf("no predictions returned")
	}

	var answer string
	for _, candidate := range body {
		for _, content := range candidate.Candidates {
			if len(content.Content.Parts) < 1 {
				continue
			}
			answer += content.Content.Parts[0].Text
		}
	}
	answer = strings.Trim(answer, " ")
	return strings.NewReader(answer), nil
}

func isPNG(data []byte) bool {
	return bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47})
}

type VisualCompletionRequest struct {
	Contents         []VisualRequestContents `json:"contents"`
	GenerationConfig GenerationConfig        `json:"generation_config"`
}

type VisualRequestContents struct {
	Role  Role  `json:"role"`
	Parts []any `json:"parts"`
}

type VisualInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}
