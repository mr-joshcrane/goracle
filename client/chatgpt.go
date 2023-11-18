package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

const (
	GPT35Turbo = "gpt-3.5-turbo-1106"
	GPT4       = "gpt-4-1106-preview"
	GPT4V      = "gpt-4-vision-preview"
	DALLE3     = "dall-e-3"
	TTS        = "tts-1"
	TTSHQ      = "tts-1-hq"
	WHISPER    = "whisper-1"
)

type ChatGPT struct {
	Token string
	Model string
}

type Dummy struct {
	FixedResponse  string
	FixedHTTPError int
}

func (d *Dummy) Completion(ctx context.Context, prompt Prompt) (string, error) {
	if d.FixedHTTPError == 200 {
		return d.FixedResponse, nil
	}
	response := http.Response{
		Status:     "client error",
		StatusCode: d.FixedHTTPError,
	}
	return "", NewClientError(&response)
}

func NewChatGPT(token string) *ChatGPT {
	return &ChatGPT{
		Token: token,
		Model: GPT4,
	}
}

func NewDummyClient(fixedResponse string, errorCode int) *Dummy {
	return &Dummy{
		FixedResponse:  fixedResponse,
		FixedHTTPError: errorCode,
	}
}

type Prompt interface {
	GetPurpose() string
	GetExamples() ([]string, []string)
	GetQuestion() string
	GetImages() []string
}

func MessageFromPrompt(prompt Prompt) []Message {
	messages := []Message{}
	messages = append(messages, Message{
		Role:    RoleSystem,
		Content: prompt.GetPurpose(),
	})
	givenInputs, idealOutputs := prompt.GetExamples()
	for i, givenInput := range givenInputs {
		messages = append(messages, Message{
			Role:    RoleUser,
			Content: givenInput,
		})
		messages = append(messages, Message{
			Role:    RoleAssistant,
			Content: idealOutputs[i],
		})
	}
	messages = append(messages, Message{
		Role:    RoleUser,
		Content: prompt.GetQuestion(),
	})
	return messages
}

func (c *ChatGPT) Completion(ctx context.Context, prompt Prompt) (string, error) {
	if len(prompt.GetImages()) > 0 {
		return c.visionCompletion(ctx, prompt.GetQuestion(), prompt.GetImages()...)
	}
	return c.standardCompletion(ctx, prompt)
}

func (c *ChatGPT) visionCompletion(ctx context.Context, prompt string, images ...string) (string, error) {
	for _, image := range images {
		resp, err := http.DefaultClient.Get(image)
		if err != nil {
			return "", ClientError{StatusCode: http.StatusBadRequest, Status: err.Error()}
		}
		if resp.StatusCode != http.StatusOK {
			return "", ClientError{StatusCode: resp.StatusCode, Status: resp.Status + " " + image}
		}
	}
	message := CreateVisionMessage(prompt, images...)
	req, err := CreateVisionRequest(c.Token, message)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", NewClientError(resp)
	}
	defer resp.Body.Close()
	var completion VisionCompletionResponse
	err = json.NewDecoder(resp.Body).Decode(&completion)
	if err != nil {
		return "", err
	}
	if len(completion.Choices) < 1 {
		return "", fmt.Errorf("no choices returned")
	}
	return completion.Choices[0].Message.Content, nil
}

func (c *ChatGPT) standardCompletion(ctx context.Context, prompt Prompt) (string, error) {
	messages := MessageFromPrompt(prompt)
	req, err := CreateChatGPTRequest(c.Token, c.Model, messages)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", NewClientError(resp)
	}
	defer resp.Body.Close()
	return ParseResponse(resp.Body)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func CreateChatGPTRequest(token string, model string, messages []Message) (*http.Request, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func ParseResponse(r io.Reader) (string, error) {
	resp := ChatCompletionResponse{}
	err := json.NewDecoder(r).Decode(&resp)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) < 1 {
		return "", fmt.Errorf("no choices returned")
	}
	return resp.Choices[0].Message.Content, nil
}

type ModelResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Id      string `json:"id"`
		Object  string `json:"object"`
		Created int    `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

func GetModels(token string) ([]string, error) {
	var models ModelResponse
	req, err := http.NewRequest(http.MethodGet, "https://api.openai.com/v1/engines", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, NewClientError(resp)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &models)

	var results []string
	for _, model := range models.Data {
		results = append(results, model.Id)
	}
	return results, nil
}

type ImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size"`
}

type ImageResponse struct {
	Created int `json:"created"`
	Data    []struct {
		Url string `json:"url"`
	} `json:"data"`
}

func CreateImageRequest(token string, prompt string) (*http.Request, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(ImageRequest{
		Model:  DALLE3,
		Prompt: prompt,
		N:      1,
		Size:   "1024x1024",
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/images/generations", buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func GenerateImage(token, prompt string) (string, error) {
	req, err := CreateImageRequest(token, prompt)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
		return "", fmt.Errorf("bad status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	fmt.Println(string(data))
	var image ImageResponse
	err = json.Unmarshal(data, &image)
	if err != nil {
		return "", err
	}
	if len(image.Data) < 1 {
		return "", fmt.Errorf("no images returned")
	}
	return image.Data[0].Url, nil

}

func CreateVisionMessage(prompt string, images ...string) VisionMessage {
	messages := VisionMessage{
		Role: RoleUser,
		Content: []map[string]string{
			{
				"type": "text",
				"text": prompt,
			},
		},
	}
	for _, imageSrc := range images {
		messages.Content = append(messages.Content, map[string]string{
			"type":      "image_url",
			"image_url": imageSrc,
		})
	}
	return messages
}

type VisionImageURL struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

type VisionMessage struct {
	Role    string              `json:"role"`
	Content []map[string]string `json:"content"`
}

type VisionRequest struct {
	Model    string          `json:"model"`
	Messages []VisionMessage `json:"messages"`
}

func CreateVisionRequest(token string, message VisionMessage) (*http.Request, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(VisionRequest{
		Model:    GPT4V,
		Messages: []VisionMessage{message},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *ChatGPT) VisionCompletion(ctx context.Context, prompt string, images ...string) (string, error) {
	message := CreateVisionMessage(prompt, images...)
	req, err := CreateVisionRequest(c.Token, message)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", NewClientError(resp)
	}
	defer resp.Body.Close()
	var completion VisionCompletionResponse
	err = json.NewDecoder(resp.Body).Decode(&completion)
	if err != nil {
		return "", err
	}
	if len(completion.Choices) < 1 {
		return "", fmt.Errorf("no choices returned")
	}
	return completion.Choices[0].Message.Content, nil
}

type VisionCompletionResponse struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishDetails struct {
			Type string `json:"type"`
		} `json:"finish_details"`
		Index int `json:"index"`
	} `json:"choices"`
}
