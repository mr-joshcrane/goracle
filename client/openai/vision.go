package openai

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
	GPT4V  = "gpt-4o"
)

// Image Generation Capability
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

func DoImageRequest(ctx context.Context, token string, prompt string) ([]byte, error) {
	req, err := CreateImageRequest(token, prompt)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	link, err := ParseCreateImageResponse(resp)
	if err != nil {
		return nil, err
	}
	return ParseLinkToImage(link)
}

func CreateImageRequest(token string, prompt string) (*http.Request, error) {
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
	req = addDefaultHeaders(token, req)
	return req, nil
}

func ParseCreateImageResponse(resp *http.Response) (string, error) {
	var imageResponse ImageResponse
	if resp.StatusCode != http.StatusOK {
		return "", NewClientError(resp)
	}
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(&imageResponse)
	if err != nil {
		return "", err
	}
	if len(imageResponse.Data) < 1 {
		return "", fmt.Errorf("no images returned")
	}
	imageUrl := imageResponse.Data[0].Url // Only one image supported by DALEE3 :\
	return imageUrl, nil
}

func ParseLinkToImage(link string) ([]byte, error) {
	resp, err := http.DefaultClient.Get(link)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// // Vision Capability

type VisionImageURL struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

type VisionMessage struct {
	Role    string           `json:"role"`
	Content []VisionImageURL `json:"content"`
}

func (m VisionMessage) GetFormat() string {
	return "Vision"
}

func (m VisionMessage) GetContent() string {
	if len(m.Content) != 1 {
		return ""
	}
	return m.Content[0].ImageURL.URL
}

type VisionRequest struct {
	Model     string   `json:"model"`
	Messages  Messages `json:"messages"`
	MaxTokens int      `json:"max_tokens"`
}
type VisionCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
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

func ParseVisionResponse(resp *http.Response) (io.Reader, error) {
	var completion VisionCompletionResponse
	if resp.StatusCode != http.StatusOK {
		return nil, NewClientError(resp)
	}
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(&completion)
	if err != nil {
		return nil, err
	}
	if len(completion.Choices) < 1 {
		return nil, fmt.Errorf("no choices returned")
	}
	answer := strings.NewReader(completion.Choices[0].Message.Content)
	return answer, nil
}

func visionCompletion(ctx context.Context, token string, message Messages) (io.Reader, error) {
	req, err := CreateVisionRequest(token, message)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	return ParseVisionResponse(resp)
}

func isPNG(a []byte) bool {
	return len(a) > 8 && bytes.Equal(a[:8], []byte("\x89PNG\x0d\x0a\x1a\x0a"))
}

func ConvertPNGToDataURI(data []byte) string {
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
