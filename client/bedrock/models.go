package bedrock

type ModelFamily string

const (
	ModelFamilyClaude  ModelFamily = "claude"
	ModelFamilyTitan   ModelFamily = "titan"
	ModelFamilyUnknown ModelFamily = "unknown"
)

type ModelConfig struct {
	ID               string
	Family           ModelFamily
	MaxTokens        int
	SupportsVision   bool
	SupportsStreaming bool
}

// Predefined model configurations for foundation models
var Models = map[string]ModelConfig{
	// Anthropic Claude Models
	"claude-3-5-sonnet-20241022": {
		ID:               "anthropic.claude-3-5-sonnet-20241022-v2:0",
		Family:           ModelFamilyClaude,
		MaxTokens:        8192,
		SupportsVision:   true,
		SupportsStreaming: true,
	},
	"claude-3-5-sonnet-20240620": {
		ID:               "anthropic.claude-3-5-sonnet-20240620-v1:0",
		Family:           ModelFamilyClaude,
		MaxTokens:        8192,
		SupportsVision:   true,
		SupportsStreaming: true,
	},
	"claude-3-5-haiku-20241022": {
		ID:               "anthropic.claude-3-5-haiku-20241022-v1:0",
		Family:           ModelFamilyClaude,
		MaxTokens:        8192,
		SupportsVision:   true,
		SupportsStreaming: true,
	},
	"claude-3-opus-20240229": {
		ID:               "anthropic.claude-3-opus-20240229-v1:0",
		Family:           ModelFamilyClaude,
		MaxTokens:        4096,
		SupportsVision:   true,
		SupportsStreaming: true,
	},
	"claude-3-sonnet-20240229": {
		ID:               "anthropic.claude-3-sonnet-20240229-v1:0",
		Family:           ModelFamilyClaude,
		MaxTokens:        4096,
		SupportsVision:   true,
		SupportsStreaming: true,
	},
	"claude-3-haiku-20240307": {
		ID:               "anthropic.claude-3-haiku-20240307-v1:0",
		Family:           ModelFamilyClaude,
		MaxTokens:        4096,
		SupportsVision:   true,
		SupportsStreaming: true,
	},
	
	// Amazon Titan Models
	"titan-text-lite": {
		ID:               "amazon.titan-text-lite-v1",
		Family:           ModelFamilyTitan,
		MaxTokens:        4096,
		SupportsVision:   false,
		SupportsStreaming: true,
	},
	"titan-text-express": {
		ID:               "amazon.titan-text-express-v1",
		Family:           ModelFamilyTitan,
		MaxTokens:        8192,
		SupportsVision:   false,
		SupportsStreaming: true,
	},
}