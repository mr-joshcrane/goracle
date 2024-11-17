package google

type ModelConfig struct {
	Provider       string
	Name           string
	SupportsVision bool
	Description    string
}

var Models = map[string]ModelConfig{
	"GeminiPro": {
		Provider:       "google",
		Name:           "gemini-1.5-pro-002",
		SupportsVision: true,
		Description:    "Created to be multimodal (text, images, code) and to scale across a wide range of tasks",
	},
	"ClaudeSonnet": {
		Provider:       "anthropic",
		Name:           "claude-3-5-sonnet-v2@20241022",
		SupportsVision: true,
		Description: `The upgraded Claude 3.5 Sonnet is now state-of-the-art 
									for a variety of tasks including real-world software engineering,
									enhanced agentic capabilities, and computer use.`,
	},
	"ClaudeHaiku": {
		Provider:       "anthropic",
		Name:           "claude-3-5-haiku@20241022",
		SupportsVision: false,
		Description: `Claude 3 Haiku is Anthropic's fastest vision and text model 
									for near-instant responses to simple queries, meant for seamless
									AI experiences mimicking human interactions.`,
	},
}
