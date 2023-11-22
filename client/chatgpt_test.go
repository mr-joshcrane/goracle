package client_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
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
		]
}`, client.GPT4V)
	want = strings.ReplaceAll(want, "\t", "")
	want = strings.ReplaceAll(want, "\n", "")
	want = strings.ReplaceAll(want, " ", "")
	want += "\n"
	if got != want {
		t.Error(cmp.Diff(want, got))
	}
}

func TestIsBase64String(t *testing.T) {
	t.Parallel()
	if !client.IsBase64("YQ==") {
		t.Errorf("Expected true, got false")
	}
	if client.IsBase64("dGVzdA") {
		t.Errorf("Expected false, got true")
	}
}

func TestIsJPG(t *testing.T) {
	t.Parallel()

}
