package anthropic

type ModelConfig struct {
	Provider       string
	Name           string
	SupportsVision bool
	Description    string
	MaxTokens      int
}

var Models = map[string]ModelConfig{

	"ClaudeOpus4": {
		Name:           "claude-opus-4-20250514",
		SupportsVision: true,
		MaxTokens:      64000,
		Description:    "Claude Opus 4 is the latest model with advanced capabilities and a large context window.",
	},
	"ClaudeSonnet4": {
		Name:           "claude-sonnet-4-20250514",
		SupportsVision: true,
		MaxTokens:      64000,
		Description:    "Claude Sonnet 4 is designed for complex reasoning tasks with a large context window.",
	},
	"ClaudeSonnet3_7": {
		Name:           "claude-3-7-sonnet-20250219",
		SupportsVision: true,
		MaxTokens:      64000,
		Description:    "Claude Sonnet 3.7 is optimized for advanced reasoning and complex tasks.",
	},
	"ClaudeSonnet3_5": {
		Name:           "claude-3-5-sonnet-20241022",
		SupportsVision: true,
		MaxTokens:      64000,
		Description:    "Claude Sonnet 3.5 (New) is a versatile model with enhanced capabilities for various tasks.",
	},
	"ClaudeHaiku3_5": {
		Name:           "claude-3-5-haiku-20241022",
		SupportsVision: true,
		MaxTokens:      64000,
		Description:    "Claude Haiku 3.5 is designed for tasks requiring concise and efficient responses.",
	},
}
