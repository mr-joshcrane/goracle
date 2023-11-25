package client_test

import (
	"bytes"
	"context"
	"image"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/mr-joshcrane/oracle"
	"github.com/mr-joshcrane/oracle/client"
)

func TestGetVisualCompletion_Images(t *testing.T) {
	t.Parallel()
	c := client.NewChatGPT(os.Getenv("OPENAI_API_KEY"))
	buf := new(bytes.Buffer)
	testImageOne := image.NewRGBA(image.Rect(0, 0, 100, 100))
	testImageTwo := image.NewRGBA(image.Rect(0, 0, 100, 99))
	prompt := oracle.Prompt{
		Purpose:  "You tell me what the color in this image is",
		Question: "What color is this?",
		Images:   []image.Image{testImageOne, testImageTwo},
		Target:   buf,
	}
	err := c.Completion(context.TODO(), prompt)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	answer := buf.String()
	if !strings.Contains(answer, "black") {
		t.Errorf("Expected black, got %s", answer)
	}
}

func TestGetVisualCompletion_URIs(t *testing.T) {
	t.Parallel()
	c := client.NewChatGPT(os.Getenv("OPENAI_API_KEY"))
	buf := new(bytes.Buffer)
	u1, _ := url.Parse("https://upload.wikimedia.org/wikipedia/commons/thumb/4/44/Microsoft_logo.svg/1024px-Microsoft_logo.svg.png")
	u2, _ := url.Parse("https://upload.wikimedia.org/wikipedia/commons/6/60/Microsoft_logo_%281975%29.svg")
	prompt := oracle.Prompt{
		Purpose:  "You guess the famous companies",
		Question: "Guess the comapny?",
		Urls:     []url.URL{*u1, *u2},
		Target:   buf,
	}
	err := c.Completion(context.TODO(), prompt)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	answer := buf.String()
	if !strings.Contains(answer, "Microsoft") {
		t.Errorf("Expected Microsoft, got %s", answer)
	}
}

func TestTextToSpeechToText(t *testing.T) {
	t.Parallel()
	token := os.Getenv("OPENAI_API_KEY")
	data, err := client.GenerateSpeech(token, "Hello world")
	if err != nil {
		t.Errorf("Error generating speech: %s", err)
	}
	if len(data) == 0 {
		t.Errorf("Expected data, got empty string")
	}
	text, err := client.SpeechToText(token, data)
	if err != nil {
		t.Errorf("Error converting speech to text: %s", err)
	}
	if !strings.Contains(text, "Hello") && !strings.Contains(text, "world") {
		t.Errorf("Expected Hello world, got %s", text)
	}
}
