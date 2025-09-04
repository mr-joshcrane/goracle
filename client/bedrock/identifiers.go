package bedrock

import (
	"errors"
	"strings"
)

// IsInferenceProfile determines if the given identifier is an AWS Bedrock inference profile ARN
func IsInferenceProfile(identifier string) bool {
	if identifier == "" {
		return false
	}
	
	// Check if it's an ARN format: arn:aws:bedrock:region:account:resource-type/resource-id
	if !strings.HasPrefix(identifier, "arn:aws:bedrock:") {
		return false
	}
	
	// Split ARN into components
	parts := strings.Split(identifier, ":")
	if len(parts) < 6 {
		return false
	}
	
	// Check the resource part (after the 5th colon)
	resourcePart := strings.Join(parts[5:], ":")
	
	// Valid inference profile resource types
	return strings.HasPrefix(resourcePart, "application-inference-profile/") ||
		   strings.HasPrefix(resourcePart, "inference-profile/")
}

// ExtractModelFamily determines the model family from an identifier
func ExtractModelFamily(identifier string) ModelFamily {
	if identifier == "" {
		return ModelFamilyUnknown
	}
	
	// For inference profiles, try to extract embedded model ID
	if IsInferenceProfile(identifier) {
		modelID := ExtractModelIDFromIdentifier(identifier)
		if modelID != "" {
			return extractFamilyFromModelID(modelID)
		}
		// For application inference profiles, we can't extract the model ID
		// Default to Claude since most inference profiles are for Claude models
		return ModelFamilyClaude
	}
	
	// For foundation model IDs, check the prefix
	return extractFamilyFromModelID(identifier)
}

func extractFamilyFromModelID(modelID string) ModelFamily {
	if strings.HasPrefix(modelID, "anthropic.claude") {
		return ModelFamilyClaude
	}
	if strings.HasPrefix(modelID, "amazon.titan") {
		return ModelFamilyTitan
	}
	return ModelFamilyUnknown
}

// ExtractModelIDFromIdentifier extracts the underlying model ID from an identifier
func ExtractModelIDFromIdentifier(identifier string) string {
	if identifier == "" {
		return ""
	}
	
	// If it's not an inference profile, return the identifier as-is (foundation model ID)
	if !IsInferenceProfile(identifier) {
		return identifier
	}
	
	// For application inference profiles, we cannot extract the model ID
	if strings.Contains(identifier, ":application-inference-profile/") {
		return ""
	}
	
	// For cross-region inference profiles, extract the embedded model ID
	if strings.Contains(identifier, ":inference-profile/") {
		parts := strings.Split(identifier, "/")
		if len(parts) >= 2 {
			resourceID := parts[1]
			// Remove the region prefix (e.g., "us.anthropic.claude..." -> "anthropic.claude...")
			if dotIndex := strings.Index(resourceID, "."); dotIndex > 0 {
				return resourceID[dotIndex+1:]
			}
		}
	}
	
	return ""
}

// ValidateIdentifier validates that the identifier is either a valid foundation model ID or inference profile ARN
func ValidateIdentifier(identifier string) error {
	if identifier == "" {
		return errors.New("identifier cannot be empty")
	}
	
	// If it looks like an ARN, validate ARN format
	if strings.HasPrefix(identifier, "arn:aws:bedrock:") {
		if !IsInferenceProfile(identifier) {
			return errors.New("invalid inference profile ARN format")
		}
		return nil
	}
	
	// For foundation model IDs, do basic validation
	if !strings.Contains(identifier, ".") {
		return errors.New("invalid foundation model ID format")
	}
	
	return nil
}