package bedrock_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/goracle"
	"github.com/mr-joshcrane/goracle/client/bedrock"
)

func testPrompt() goracle.Prompt {
	return goracle.Prompt{
		Purpose:       "You are a helpful assistant",
		InputHistory:  []string{"Hello"},
		OutputHistory: []string{"Hi there!"},
		Question:      "What is the capital of France?",
		References:    [][]byte{[]byte("Reference text")},
	}
}

func TestNewBedrock(t *testing.T) {
	t.Parallel()
	identifier := "anthropic.claude-3-5-sonnet-20241022-v2:0"
	
	client := bedrock.NewBedrock(identifier)
	if client == nil {
		t.Fatal("Expected non-nil Bedrock client")
	}
	
	if client.ModelIdentifier != identifier {
		t.Errorf("Expected model identifier %s, got %s", identifier, client.ModelIdentifier)
	}
}

func TestBedrockWithModel(t *testing.T) {
	t.Parallel()
	client := bedrock.NewBedrock("anthropic.claude-3-5-sonnet-20241022-v2:0")
	
	tests := []struct {
		name      string
		modelName string
		expectErr bool
	}{
		{
			name:      "valid claude model",
			modelName: "claude-3-5-sonnet-20241022",
			expectErr: false,
		},
		{
			name:      "valid titan model", 
			modelName: "titan-text-express",
			expectErr: false,
		},
		{
			name:      "unknown model",
			modelName: "unknown-model",
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := client.WithModel(tt.modelName)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error for model %s, got nil", tt.modelName)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error for model %s, got %v", tt.modelName, err)
			}
		})
	}
}

func TestPromptToClaudeRequest(t *testing.T) {
	t.Parallel()
	
	prompt := testPrompt()
	
	request, err := bedrock.PromptToClaudeRequest(prompt, 4096)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if request.System != prompt.GetPurpose() {
		t.Errorf("Expected system %s, got %s", prompt.GetPurpose(), request.System)
	}
	
	if request.AnthropicVersion != "bedrock-2023-05-31" {
		t.Errorf("Expected anthropic_version bedrock-2023-05-31, got %s", request.AnthropicVersion)
	}
	
	if request.MaxTokens != 4096 {
		t.Errorf("Expected max tokens 4096, got %d", request.MaxTokens)
	}
	
	// Should have 3 messages: history pair (user+assistant) + current question with references
	expectedMessages := 3
	if len(request.Messages) != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, len(request.Messages))
	}
	
	// Check history messages
	if request.Messages[0].Role != "user" || request.Messages[0].Content != "Hello" {
		t.Errorf("Expected first message to be user: Hello, got %s: %v", request.Messages[0].Role, request.Messages[0].Content)
	}
	
	if request.Messages[1].Role != "assistant" || request.Messages[1].Content != "Hi there!" {
		t.Errorf("Expected second message to be assistant: Hi there!, got %s: %v", request.Messages[1].Role, request.Messages[1].Content)
	}
	
	// Check the current question message - should have content blocks
	if request.Messages[2].Role != "user" {
		t.Errorf("Expected third message to be user role, got %s", request.Messages[2].Role)
	}
	
	// Content should be an array of content blocks, not a string
	content, ok := request.Messages[2].Content.([]bedrock.ClaudeContentBlock)
	if !ok {
		t.Fatalf("Expected content to be []bedrock.ClaudeContentBlock, got %T", request.Messages[2].Content)
	}
	
	if len(content) != 2 {
		t.Errorf("Expected 2 content blocks (question + reference), got %d", len(content))
	}
	
	// Check that first block is the question text
	if content[0].Type != "text" || content[0].Text != "What is the capital of France?" {
		t.Errorf("Expected first content block to be text: 'What is the capital of France?', got type: %s, text: %s", content[0].Type, content[0].Text)
	}
	
	// Check that second block is the reference text
	if content[1].Type != "text" || content[1].Text != "Reference text" {
		t.Errorf("Expected second content block to be text: 'Reference text', got type: %s, text: %s", content[1].Type, content[1].Text)
	}
}

