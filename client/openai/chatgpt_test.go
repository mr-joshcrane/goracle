package openai_test

//
// import (
// 	"bytes"
// 	"context"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"net/http/httptest"
// 	"net/url"
// 	"os"
// 	"strings"
// 	"testing"
// 	"time"
//
// 	"github.com/google/go-cmp/cmp"
// 	"github.com/mr-joshcrane/oracle"
// )
//
// func testPrompt() oracle.Prompt {
// 	return oracle.Prompt{
// 		Purpose:       "A test purpose",
// 		InputHistory:  []string{"GivenInput", "GivenInput2"},
// 		OutputHistory: []string{"IdealOutput", "IdealOutput2"},
// 		Question:      "A test question",
// 		Pages:         oracle.NewDocuments(bytes.NewBufferString("page1"), bytes.NewBufferString("page2")),
// 		Artifacts:     oracle.NewArtifacts(bytes.NewBufferString("test"), bytes.NewBufferString("test2")),
// 	}
// }
//
// func testMessages() client.Messages {
// 	return client.MessageFromPrompt(testPrompt())
// }
//
// func TestCreateTextCompletionRequestHeaders(t *testing.T) {
// 	t.Parallel()
// 	messages := testMessages()
// 	req, err := client.CreateTextCompletionRequest("dummy-token-openai", client.GPT4, messages)
// 	if err != nil {
// 		t.Errorf("Error creating request: %s", err)
// 	}
// 	auth := req.Header.Get("Authorization")
// 	if auth != "Bearer dummy-token-openai" {
// 		t.Errorf("Expected dummy-token-openai, got %s", auth)
// 	}
// 	contentType := req.Header.Get("Content-Type")
// 	if contentType != "application/json" {
// 		t.Errorf("Expected application/json, got %s", contentType)
// 	}
// }
//
// func TestCreateTextCompletionRequest(t *testing.T) {
// 	t.Parallel()
// 	messages := testMessages()
// 	req, err := client.CreateTextCompletionRequest("dummy-token-openai", client.GPT4, messages)
// 	if err != nil {
// 		t.Errorf("Error creating request: %s", err)
// 	}
// 	want := fmt.Sprintf(`{"model":"%s","messages":[{"role":"system","content":"A test purpose"},{"role":"user","content":"GivenInput"},{"role":"assistant","content":"IdealOutput"},{"role":"user","content":"GivenInput2"},{"role":"assistant","content":"IdealOutput2"},{"role":"user","content":"A test question"},{"role":"user","content":"Reference 1: page1"},{"role":"user","content":"Reference 2: page2"}]}%v`, client.GPT4, "\n")
// 	data, err := io.ReadAll(req.Body)
// 	if err != nil {
// 		t.Errorf("Error reading request body: %s", err)
// 	}
// 	got := string(data)
// 	if !cmp.Equal(got, want) {
// 		t.Error(cmp.Diff(want, got))
// 	}
// }
//
// func TestParseTextCompletionResponse(t *testing.T) {
// 	t.Parallel()
// 	f, err := os.Open("testdata/response.json")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer f.Close()
// 	content, cErr := client.ParseTextCompletionReponse(f)
// 	if cErr != nil {
// 		t.Errorf("Error parsing response: %s", err)
// 	}
// 	data, err := io.ReadAll(content)
// 	if err != nil {
// 		t.Errorf("Error reading response: %s", err)
// 	}
// 	got := string(data)
// 	want := "A woodchuck would chuck as much wood as a woodchuck could chuck if a woodchuck could chuck wood."
// 	if got != want {
// 		t.Errorf("Expected %s', got %s", want, got)
// 	}
// }
//
// func TestNewChatGPTToken(t *testing.T) {
// 	t.Parallel()
// 	c := client.NewChatGPT("dummy-token-openai")
// 	if c.Token != "dummy-token-openai" {
// 		t.Errorf("Expected dummy-token-openai, got %s", c.Token)
// 	}
// }
//
// func TestMessageFromPrompt(t *testing.T) {
// 	t.Parallel()
// 	prompt := oracle.Prompt{}
// 	msg := client.MessageFromPrompt(prompt)
// 	if len(msg) != 2 {
// 		t.Errorf("Expected 2 message, got %d ::: %v", len(msg), msg)
// 	}
// 	prompt.Purpose = "A test purpose"
//
// 	prompt.InputHistory = []string{"GivenInput", "GivenInput2"}
// 	prompt.OutputHistory = []string{"IdealOutput", "IdealOutput2"}
//
// 	prompt.Question = "A test question"
//
// 	want := client.Messages{
// 		client.TextMessage{
// 			Role:    client.RoleSystem,
// 			Content: "A test purpose",
// 		},
// 		client.TextMessage{
// 			Role:    client.RoleUser,
// 			Content: "GivenInput",
// 		},
// 		client.TextMessage{
// 			Role:    client.RoleAssistant,
// 			Content: "IdealOutput",
// 		},
// 		client.TextMessage{
// 			Role:    client.RoleUser,
// 			Content: "GivenInput2",
// 		},
// 		client.TextMessage{
// 			Role:    client.RoleAssistant,
// 			Content: "IdealOutput2",
// 		},
// 		client.TextMessage{
// 			Role:    client.RoleUser,
// 			Content: "A test question",
// 		},
// 	}
// 	got := client.MessageFromPrompt(prompt)
// 	if !cmp.Equal(want, got) {
// 		t.Error(cmp.Diff(want, got))
// 	}
// }
//
// func TestGetCompletionWithInvalidTokenErrors(t *testing.T) {
// 	t.Parallel()
// 	c := client.NewChatGPT("dummy-token-openai")
// 	_, err := c.Completion(context.Background(), oracle.Prompt{})
// 	want := &client.ClientError{}
// 	if !errors.As(err, want) {
// 		t.Errorf("Expected %v, got %v", want, err)
// 	}
// 	if want.StatusCode != 401 {
// 		t.Errorf("Expected 401, got %d", want.StatusCode)
// 	}
// }
//
// func TestErrorRateLimitThatHitsNoLimitSignalsRetryImmediately(t *testing.T) {
// 	t.Parallel()
// 	req := http.Response{}
// 	req.Header = http.Header{
// 		"X-Ratelimit-Remaining-Requests": []string{"3500"},
// 		"X-Ratelimit-Remaining-Tokens":   []string{"90000"},
// 		"X-Ratelimit-Reset-Requests":     []string{"17ms"},
// 		"X-Ratelimit-Reset-Tokens":       []string{"28ms"},
// 	}
// 	err := client.ErrorRateLimitExceeded(req)
// 	want := &client.RateLimitError{}
// 	if !errors.As(err, want) {
// 		t.Errorf("wanted %v, got %v", want, err)
// 	}
// 	if want.RetryAfter != 0 {
// 		t.Errorf("Expected 0, got %d", want.RetryAfter)
// 	}
// }
//
// func TestErrorRateLimitHitsTokenLimitSignalsRetryAfterTokensReset(t *testing.T) {
// 	t.Parallel()
// 	req := http.Response{}
// 	req.Header = http.Header{
// 		"X-Ratelimit-Remaining-Requests": []string{"3500"},
// 		"X-Ratelimit-Remaining-Tokens":   []string{"0"},
// 		"X-Ratelimit-Reset-Requests":     []string{"17ms"},
// 		"X-Ratelimit-Reset-Tokens":       []string{"28ms"},
// 	}
// 	err := client.ErrorRateLimitExceeded(req)
// 	want := &client.RateLimitError{}
// 	if !errors.As(err, want) {
// 		t.Errorf("wanted %v, got %v", want, err)
// 	}
// 	if want.RetryAfter != time.Duration(28*time.Millisecond) {
// 		t.Errorf("Expected 28ms, got %d", want.RetryAfter)
// 	}
// }
//
// func TestErrorRateLimitsHitsRetryLimitsSignalsTryAfterRequestsReset(t *testing.T) {
// 	t.Parallel()
// 	req := http.Response{}
// 	req.Header = http.Header{
// 		"X-Ratelimit-Remaining-Requests": []string{"0"},
// 		"X-Ratelimit-Remaining-Tokens":   []string{"90000"},
// 		"X-Ratelimit-Reset-Requests":     []string{"17ms"},
// 		"X-Ratelimit-Reset-Tokens":       []string{"28ms"},
// 	}
// 	err := client.ErrorRateLimitExceeded(req)
// 	want := &client.RateLimitError{}
// 	if !errors.As(err, want) {
// 		t.Errorf("wanted %v, got %v", want, err)
// 	}
// 	if want.RetryAfter != time.Duration(17*time.Millisecond) {
// 		t.Errorf("Expected 17ms, got %d", want.RetryAfter)
// 	}
// }
//
// func TestCreateVisionRequest(t *testing.T) {
// 	t.Parallel()
// 	messages := testMessages()
// 	req, err := client.CreateVisionRequest("dummy-token-openai", messages)
// 	if err != nil {
// 		t.Errorf("Error creating request: %s", err)
// 	}
// 	data, err := io.ReadAll(req.Body)
// 	if err != nil {
// 		t.Errorf("Error reading request body: %s", err)
// 	}
// 	got := string(data)
// 	want := `{"model":"gpt-4-vision-preview","messages":[{"role":"system","content":"A test purpose"},{"role":"user","content":"GivenInput"},{"role":"assistant","content":"IdealOutput"},{"role":"user","content":"GivenInput2"},{"role":"assistant","content":"IdealOutput2"},{"role":"user","content":"A test question"},{"role":"user","content":"Reference 1: page1"},{"role":"user","content":"Reference 2: page2"}],"max_tokens":300}` + "\n"
// 	if err != nil {
// 		t.Errorf("Error unmarshalling request body: %s", err)
// 	}
// 	if !cmp.Equal(got, want) {
// 		t.Error(cmp.Diff(want, got))
// 	}
// }
//
// func TestURLToURI(t *testing.T) {
// 	t.Parallel()
// 	pngURL, _ := url.Parse("https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png")
// 	got, err := client.URLToURI(*pngURL)
// 	if err != nil {
// 		t.Errorf("Error converting url to data uri: %s", err)
// 	}
// 	want := "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png"
// 	if want != got {
// 		t.Fatalf("Expected %s, got %s", want, got)
// 	}
// }
//
// func TestURLToURI_IfNotValidType(t *testing.T) {
// 	t.Parallel()
// 	nonPNGURL, _ := url.Parse("https://www.google.com")
// 	_, err := client.URLToURI(*nonPNGURL)
// 	if err == nil {
// 		t.Errorf("Expected error, got nil")
// 	}
// }
//
// func TestTextToSpeechRequest(t *testing.T) {
// 	t.Parallel()
// 	got, err := client.CreateTextToSpeechRequest("testToken", "someText")
// 	if err != nil {
// 		t.Errorf("Error creating response: %s", err)
// 	}
// 	buf := bytes.NewReader([]byte(`{"model":"tts-1","input":"someText","voice":"alloy"}`))
// 	want := httptest.NewRequest("POST", "https://api.openai.com/v1/audio/speech", buf)
// 	if got.Method != want.Method {
// 		t.Errorf("Expected %s, got %s", want.Method, got.Method)
// 	}
// 	if got.URL.Host != want.URL.Host {
// 		t.Errorf("Expected %s, got %s", want.URL.Host, got.URL.Host)
// 	}
// 	if got.URL.Path != want.URL.Path {
// 		t.Errorf("Expected %s, got %s", want.URL.Path, got.URL.Path)
// 	}
// 	wantBody := []byte{}
// 	gotBody := []byte{}
// 	_, _ = want.Body.Read(wantBody)
// 	_, _ = got.Body.Read(gotBody)
//
// 	if !cmp.Equal(wantBody, gotBody) {
// 		t.Error(cmp.Diff(wantBody, gotBody))
// 	}
// }
//
// func TestSpeechToTextRequest(t *testing.T) {
// 	t.Parallel()
// 	path := t.TempDir() + "/test.wav"
// 	err := os.WriteFile(path, []byte("test"), 0644)
// 	if err != nil {
// 		t.Errorf("Error creating test file: %s", err)
// 	}
// 	got, err := client.CreateSpeechToTextRequest("", []byte{})
// 	if err != nil {
// 		t.Errorf("Error creating response: %s", err)
// 	}
// 	contentType := "multipart/form-data; boundary="
// 	if !strings.Contains(got.Header.Get("Content-Type"), contentType) {
// 		t.Errorf("Expected content type to contain %v, got %s", contentType, got.Header.Get("Content-Type"))
// 	}
// 	if got.URL.Host != "api.openai.com" {
// 		t.Errorf("Expected api.openai.com, got %s", got.URL.Host)
// 	}
// 	if got.URL.Path != "/v1/audio/transcriptions" {
// 		t.Errorf("Expected /v1/audio/speech, got %s", got.URL.Path)
// 	}
// 	body := new(bytes.Buffer)
// 	body.ReadFrom(got.Body)
// 	if body.Len() == 0 {
// 		t.Errorf("Expected non-empty body, got empty body")
// 	}
// }
