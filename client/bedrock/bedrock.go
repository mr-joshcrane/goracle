package bedrock

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// Prompt interface defines the contract for prompt data
type Prompt interface {
	GetPurpose() string
	GetHistory() ([]string, []string)
	GetQuestion() string
	GetReferences() [][]byte
	GetResponseFormat() []string
}

// Bedrock client for AWS Bedrock Runtime
type Bedrock struct {
	ModelIdentifier string
	runtime         *bedrockruntime.Client
}

// NewBedrock creates a new Bedrock client with the given model identifier
func NewBedrock(modelIdentifier string) *Bedrock {
	return &Bedrock{
		ModelIdentifier: modelIdentifier,
		runtime:         nil, // Will be initialized lazily in Completion
	}
}

// NewBedrockWithInferenceProfile creates a new Bedrock client specifically for inference profiles
// This assumes Claude family by default since most inference profiles are for Claude models
func NewBedrockWithInferenceProfile(inferenceProfileArn string) *Bedrock {
	return &Bedrock{
		ModelIdentifier: inferenceProfileArn,
		runtime:         nil, // Will be initialized lazily in Completion
	}
}

// WithModel changes the model used by the client
func (b *Bedrock) WithModel(modelName string) error {
	// Look up the model configuration
	model, exists := Models[modelName]
	if !exists {
		supportedModels := make([]string, 0, len(Models))
		for k := range Models {
			supportedModels = append(supportedModels, k)
		}
		return fmt.Errorf("model %s not found. Supported models include: %s", modelName, strings.Join(supportedModels, ", "))
	}

	b.ModelIdentifier = model.ID
	return nil
}

func CapabilityCheck(modelIdentifier string, prompt Prompt) error {
	// For inference profiles, we can't easily determine vision support
	// So we only check for known foundation model IDs
	if IsInferenceProfile(modelIdentifier) {
		return nil // Skip check for inference profiles
	}

	// Extract the actual model ID from the identifier
	modelID := ExtractModelIDFromIdentifier(modelIdentifier)

	// Look up model configuration by scanning Models map
	var modelConfig *ModelConfig
	for _, config := range Models {
		if config.ID == modelID {
			modelConfig = &config
			break
		}
	}

	// If we can't find the model config, allow it through (could be a newer model)
	if modelConfig == nil {
		return nil
	}

	// Check if model supports vision for image references
	if !modelConfig.SupportsVision {
		for _, ref := range prompt.GetReferences() {
			kind := detectDataKind(ref)
			if kind == DataKindImage {
				return fmt.Errorf("model %s does not support image references", modelIdentifier)
			}
		}
	}
	return nil
}


