package client_test

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/oracle"
	"github.com/mr-joshcrane/oracle/client"
)

func TestGetChatCompletionsRequestHeaders(t *testing.T) {
	t.Parallel()
	req, err := client.CreateChatGPTRequest("dummy-token-openai", []client.Message{})
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

	req, err := client.CreateChatGPTRequest("dummy-token-openai", messages)
	if err != nil {
		t.Errorf("Error creating request: %s", err)
	}
	want := `{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Say this is a test!"}]}` + "\n"
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
	_, err := c.Completion(oracle.Prompt{})
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
	_, err := c.Completion(oracle.Prompt{})
	want := &client.RateLimitError{}
	if !errors.As(err, want) {
		t.Errorf("wanted %v, got %v", want, err)
	}
	if want.RetryAfter != 0 {
		t.Errorf("Expected 0, got %d", want.RetryAfter)
	}
}
