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

//from openai import OpenAI
//client = OpenAI()
//
//client.images.generate(
//  model="dall-e-3",
//  prompt="A cute baby sea otter",
//  n=1,
//  size="1024x1024"
//)

//{
//  "created": 1589478378,
//  "data": [
//    {
//      "url": "https://..."
//    },
//    {
//      "url": "https://..."
//    }
//  ]
//}

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