// Completion makes a completion request to AWS Bedrock Runtime
func (b *Bedrock) Completion(ctx context.Context, prompt Prompt) (io.Reader, error) {
	// Check if the model supports the requested capabilities
	if err := CapabilityCheck(b.ModelIdentifier, prompt); err != nil {
		return nil, err
	}

	// Initialize AWS client lazily
	if b.runtime == nil {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
		b.runtime = bedrockruntime.NewFromConfig(cfg)
	}

	// Determine model family and format request accordingly
	family := ExtractModelFamily(b.ModelIdentifier)

	var requestBody []byte
	var err error

	switch family {
	case ModelFamilyClaude:
		claudeReq, err := PromptToClaudeRequest(prompt, 4096)
		if err != nil {
			return nil, fmt.Errorf("failed to create Claude request: %w", err)
		}
		requestBody, err = json.Marshal(claudeReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Claude request: %w", err)
		}
	case ModelFamilyTitan:
		titanReq, err := PromptToTitanRequest(prompt, 4096)
		if err != nil {
			return nil, fmt.Errorf("failed to create Titan request: %w", err)
		}
		requestBody, err = json.Marshal(titanReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Titan request: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported model family: %s for model %s", family, b.ModelIdentifier)
	}

	// Call AWS Bedrock Runtime API
	input := &bedrockruntime.InvokeModelInput{
		ModelId: aws.String(b.ModelIdentifier),
		Body:    requestBody,
	}

	output, err := b.runtime.InvokeModel(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Bedrock model %s: %w", b.ModelIdentifier, err)
	}

	// Parse response based on model family
	switch family {
	case ModelFamilyClaude:
		return ParseClaudeResponse(output.Body)
	case ModelFamilyTitan:
		return ParseTitanResponse(output.Body)
	default:
		return nil, fmt.Errorf("unsupported model family for response parsing: %s", family)
	}
}

// Claude request/response structures
type ClaudeRequest struct {
	AnthropicVersion string          `json:"anthropic_version"`
	System           string          `json:"system,omitempty"`
	Messages         []ClaudeMessage `json:"messages"`
	MaxTokens        int             `json:"max_tokens"`
}

type ClaudeMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ClaudeContentBlock struct {
	Type   string             `json:"type"`
	Text   string             `json:"text,omitempty"`
	Source *ClaudeImageSource `json:"source,omitempty"`
}

type ClaudeImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// Titan request/response structures
type TitanRequest struct {
	InputText     string `json:"inputText"`
	MaxTokenCount int    `json:"maxTokenCount"`
}

type TitanResponse struct {
	Results []struct {
		OutputText string `json:"outputText"`
	} `json:"results"`
}

// DataKind represents the type of data contained in the []byte
type DataKind int

const (
	DataKindUnknown DataKind = iota
	DataKindImage
	DataKindText
)

// detectDataKind determines the kind of data in the byte slice.
func detectDataKind(data []byte) DataKind {
	_, _, err := image.Decode(bytes.NewReader(data))
	if err == nil {
		return DataKindImage
	}
	return DataKindText // Default to text if not an image
}

func processReference(data []byte) ([]ClaudeContentBlock, error) {
	kind := detectDataKind(data)

	switch kind {
	case DataKindImage:
		// Re-encode to JPEG for consistency
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("image decoding failed: %w", err)
		}
		var buf bytes.Buffer
		err = jpeg.Encode(&buf, img, nil)
		if err != nil {
			return nil, fmt.Errorf("image re-encoding failed: %w", err)
		}
		base64Data := base64.StdEncoding.EncodeToString(buf.Bytes())
		return []ClaudeContentBlock{{
			Type: "image",
			Source: &ClaudeImageSource{
				Type:      "base64",
				MediaType: "image/jpeg",
				Data:      base64Data,
			},
		}}, nil

	case DataKindText:
		return []ClaudeContentBlock{{
			Type: "text",
			Text: string(data),
		}}, nil

	default:
		return nil, fmt.Errorf("unknown data kind")
	}
}

// PromptToClaudeRequest converts a Prompt to Claude API request format
func PromptToClaudeRequest(prompt Prompt, maxTokens int) (*ClaudeRequest, error) {
	req := &ClaudeRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		System:           prompt.GetPurpose(),
		MaxTokens:        maxTokens,
		Messages:         []ClaudeMessage{},
	}

	// Add history
	inputHistory, outputHistory := prompt.GetHistory()
	for i := range inputHistory {
		req.Messages = append(req.Messages, ClaudeMessage{
			Role:    "user",
			Content: inputHistory[i],
		})
		if i < len(outputHistory) {
			req.Messages = append(req.Messages, ClaudeMessage{
				Role:    "assistant",
				Content: outputHistory[i],
			})
		}
	}

	// Build content array for the current question and references
	contentBlocks := []ClaudeContentBlock{{
		Type: "text",
		Text: prompt.GetQuestion(),
	}}

	// Add references to the same message
	references := prompt.GetReferences()
	for _, ref := range references {
		refBlocks, err := processReference(ref)
		if err != nil {
			// Skip invalid references rather than failing entirely
			continue
		}
		contentBlocks = append(contentBlocks, refBlocks...)
	}

	// Add the user message with all content
	req.Messages = append(req.Messages, ClaudeMessage{
		Role:    "user",
		Content: contentBlocks,
	})

	return req, nil
}

// PromptToTitanRequest converts a Prompt to Titan API request format
func PromptToTitanRequest(prompt Prompt, maxTokens int) (*TitanRequest, error) {
	// Titan uses a simple text input format
	inputText := prompt.GetPurpose() + "\n\n" + prompt.GetQuestion()

	req := &TitanRequest{
		InputText:     inputText,
		MaxTokenCount: maxTokens,
	}

	return req, nil
}

// ParseClaudeResponse parses a Claude API response
func ParseClaudeResponse(responseBody []byte) (io.Reader, error) {
	var resp ClaudeResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, err
	}

	var completion string
	for _, content := range resp.Content {
		completion += content.Text
	}

	return strings.NewReader(completion), nil
}

// ParseTitanResponse parses a Titan API response
func ParseTitanResponse(responseBody []byte) (io.Reader, error) {
	var resp TitanResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, err
	}

	var completion string
	for _, result := range resp.Results {
		completion += result.OutputText
	}

	return strings.NewReader(completion), nil
}
