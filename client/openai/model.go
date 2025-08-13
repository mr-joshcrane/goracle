package openai

type ModelConfig struct {
	Name                   string
	SupportsSystemMessages bool
	SupportsVision         bool
}

var Models = map[string]ModelConfig{
	"gpt-5": {
		Name:                   "gpt-5",
		SupportsSystemMessages: true,
		SupportsVision:         true,
	},
	"gpt-5-mini": {
		Name:                   "gpt-5-mini",
		SupportsSystemMessages: true,
		SupportsVision:         true,
	},
	"gpt-5-nano": {
		Name:                   "gpt-5-nano",
		SupportsSystemMessages: true,
		SupportsVision:         true,
	},
	"gpt-4.1": {
		Name:                   "gpt-4.1",
		SupportsSystemMessages: true,
		SupportsVision:         true,
	},
	"gpt-4o": {
		Name:                   "gpt-4o",
		SupportsSystemMessages: true,
		SupportsVision:         true,
	},
	"gpt-4o-mini": {
		Name:                   "gpt-4o-mini",
		SupportsSystemMessages: true,
		SupportsVision:         true,
	},
	"o1-preview": {
		Name:                   "o1-preview",
		SupportsSystemMessages: false,
		SupportsVision:         false,
	},
	"o1-mini": {
		Name:                   "o1-mini",
		SupportsSystemMessages: false,
		SupportsVision:         false,
	},
}
