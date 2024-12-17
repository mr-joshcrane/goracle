package anthropic

type ModelConfig struct {
	Provider       string
	Name           string
	SupportsVision bool
	Description    string
	MaxTokens int
}

var Models = map[string]ModelConfig{
	"ClaudeSonnet": {
		Name:           "claude-3-5-sonnet-latest",
		SupportsVision: true,
		MaxTokens: 100_000_000,
		Description: `The upgraded Claude 3.5 Sonnet is now state-of-the-art 
									for a variety of tasks including real-world software engineering,
									enhanced agentic capabilities, and computer use.`,
	},
	"ClaudeHaiku": {
		Provider:       "anthropic",
		Name:           "claude-3-5-haiku-latest",
		SupportsVision: false,
		MaxTokens: 100_000_000,
		Description: `Claude 3 Haiku is Anthropic's fastest vision and text model 
									for near-instant responses to simple queries, meant for seamless
									AI experiences mimicking human interactions.`,
	},
}