func TestPromptToTitanRequest(t *testing.T) {
	t.Parallel()
	
	prompt := testPrompt()
	
	request, err := bedrock.PromptToTitanRequest(prompt, 4096)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if request.MaxTokenCount != 4096 {
		t.Errorf("Expected max token count 4096, got %d", request.MaxTokenCount)
	}
	
	expectedInput := "You are a helpful assistant\n\nWhat is the capital of France?"
	if request.InputText != expectedInput {
		t.Errorf("Expected input text %s, got %s", expectedInput, request.InputText)
	}
}

func TestParseClaudeResponse(t *testing.T) {
	t.Parallel()
	
	responseBody := []byte(`{"content":[{"text":"Paris is the capital of France."}]}`)
	
	result, err := bedrock.ParseClaudeResponse(responseBody)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	expected := "Paris is the capital of France."
	content, _ := io.ReadAll(result)
	if !cmp.Equal(string(content), expected) {
		t.Fatal(cmp.Diff(expected, string(content)))
	}
}

func TestParseClaudeResponseMultipleContent(t *testing.T) {
	t.Parallel()
	
	responseBody := []byte(`{"content":[{"text":"Paris "},{"text":"is the capital of France."}]}`)
	
	result, err := bedrock.ParseClaudeResponse(responseBody)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	expected := "Paris is the capital of France."
	content, _ := io.ReadAll(result)
	if !cmp.Equal(string(content), expected) {
		t.Fatal(cmp.Diff(expected, string(content)))
	}
}

func TestParseTitanResponse(t *testing.T) {
	t.Parallel()
	
	responseBody := []byte(`{"results":[{"outputText":"Paris is the capital of France."}]}`)
	
	result, err := bedrock.ParseTitanResponse(responseBody)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	expected := "Paris is the capital of France."
	content, _ := io.ReadAll(result)
	if !cmp.Equal(string(content), expected) {
		t.Fatal(cmp.Diff(expected, string(content)))
	}
}

func TestParseTitanResponseMultipleResults(t *testing.T) {
	t.Parallel()
	
	responseBody := []byte(`{"results":[{"outputText":"Paris "},{"outputText":"is the capital of France."}]}`)
	
	result, err := bedrock.ParseTitanResponse(responseBody)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	expected := "Paris is the capital of France."
	content, _ := io.ReadAll(result)
	if !cmp.Equal(string(content), expected) {
		t.Fatal(cmp.Diff(expected, string(content)))
	}
}

