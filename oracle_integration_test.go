//go:build integration

package oracle_test

import (
	"bytes"
	"context"
	"image"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/oracle"
)

func TestGiveExample(t *testing.T) {
	t.Parallel()
	o := newTestOracle(t)
	buf := new(bytes.Buffer)
	prompt := oracle.Prompt{
		Purpose: "To answer if a number is odd or even in a specific format",
		InputHistory: []string{
			"2",
			"3",
		},
		OutputHistory: []string{
			"+++even+++",
			"---odd---",
		},
		Question: "6",
		Target:   buf,
	}
	err := o.Completion(context.TODO(), prompt)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}

	got := buf.String()
	if got != "+++even+++" {
		t.Fatal(cmp.Diff("+++even+++", got))
	}
}

func TestAsk(t *testing.T) {
	t.Parallel()
	o := newTestOracle(t)
	o.SetPurpose("You always answer questions with the number 42.")
	question := "What is the meaning of life?"
	answer, err := o.Ask(context.TODO(), question)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	if !strings.Contains(answer, "42") {
		t.Errorf("Expected 42, got %s", answer)
	}
}

func TestCreateImageDescribeImage(t *testing.T) {
	t.Parallel()
	o := newTestOracle(t)
	buf := new(bytes.Buffer)
	err := o.CreateImage(context.Background(), "please create a simple black square, nothing else", buf)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	question := "What color and shape is this?"
	image, _, err := image.Decode(buf)
	if err != nil {
		t.Errorf("Error decoding image: %s", err)
	}
	images := oracle.WithImages(image)
	answer, err := o.DescribeImage(context.TODO(), question, images)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	if !strings.Contains(answer, "black") {
		t.Errorf("Expected black, got %s", answer)
	}
}

func TestCreateTranscriptFromAudio(t *testing.T) {
	t.Parallel()
	o := newTestOracle(t)
	buf := new(bytes.Buffer)
	err := o.CreateAudio(context.Background(), "Hello, world!", buf)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	answer, err := o.CreateTranscript(context.Background(), buf)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	if !strings.Contains(answer, "Hello") && !strings.Contains(answer, "world") {
		t.Errorf("Expected Hello, world!, got %s", answer)
	}
}

func newTestOracle(t *testing.T) *oracle.Oracle {
	t.Helper()
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		t.Fatal("OPENAI_API_KEY is not set")
	}
	return oracle.NewOracle(token)
}
