package client_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
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
)

func TestGetChatCompletionsRequestHeaders(t *testing.T) {
	t.Parallel()
	req, err := client.CreateChatGPTRequest("dummy-token-openai", client.GPT35Turbo, []client.Message{})
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

func TestGetChatCompletionsRequestBody(t *testing.T) {
	t.Parallel()
	messages := []client.Message{
		{
			Role:    "user",
			Content: "Say this is a test!",
		},
	}

	req, err := client.CreateChatGPTRequest("dummy-token-openai", client.GPT35Turbo, messages)
	if err != nil {
		t.Errorf("Error creating request: %s", err)
	}
	want := fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"Say this is a test!"}]}%s`, client.GPT35Turbo, "\n")
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("Error reading request body: %s", err)
	}
	got := string(data)
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestParseResponse(t *testing.T) {
	t.Parallel()
	f, err := os.Open("testdata/response.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	content, cErr := client.ParseResponse(f)
	if cErr != nil {
		t.Errorf("Error parsing response: %s", err)
	}
	want := "A woodchuck would chuck as much wood as a woodchuck could chuck if a woodchuck could chuck wood."
	if content != want {
		t.Errorf("Expected %s', got %s", want, content)
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
	msg := client.MessageFromPrompt(prompt)
	if len(msg) != 2 {
		t.Errorf("Expected 2 message, got %d ::: %v", len(msg), msg)
	}
	prompt.Purpose = "A test purpose"

	prompt.ExampleInputs = []string{"GivenInput", "GivenInput2"}
	prompt.IdealOutputs = []string{"IdealOutput", "IdealOutput2"}

	prompt.Question = "A test question"

	want := []client.Message{
		{
			Role:    "system",
			Content: "A test purpose",
		},
		{
			Role:    "user",
			Content: "GivenInput",
		},
		{
			Role:    "assistant",
			Content: "IdealOutput",
		},
		{
			Role:    "user",
			Content: "GivenInput2",
		},
		{
			Role:    "assistant",
			Content: "IdealOutput2",
		},
		{
			Role:    "user",
			Content: "A test question",
		},
	}
	got := client.MessageFromPrompt(prompt)
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestGetCompletionWithInvalidTokenErrors(t *testing.T) {
	t.Parallel()
	c := client.NewChatGPT("dummy-token-openai")
	_, err := c.Completion(context.Background(), oracle.Prompt{})
	want := &client.ClientError{}
	if !errors.As(err, want) {
		t.Errorf("Expected %v, got %v", want, err)
	}
	if want.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", want.StatusCode)
	}
}

func TestCompletionWithRateLimitErrorReturnsARetryAfterValue(t *testing.T) {
	t.Parallel()
	c := client.NewDummyClient("response", 429)
	_, err := c.Completion(context.Background(), oracle.Prompt{})
	want := &client.RateLimitError{}
	if !errors.As(err, want) {
		t.Errorf("wanted %v, got %v", want, err)
	}
	if want.RetryAfter != 0 {
		t.Errorf("Expected 0, got %d", want.RetryAfter)
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
	err := client.ErrorRateLimitExceeded(req)
	want := &client.RateLimitError{}
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
	err := client.ErrorRateLimitExceeded(req)
	want := &client.RateLimitError{}
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
	err := client.ErrorRateLimitExceeded(req)
	want := &client.RateLimitError{}
	if !errors.As(err, want) {
		t.Errorf("wanted %v, got %v", want, err)
	}
	if want.RetryAfter != time.Duration(17*time.Millisecond) {
		t.Errorf("Expected 17ms, got %d", want.RetryAfter)
	}
}

func TestCreateVisionMessages(t *testing.T) {
	t.Parallel()
	msg := client.CreateVisionMessage("somePrompt", "someUrl")
	want := client.VisionMessage{

		Role: "user",
		Content: []map[string]string{
			{
				"type": "text",
				"text": "somePrompt",
			},
			{
				"type":      "image_url",
				"image_url": "someUrl",
			},
		},
	}
	if !cmp.Equal(want, msg) {
		t.Error(cmp.Diff(want, msg))
	}
}

func TestCreateVisionRequest(t *testing.T) {
	t.Parallel()
	msg := client.CreateVisionMessage("somePrompt", "someUrl")
	req, err := client.CreateVisionRequest("dummy-token-openai", msg)
	if err != nil {
		t.Errorf("Error creating request: %s", err)
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("Error reading request body: %s", err)
	}
	defer req.Body.Close()
	got := string(data)
	want := fmt.Sprintf(`
		{"model":"%s","messages": [
			{
				"role":"user",
				"content": [
					{"text":"somePrompt", "type":"text"},
		    	{"image_url":"someUrl", "type":"image_url"}
				]
			}
		],
		"max_tokens": 300
}`, client.GPT4V)
	want = strings.ReplaceAll(want, "\t", "")
	want = strings.ReplaceAll(want, "\n", "")
	want = strings.ReplaceAll(want, " ", "")
	want += "\n"
	if got != want {
		t.Error(cmp.Diff(want, got))
	}
}

func TestImageToDataURI(t *testing.T) {
	t.Parallel()
	testImage := image.NewRGBA(image.Rect(0, 0, 1, 1))
	got, err := client.ImageToDataURI(testImage)
	if err != nil {
		t.Errorf("Error converting image to data uri: %s", err)
	}
	want := "data:image/png;base64,"
	if !strings.HasPrefix(got, want) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestURLToURI(t *testing.T) {
	t.Parallel()
	pngURL, _ := url.Parse("https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png")
	got, err := client.URLToURI(*pngURL)
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
	nonPNGURL, _ := url.Parse("https://www.google.com")
	_, err := client.URLToURI(*nonPNGURL)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestTextToSpeechRequest(t *testing.T) {
	t.Parallel()
	got, err := client.CreateTextToSpeechRequest("testToken", "someText")
	if err != nil {
		t.Errorf("Error creating response: %s", err)
	}
	buf := bytes.NewReader([]byte(`{"model":"tts-1","input":"someText","voice":"alloy"}`))
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
	wantBody := []byte{}
	gotBody := []byte{}
	_, _ = want.Body.Read(wantBody)
	_, _ = got.Body.Read(gotBody)

	if !cmp.Equal(wantBody, gotBody) {
		t.Error(cmp.Diff(wantBody, gotBody))
	}
}

func TestSpeechToTextRequest(t *testing.T) {
	t.Parallel()
	path := t.TempDir() + "/test.wav"
	err := os.WriteFile(path, []byte("test"), 0644)
	if err != nil {
		t.Errorf("Error creating test file: %s", err)
	}
	got, err := client.CreateSpeechToTextRequest("", []byte{})
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
	body.ReadFrom(got.Body)
	if body.Len() == 0 {
		t.Errorf("Expected non-empty body, got empty body")
	}
}
