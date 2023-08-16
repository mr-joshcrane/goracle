package oracle_test

import (
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/oracle"
)

// func TestAsk(t *testing.T) {
// 	t.Parallel()
// 	oracle := oracle.NewOracle()
// 	oracle.SetPurpose("You always answer questions with the number 42.")
// 	question := "What is the meaning of life?"
// 	answer, err := oracle.Ask(question)
// 	if err != nil {
// 		t.Errorf("Error asking question: %s", err)
// 	}
// 	if answer != "42" {
// 		t.Errorf("Expected 42, got %s", answer)
// 	}
// }

// func TestGiveExample(t *testing.T) {
// 	t.Parallel()
// 	oracle := oracle.NewOracle()
// 	oracle.SetPurpose("To answer if a number is odd or even in a specific format")
// 	oracle.GiveExamplePrompt("2", "+++even+++")
// 	oracle.GiveExamplePrompt("3", "---odd---")
// 	oracle.GiveExamplePrompt("4", "+++even+++")
// 	oracle.GiveExamplePrompt("5", "---odd---")

// 	answer, err := oracle.Ask("6")
// 	if err != nil {
// 		t.Errorf("Error asking question: %s", err)
// 	}

// 	if answer != "+++even+++" {
// 		t.Errorf("Expected +++even+++, got %s", answer)
// 	}
// }

func TestGetChatCompletionsRequestHeaders(t *testing.T) {
	t.Parallel()
	req := oracle.CreateChatGPTRequest("dummy-token-openai", []oracle.Message{})
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
	messages := []oracle.Message{
		{
			Role:    "user",
			Content: "Say this is a test!",
		},
	}

	req := oracle.CreateChatGPTRequest("dummy-token-openai", messages)
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
	content, err := oracle.ParseResponse(f)
	if err != nil {
		t.Errorf("Error parsing response: %s", err)
	}
	want := "A woodchuck would chuck as much wood as a woodchuck could chuck if a woodchuck could chuck wood."
	if content != want {
		t.Errorf("Expected %s', got %s", want, content)
	}
}