// Test identifier detection functions
func TestIsInferenceProfile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		identifier string
		expected   bool
	}{
		{
			name:       "inference profile ARN",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:application-inference-profile/abcdef123456",
			expected:   true,
		},
		{
			name:       "foundation model ID",
			identifier: "anthropic.claude-3-5-sonnet-20241022-v2:0",
			expected:   false,
		},
		{
			name:       "cross-region inference profile ARN",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:inference-profile/us.anthropic.claude-3-5-sonnet-20241022-v2:0",
			expected:   true,
		},
		{
			name:       "empty string",
			identifier: "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := bedrock.IsInferenceProfile(tt.identifier)
			if result != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestExtractModelFamily(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		identifier string
		expected   bedrock.ModelFamily
	}{
		{
			name:       "anthropic claude foundation model",
			identifier: "anthropic.claude-3-5-sonnet-20241022-v2:0",
			expected:   bedrock.ModelFamilyClaude,
		},
		{
			name:       "amazon titan foundation model",
			identifier: "amazon.titan-text-express-v1",
			expected:   bedrock.ModelFamilyTitan,
		},
		{
			name:       "cross-region inference profile with claude",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:inference-profile/us.anthropic.claude-3-5-sonnet-20241022-v2:0",
			expected:   bedrock.ModelFamilyClaude,
		},
		{
			name:       "unknown model provider",
			identifier: "openai.gpt-4",
			expected:   bedrock.ModelFamilyUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := bedrock.ExtractModelFamily(tt.identifier)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExtractModelIDFromIdentifier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		identifier string
		expected   string
	}{
		{
			name:       "foundation model ID returns itself",
			identifier: "anthropic.claude-3-5-sonnet-20241022-v2:0",
			expected:   "anthropic.claude-3-5-sonnet-20241022-v2:0",
		},
		{
			name:       "cross-region inference profile extracts embedded model ID",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:inference-profile/us.anthropic.claude-3-5-sonnet-20241022-v2:0",
			expected:   "anthropic.claude-3-5-sonnet-20241022-v2:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := bedrock.ExtractModelIDFromIdentifier(tt.identifier)
			if !cmp.Equal(result, tt.expected) {
				t.Fatal(cmp.Diff(tt.expected, result))
			}
		})
	}
}

func TestValidateIdentifier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		identifier string
		expectErr  bool
	}{
		{
			name:       "valid foundation model ID",
			identifier: "anthropic.claude-3-5-sonnet-20241022-v2:0",
			expectErr:  false,
		},
		{
			name:       "valid application inference profile ARN",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:application-inference-profile/abcdef123456",
			expectErr:  false,
		},
		{
			name:       "empty identifier",
			identifier: "",
			expectErr:  true,
		},
		{
			name:       "malformed ARN",
			identifier: "arn:aws:bedrock:us-west-2",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := bedrock.ValidateIdentifier(tt.identifier)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestPromptToClaudeRequestWithImage(t *testing.T) {
	t.Parallel()
	
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(5, 5, color.RGBA{255, 0, 0, 255}) // Red pixel
	
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, nil)
	if err != nil {
		t.Fatal(err)
	}
	
	prompt := goracle.Prompt{
		Purpose:       "You are a helpful assistant",
		InputHistory:  []string{},
		OutputHistory: []string{},
		Question:      "What do you see in this image?",
		References:    [][]byte{buf.Bytes()}, // Image as reference
	}
	
	request, err := bedrock.PromptToClaudeRequest(prompt, 4096)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Should have 1 message with 2 content blocks: text + image
	if len(request.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(request.Messages))
	}
	
	content, ok := request.Messages[0].Content.([]bedrock.ClaudeContentBlock)
	if !ok {
		t.Fatalf("Expected content to be []bedrock.ClaudeContentBlock, got %T", request.Messages[0].Content)
	}
	
	if len(content) != 2 {
		t.Errorf("Expected 2 content blocks (text + image), got %d", len(content))
	}
	
	// Check text block
	if content[0].Type != "text" || content[0].Text != "What do you see in this image?" {
		t.Errorf("Expected first block to be text question, got type: %s, text: %s", content[0].Type, content[0].Text)
	}
	
	// Check image block
	if content[1].Type != "image" {
		t.Errorf("Expected second block to be image, got type: %s", content[1].Type)
	}
	
	if content[1].Source == nil {
		t.Fatal("Expected image source to be set")
	}
	
	if content[1].Source.Type != "base64" {
		t.Errorf("Expected source type to be base64, got %s", content[1].Source.Type)
	}
	
	if content[1].Source.MediaType != "image/jpeg" {
		t.Errorf("Expected media type to be image/jpeg, got %s", content[1].Source.MediaType)
	}
	
	if content[1].Source.Data == "" {
		t.Error("Expected base64 data to be set")
	}
}

func TestCapabilityCheck(t *testing.T) {
	t.Parallel()
	
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(5, 5, color.RGBA{255, 0, 0, 255}) // Red pixel
	
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, nil)
	if err != nil {
		t.Fatal(err)
	}
	
	tests := []struct {
		name           string
		modelID        string
		hasImage       bool
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name:      "Claude model with image should pass",
			modelID:   "anthropic.claude-3-5-sonnet-20241022-v2:0",
			hasImage:  true,
			expectErr: false,
		},
		{
			name:           "Titan model with image should fail",
			modelID:        "amazon.titan-text-express-v1",
			hasImage:       true,
			expectErr:      true,
			expectedErrMsg: "does not support image references",
		},
		{
			name:      "Titan model without image should pass", 
			modelID:   "amazon.titan-text-express-v1",
			hasImage:  false,
			expectErr: false,
		},
		{
			name:      "Inference profile should pass (skip check)",
			modelID:   "arn:aws:bedrock:us-west-2:123456789012:application-inference-profile/abcdef123456",
			hasImage:  true,
			expectErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			prompt := goracle.Prompt{
				Purpose:       "You are a helpful assistant",
				InputHistory:  []string{},
				OutputHistory: []string{},
				Question:      "What do you see?",
				References:    [][]byte{},
			}
			
			if tt.hasImage {
				prompt.References = [][]byte{buf.Bytes()}
			}
			
			err := bedrock.CapabilityCheck(tt.modelID, prompt)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Errorf("Expected error to contain %q, got %q", tt.expectedErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}