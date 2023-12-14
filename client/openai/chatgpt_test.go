package openai_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/oracle"
	"github.com/mr-joshcrane/oracle/client"
	"github.com/mr-joshcrane/oracle/client/openai"
)

func testPrompt() oracle.Prompt {
	return oracle.Prompt{
		Purpose:       "A test purpose",
		InputHistory:  []string{"GivenInput", "GivenInput2"},
		OutputHistory: []string{"IdealOutput", "IdealOutput2"},
		Question:      "A test question",
		Pages:         [][]byte{[]byte("page1"), []byte("page2")},
	}
}

func testMessages() openai.Messages {
	return openai.MessageFromPrompt(testPrompt())
}

func TestCreateTextCompletionRequestHeaders(t *testing.T) {
	t.Parallel()
	messages := testMessages()
	req, err := openai.CreateTextCompletionRequest("dummy-token-openai", openai.GPT4, messages)
	if err != nil {
		t.Errorf("Error creating request: %s", err)
	}
	auth := req.Header.Get("Authorization")
	if auth != "Bearer dummy-token-openai" {
		t.Errorf("Expected dummy-token-openai, got %s", auth)
	}
	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected application/json, got %s", contentType)
	}
}

func TestCreateTextCompletionRequest(t *testing.T) {
	t.Parallel()
	messages := testMessages()
	req, err := openai.CreateTextCompletionRequest("dummy-token-openai", openai.GPT4, messages)
	if err != nil {
		t.Errorf("Error creating request: %s", err)
	}
	want := fmt.Sprintf(`{"model":"%s","messages":[{"role":"system","content":"A test purpose"},{"role":"user","content":"GivenInput"},{"role":"assistant","content":"IdealOutput"},{"role":"user","content":"GivenInput2"},{"role":"assistant","content":"IdealOutput2"},{"role":"user","content":"A test question"},{"role":"user","content":"Reference 1: page1"},{"role":"user","content":"Reference 2: page2"}]}%v`, openai.GPT4, "\n")
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("Error reading request body: %s", err)
	}
	got := string(data)
	if !cmp.Equal(got, want) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestParseTextCompletionResponse(t *testing.T) {
	t.Parallel()
	req := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"role":"assistant","content":"A woodchuck would chuck as much wood as a woodchuck could chuck if a woodchuck could chuck wood."}}]}`)),
	}
	content, cErr := openai.ParseTextCompletionRequest(req)
	if cErr != nil {
		t.Errorf("Error parsing response: %s", cErr)
	}
	data, err := io.ReadAll(content)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	got := string(data)
	want := "A woodchuck would chuck as much wood as a woodchuck could chuck if a woodchuck could chuck wood."
	if got != want {
		t.Errorf("Expected %s', got %s", want, got)
	}
}

func TestNewChatGPTToken(t *testing.T) {
	t.Parallel()
	c := client.NewChatGPT("dummy-token-openai")
	if c.Token != "dummy-token-openai" {
		t.Errorf("Expected dummy-token-openai, got %s", c.Token)
	}
}

func TestMessageFromPrompt(t *testing.T) {
	t.Parallel()
	prompt := oracle.Prompt{}
	msg := openai.MessageFromPrompt(prompt)
	if len(msg) != 2 {
		t.Errorf("Expected 2 message, got %d ::: %v", len(msg), msg)
	}
	prompt.Purpose = "A test purpose"

	prompt.InputHistory = []string{"GivenInput", "GivenInput2"}
	prompt.OutputHistory = []string{"IdealOutput", "IdealOutput2"}

	prompt.Question = "A test question"

	want := openai.Messages{
		openai.TextMessage{
			Role:    openai.RoleSystem,
			Content: "A test purpose",
		},
		openai.TextMessage{
			Role:    openai.RoleUser,
			Content: "GivenInput",
		},
		openai.TextMessage{
			Role:    openai.RoleAssistant,
			Content: "IdealOutput",
		},
		openai.TextMessage{
			Role:    openai.RoleUser,
			Content: "GivenInput2",
		},
		openai.TextMessage{
			Role:    openai.RoleAssistant,
			Content: "IdealOutput2",
		},
		openai.TextMessage{
			Role:    openai.RoleUser,
			Content: "A test question",
		},
	}
	got := openai.MessageFromPrompt(prompt)
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestMessageFromPromptWithImages(t *testing.T) {
	t.Parallel()
	testImage := image.NewGray(image.Rect(0, 0, 1, 1))
	buf := new(bytes.Buffer)
	err := png.Encode(buf, testImage)
	if err != nil {
		t.Errorf("Error encoding test image: %s", err)
	}
	prompt := oracle.Prompt{
		Pages: [][]byte{buf.Bytes()},
	}
	messages := openai.MessageFromPrompt(prompt)
	if len(messages) != 3 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
	got, ok := messages[2].(openai.VisionMessage)
	if !ok {
		t.Errorf("Expected ImageMessage, got %v", messages[2])
	}
	if got.Role != openai.RoleUser {
		t.Errorf("Expected RoleUser, got %v", got.Role)
	}
	want := "data:image/png;base64,iVBORw0KGgoA"
	if !strings.Contains(got.GetContent(), want) {
		t.Errorf("Expected %s, got %s", want, got.Content)
	}
}

func TestGetCompletionWithInvalidTokenErrors(t *testing.T) {
	t.Parallel()
	c := client.NewChatGPT("dummy-token-openai")
	_, err := c.Completion(context.Background(), oracle.Prompt{})
	want := &openai.ClientError{}
	if !errors.As(err, want) {
		t.Errorf("Expected %v, got %v", want, err)
	}
	if want.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", want.StatusCode)
	}
}

func TestErrorRateLimitThatHitsNoLimitSignalsRetryImmediately(t *testing.T) {
	t.Parallel()
	req := http.Response{}
	req.Header = http.Header{
		"X-Ratelimit-Remaining-Requests": []string{"3500"},
		"X-Ratelimit-Remaining-Tokens":   []string{"90000"},
		"X-Ratelimit-Reset-Requests":     []string{"17ms"},
		"X-Ratelimit-Reset-Tokens":       []string{"28ms"},
	}
	err := openai.ErrorRateLimitExceeded(req)
	want := &openai.RateLimitError{}
	if !errors.As(err, want) {
		t.Errorf("wanted %v, got %v", want, err)
	}
	if want.RetryAfter != 0 {
		t.Errorf("Expected 0, got %d", want.RetryAfter)
	}
}

func TestErrorRateLimitHitsTokenLimitSignalsRetryAfterTokensReset(t *testing.T) {
	t.Parallel()
	req := http.Response{}
	req.Header = http.Header{
		"X-Ratelimit-Remaining-Requests": []string{"3500"},
		"X-Ratelimit-Remaining-Tokens":   []string{"0"},
		"X-Ratelimit-Reset-Requests":     []string{"17ms"},
		"X-Ratelimit-Reset-Tokens":       []string{"28ms"},
	}
	err := openai.ErrorRateLimitExceeded(req)
	want := &openai.RateLimitError{}
	if !errors.As(err, want) {
		t.Errorf("wanted %v, got %v", want, err)
	}
	if want.RetryAfter != time.Duration(28*time.Millisecond) {
		t.Errorf("Expected 28ms, got %d", want.RetryAfter)
	}
}

func TestErrorRateLimitsHitsRetryLimitsSignalsTryAfterRequestsReset(t *testing.T) {
	t.Parallel()
	req := http.Response{}
	req.Header = http.Header{
		"X-Ratelimit-Remaining-Requests": []string{"0"},
		"X-Ratelimit-Remaining-Tokens":   []string{"90000"},
		"X-Ratelimit-Reset-Requests":     []string{"17ms"},
		"X-Ratelimit-Reset-Tokens":       []string{"28ms"},
	}
	err := openai.ErrorRateLimitExceeded(req)
	want := &openai.RateLimitError{}
	if !errors.As(err, want) {
		t.Errorf("wanted %v, got %v", want, err)
	}
	if want.RetryAfter != time.Duration(17*time.Millisecond) {
		t.Errorf("Expected 17ms, got %d", want.RetryAfter)
	}
}

func TestCreateVisionRequest(t *testing.T) {
	t.Parallel()
	messages := testMessages()
	req, err := openai.CreateVisionRequest("dummy-token-openai", messages)
	if err != nil {
		t.Errorf("Error creating request: %s", err)
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("Error reading request body: %s", err)
	}
	got := string(data)
	want := `{"model":"gpt-4-vision-preview","messages":[{"role":"system","content":"A test purpose"},{"role":"user","content":"GivenInput"},{"role":"assistant","content":"IdealOutput"},{"role":"user","content":"GivenInput2"},{"role":"assistant","content":"IdealOutput2"},{"role":"user","content":"A test question"},{"role":"user","content":"Reference 1: page1"},{"role":"user","content":"Reference 2: page2"}],"max_tokens":300}` + "\n"
	if err != nil {
		t.Errorf("Error unmarshalling request body: %s", err)
	}
	if !cmp.Equal(got, want) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestParseVisionResponse(t *testing.T) {
	t.Parallel()
	body := `{"id": "chatcmpl-8UQxuLHv4dLw32PW7BDu3ytBhWTpU", "object": "chat.completion", "created": 1702264186, "model": "gpt-4-1106-vision-preview", "usage": {"prompt_tokens": 790, "completion_tokens": 67, "total_tokens": 10}, "choices": [{"message": {"role": "assistant", "content": "This is a quokka"}, "finish_details": {"type": "stop", "stop": "<|fim_suffix|>"}, "index": 0}]}`
	req := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	content, err := openai.ParseVisionResponse(req)
	if err != nil {
		t.Errorf("Error parsing response: %s", err)
	}
	data, err := io.ReadAll(content)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	got := string(data)
	if got != "This is a quokka" {
		t.Errorf("Expected This is a quokka, got %s", got)
	}
}

func TestParseVisionResponse_ReturnsClientErrorIfNotOK(t *testing.T) {
	t.Parallel()
	req := &http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     "400 Bad Request",
		Body:       io.NopCloser(strings.NewReader(`{}`)),
	}
	_, err := openai.ParseVisionResponse(req)
	if errors.Is(err, &openai.ClientError{}) {
		t.Errorf("Expected ClientError, got %v", err)
	}
}

func TestURLToURI(t *testing.T) {
	t.Parallel()
	pngURL, _ := url.Parse("https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png")
	got, err := openai.URLToURI(*pngURL)
	if err != nil {
		t.Errorf("Error converting url to data uri: %s", err)
	}
	want := "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png"
	if want != got {
		t.Fatalf("Expected %s, got %s", want, got)
	}
}

func TestURLToURI_IfNotValidType(t *testing.T) {
	t.Parallel()
	nonPNGURL, _ := url.Parse("https:www.google.com")
	_, err := openai.URLToURI(*nonPNGURL)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestCreateImageRequest(t *testing.T) {
	t.Parallel()
	pngURL, _ := url.Parse("https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png")
	req, err := openai.CreateImageRequest("dummy-token-openai", pngURL.String())
	if err != nil {
		t.Errorf("Error creating request: %s", err)
	}
	if req.URL.Host != "api.openai.com" {
		t.Errorf("Expected api.openai.com, got %s", req.URL.Host)
	}
	if req.URL.Path != "/v1/images/generations" {
		t.Errorf("Expected /v1/images, got %s", req.URL.Path)
	}
	if req.Method != http.MethodPost {
		t.Errorf("Expected POST, got %s", req.Method)
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected application/json, got %s", req.Header.Get("Content-Type"))
	}
	if req.Header.Get("Authorization") != "Bearer dummy-token-openai" {
		t.Errorf("Expected dummy-token-openai, got %s", req.Header.Get("Authorization"))
	}
}

func TestParseImageResponse(t *testing.T) {
	t.Parallel()
	body := `{ "id": "img-1-1234", "object": "image", "created": 1589478378, "data": [ { "url": "http://somelinktosomeurl" } ]}`
	req := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	got, err := openai.ParseCreateImageResponse(req)
	if err != nil {
		t.Errorf("Error parsing response: %s", err)
	}
	want := "http://somelinktosomeurl"
	if got != want {
		t.Errorf("Expected %s, got %s", want, got)
	}
}

func TestParseImageResponse_ReturnsClientErrorIfNotOK(t *testing.T) {
	t.Parallel()
	req := &http.Response{
		StatusCode: http.StatusForbidden,
		Status:     "403 Forbidden",
		Body:       io.NopCloser(strings.NewReader(`{}`)),
	}
	_, err := openai.ParseCreateImageResponse(req)
	if errors.Is(err, &openai.ClientError{}) {
		t.Errorf("Expected ClientError, got %v", err)
	}
}

func TestParseLinkToImage(t *testing.T) {
	t.Parallel()
	req := "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png"
	got, err := openai.ParseLinkToImage(req)
	if err != nil {
		t.Fatalf("Error parsing link to image: %s", err)
	}
	_, err = png.Decode(bytes.NewBuffer(got))
	if err != nil {
		t.Fatalf("Error decoding image: %s", err)
	}
}

func TestTextToSpeechRequest(t *testing.T) {
	t.Parallel()
	got, err := openai.CreateTextToSpeechRequest("testToken", "someText")
	if err != nil {
		t.Errorf("Error creating response: %s", err)
	}
	buf := new(bytes.Buffer)
	want := httptest.NewRequest("POST", "https://api.openai.com/v1/audio/speech", buf)
	if got.Method != want.Method {
		t.Errorf("Expected %s, got %s", want.Method, got.Method)
	}
	if got.URL.Host != want.URL.Host {
		t.Errorf("Expected %s, got %s", want.URL.Host, got.URL.Host)
	}
	if got.URL.Path != want.URL.Path {
		t.Errorf("Expected %s, got %s", want.URL.Path, got.URL.Path)
	}
	wantBody := `{"model":"tts-1","input":"someText","voice":"echo"}`
	data, err := io.ReadAll(got.Body)
	if err != nil {
		t.Errorf("Error reading request body: %s", err)
	}
	gotBody := string(data)
	if wantBody != gotBody {
		cmp.Diff(wantBody, gotBody)
	}
}

func TestSpeechToTextRequest(t *testing.T) {
	t.Parallel()
	path := t.TempDir() + "/test.wav"
	err := os.WriteFile(path, []byte("test"), 0644)
	if err != nil {
		t.Errorf("Error creating test file: %s", err)
	}
	got, err := openai.CreateSpeechToTextRequest("", []byte{})
	if err != nil {
		t.Errorf("Error creating response: %s", err)
	}
	contentType := "multipart/form-data; boundary="
	if !strings.Contains(got.Header.Get("Content-Type"), contentType) {
		t.Errorf("Expected content type to contain %v, got %s", contentType, got.Header.Get("Content-Type"))
	}
	if got.URL.Host != "api.openai.com" {
		t.Errorf("Expected api.openai.com, got %s", got.URL.Host)
	}
	if got.URL.Path != "/v1/audio/transcriptions" {
		t.Errorf("Expected /v1/audio/speech, got %s", got.URL.Path)
	}
	body := new(bytes.Buffer)
	_, err = body.ReadFrom(got.Body)
	if err != nil {
		t.Errorf("Error reading request body: %s", err)
	}
	if body.Len() == 0 {
		t.Errorf("Expected non-empty body, got empty body")
	}
}
