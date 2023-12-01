package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	DALLE3 = "dall-e-3"
	GPT4V  = "gpt-4-vision-preview"
)

func PNGToDataURI(data []byte) string {
	base64Str := base64.StdEncoding.EncodeToString(data)
	dataURI := "data:image/png;base64," + base64Str
	return dataURI
}

func URLToURI(url url.URL) (string, error) {
	visionMimeType := []string{
		"image/png",
		"image/jpeg",
		"image/jpg",
	}
	resp, err := http.DefaultClient.Head(url.String())
	if err != nil {
		return "", err
	}
	for _, mimeType := range visionMimeType {
		if resp.Header.Get("Content-Type") == mimeType {
			return url.String(), nil
		}
	}
	return "", fmt.Errorf("unsupported visual mime type: %s", resp.Header.Get("Content-Type"))
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

func (c *ChatGPT) CreateImageRequest(prompt string, n int) (*http.Request, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(ImageRequest{
		Model:  DALLE3,
		Prompt: prompt,
		N:      1, // Only one is supported by DALLE3 :\
		Size:   "1024x1024",
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/images/generations", buf)
	if err != nil {
		return nil, err
	}
	req = addDefaultHeaders(c.Token, req)
	return req, nil
}

func (c *ChatGPT) GenerateImage(prompt string, n int) (ImageResponse, error) {
	var image ImageResponse
	req, err := c.CreateImageRequest(prompt, n)
	if err != nil {
		return image, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return image, err
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
		return image, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return image, err
	}
	err = json.Unmarshal(data, &image)
	if err != nil {
		return image, err
	}
	if len(image.Data) < 1 {
		return image, fmt.Errorf("no images returned")
	}
	return image, nil
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

func (m VisionMessage) GetFormat() string {
	return MessageImage
}

type VisionRequest struct {
	Model     string   `json:"model"`
	Messages  Messages `json:"messages"`
	MaxTokens int      `json:"max_tokens"`
}

func CreateVisionRequest(token string, messages Messages) (*http.Request, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(VisionRequest{
		Model:     GPT4V,
		Messages:  messages,
		MaxTokens: 300,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", buf)
	if err != nil {
		return nil, err
	}
	req = addDefaultHeaders(token, req)
	return req, nil
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

func (c *ChatGPT) visionCompletion(ctx context.Context, message Messages) (io.Reader, error) {
	req, err := CreateVisionRequest(c.Token, message)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, NewClientError(resp)
	}
	defer resp.Body.Close()
	var completion VisionCompletionResponse
	err = json.NewDecoder(resp.Body).Decode(&completion)
	if err != nil {
		return nil, err
	}
	if len(completion.Choices) < 1 {
		return nil, fmt.Errorf("no choices returned")
	}
	answer := strings.NewReader(completion.Choices[0].Message.Content)
	return answer, nil
}

func isPNG(a []byte) bool {
	return len(a) > 8 && bytes.Equal(a[:8], []byte("\x89PNG\x0d\x0a\x1a\x0a"))
}
