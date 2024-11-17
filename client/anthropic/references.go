package anthropic

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif" // Register image formats
	"image/jpeg"  // For re-encoding
	_ "image/jpeg"
	_ "image/png"
)

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

func processReference(data []byte) (interface{}, error) {
	kind := detectDataKind(data)

	switch kind {
	case DataKindImage:
		// Re-encode to JPEG for consistency if already an image
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil { // Should not happen, but handle it defensively
			return nil, fmt.Errorf("image decoding failed after positive detection: %w", err)
		}
		var buf bytes.Buffer
		err = jpeg.Encode(&buf, img, nil) // Use nil for default options or customize
		if err != nil {
			return nil, fmt.Errorf("image re-encoding failed: %w", err)
		}
		return []ImagePayload{createImageContent(buf.Bytes())}, nil // Wrap in a slice as before

	case DataKindText:
		return string(data), nil // Directly use the string

	default:
		return nil, fmt.Errorf("unknown data kind")
	}
}

func createImageContent(imageData []byte) ImagePayload {
	return ImagePayload{
		Type: "image",
		Source: struct {
			Type      string `json:"type"`
			MediaType string `json:"media_type"`
			Data      string `json:"data"`
		}{
			Type:      "base64",
			MediaType: "image/jpeg", // Consistent media type
			Data:      base64.StdEncoding.EncodeToString(imageData),
		},
	}
}
