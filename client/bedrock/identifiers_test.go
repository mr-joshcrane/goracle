package bedrock

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

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
			name:       "amazon titan model ID",
			identifier: "amazon.titan-text-express-v1",
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
		{
			name:       "invalid ARN format",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:invalid-resource-type/abcdef123456",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsInferenceProfile(tt.identifier)
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
		expected   ModelFamily
	}{
		{
			name:       "anthropic claude foundation model",
			identifier: "anthropic.claude-3-5-sonnet-20241022-v2:0",
			expected:   ModelFamilyClaude,
		},
		{
			name:       "amazon titan foundation model",
			identifier: "amazon.titan-text-express-v1",
			expected:   ModelFamilyTitan,
		},
		{
			name:       "amazon titan lite model",
			identifier: "amazon.titan-text-lite-v1",
			expected:   ModelFamilyTitan,
		},
		{
			name:       "inference profile ARN with claude",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:application-inference-profile/abcdef123456",
			expected:   ModelFamilyUnknown, // Can't determine from profile ARN alone
		},
		{
			name:       "cross-region inference profile with claude",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:inference-profile/us.anthropic.claude-3-5-sonnet-20241022-v2:0",
			expected:   ModelFamilyClaude, // Can extract from embedded model ID
		},
		{
			name:       "unknown model provider",
			identifier: "openai.gpt-4",
			expected:   ModelFamilyUnknown,
		},
		{
			name:       "empty identifier",
			identifier: "",
			expected:   ModelFamilyUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ExtractModelFamily(tt.identifier)
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
			name:       "application inference profile returns empty",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:application-inference-profile/abcdef123456",
			expected:   "", // Cannot extract model ID from application profile ARN
		},
		{
			name:       "cross-region inference profile extracts embedded model ID",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:inference-profile/us.anthropic.claude-3-5-sonnet-20241022-v2:0",
			expected:   "anthropic.claude-3-5-sonnet-20241022-v2:0",
		},
		{
			name:       "cross-region titan profile",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:inference-profile/us.amazon.titan-text-express-v1",
			expected:   "amazon.titan-text-express-v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ExtractModelIDFromIdentifier(tt.identifier)
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
			name:       "valid cross-region inference profile ARN",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:inference-profile/us.anthropic.claude-3-5-sonnet-20241022-v2:0",
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
		{
			name:       "invalid resource type in ARN",
			identifier: "arn:aws:bedrock:us-west-2:123456789012:foundation-model/anthropic.claude-3-5-sonnet-20241022-v2:0",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateIdentifier(tt.identifier)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}