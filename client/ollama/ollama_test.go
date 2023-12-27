package ollama_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/goracle"
	"github.com/mr-joshcrane/goracle/client/ollama"
)

func testPrompt() goracle.Prompt {
	return goracle.Prompt{
		Purpose:       "A test purpose",
		InputHistory:  []string{"GivenInput", "GivenInput2"},
		OutputHistory: []string{"IdealOutput", "IdealOutput2"},
		Question:      "A test question",
		References:    [][]byte{[]byte("page1"), []byte("page2")},
	}
}

func TestNewChatCompletionRequest_TransformsPromptIntoChatCompletionBody(t *testing.T) {
	t.Parallel()
	prompt := testPrompt()
	messages := ollama.Messages{
		{Role: "system", Content: "A test purpose"},
		{Role: "user", Content: "GivenInput"},
		{Role: "assistant", Content: "IdealOutput"},
		{Role: "user", Content: "GivenInput2"},
		{Role: "assistant", Content: "IdealOutput2"},
		{Role: "user", Content: "Reference 1: page1"},
		{Role: "user", Content: "Reference 2: page2"},
		{Role: "assistant", Content: "A test question"},
	}
	got := ollama.NewChatCompletionRequest("phi", prompt)
	want := ollama.ChatCompletion{
		Model:    "phi",
		Messages: messages,
		Stream:   false,
	}
	if !cmp.Equal(got, want) {
		t.Fatalf(cmp.Diff(got, want))
	}
}

func TestParseChatCompletionResponse_ReturnsErrorIfResponseIsNotOK(t *testing.T) {
	t.Parallel()
	resp := &http.Response{StatusCode: http.StatusBadRequest}
	_, err := ollama.ParseChatCompletionResponse(resp)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseChatCompletionResponse_ReturnsErrorIfNoBody(t *testing.T) {
	t.Parallel()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}
	_, err := ollama.ParseChatCompletionResponse(resp)
	if err == nil {
		t.Fatal("error expected")
	}
}

func TestParseChatCompletionResponse_ReturnsDataFromResponse(t *testing.T) {
	t.Parallel()
	body := io.NopCloser(strings.NewReader(`{"message": {"role": "assistant", "content": "A test response"}}`))

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       body,
	}
	got, err := ollama.ParseChatCompletionResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	want := "A test response"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestOllamaClient_ChatCompletion_ReturnsErrorIfRequestFails(t *testing.T) {
	t.Parallel()
	_, err := ollama.DoChatCompletion("model", "http://localhost:9999", testPrompt())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOllamaClient_ChatCompletion_ReturnsDataFromResponse(t *testing.T) {
	t.Parallel()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": {"role": "assistant", "content": "A test response"}}`))
	}))
	defer mockServer.Close()

	got, err := ollama.DoChatCompletion("model", mockServer.URL, testPrompt())
	if err != nil {
		t.Fatal(err)
	}
	want := "A test response"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
