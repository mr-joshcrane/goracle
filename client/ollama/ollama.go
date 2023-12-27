package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Prompt interface {
	GetPurpose() string
	GetHistory() ([]string, []string)
	GetQuestion() string
	GetReferences() [][]byte
}

func DoChatCompletion(model string, endpoint string, prompt Prompt) (string, error) {
	body := NewChatCompletionRequest(model, prompt)
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	endpoint = fmt.Sprintf("%s/api/chat", endpoint)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	return ParseChatCompletionResponse(resp)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Messages []Message

func (m *Messages) Add(role string, content string) {
	message := Message{Role: role, Content: content}
	*m = append(*m, message)
}

func referenceFormatter(reference []byte, refNo int) string {
	return fmt.Sprintf("Reference %d: %s", refNo, reference)
}

type ChatCompletion struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Images   []string  `json:"images,omitempty"`
	Stream   bool      `json:"stream"`
	Raw      bool      `json:"raw"`
}

func PromptToMessages(prompt Prompt) Messages {
	var messages Messages
	messages.Add("system", prompt.GetPurpose())
	prevInput, prevOutputs := prompt.GetHistory()
	for i := range prevInput {
		messages.Add("user", prevInput[i])
		messages.Add("assistant", prevOutputs[i])
	}
	for i, ref := range prompt.GetReferences() {
		messages.Add("user", referenceFormatter(ref, i+1))
		messages.Add("assistant", "Reference added")
	}
	messages.Add("assistant", prompt.GetQuestion())
	return messages
}

func NewChatCompletionRequest(model string, prompt Prompt) ChatCompletion {
	messages := PromptToMessages(prompt)
	return ChatCompletion{
		Model:    model,
		Messages: messages,
		Stream:   true,
		Raw:      false,
	}
}

func ParseChatCompletionResponse(resp *http.Response) (string, error) {
	fmt.Println("Parsing...")
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama response status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	content := new(bytes.Buffer)
	decoder := json.NewDecoder(resp.Body)
	for {
		var body struct {
			Message Message `json:"message"`
		}
		err := decoder.Decode(&body)
		if err != nil {
			if err == io.EOF {
				break // end of Stream
			}
			return "", err
		}
		fmt.Print(body.Message.Content)

		_, err = content.WriteString(body.Message.Content)
		if err != nil {
			return "", err
		}

	}
	return content.String(), nil
}
