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

func TestOracleIntegration_ExamplesGuideOutput(t *testing.T) {
	t.Parallel()
	o := newTestOracle(t)
	o.GiveExample("2", "+++even+++")
	o.GiveExample("3", "---odd---")
	got, err := o.Ask(context.Background(), "6")
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}

	if got != "+++even+++" {
		t.Fatal(cmp.Diff("+++even+++", got))
	}
}

func TestOracleIntegration_PurposeGuidesOutput(t *testing.T) {
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

func TestOracleIntegration_CreateAnImageThenDescribeIt(t *testing.T) {
	t.Parallel()
	o := newTestOracle(t)
	buf := new(bytes.Buffer)
	artifact := oracle.NewArtifacts(buf)
	_, err := o.Ask(context.TODO(), "please create a simple red square on a black background, nothing else", artifact)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	image, _, err := image.Decode(buf)
	if err != nil {
		t.Errorf("Error decoding image: %s", err)
	}
	images := oracle.NewVisuals(image)
	answer, err := o.Ask(context.TODO(), "What color and shape is this?", images)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	if !strings.Contains(answer, "red") && !strings.Contains(answer, "square") {
		t.Errorf("Expected red, got %s", answer)
	}
}

func TestOracleIntegration_CreateSpeechThenTranscribeIt(t *testing.T) {
	t.Parallel()
	o := newTestOracle(t)
	speech, err := o.TextToSpeech(context.TODO(), "Hello, world!")
	if err != nil {
		t.Errorf("error generating speech from text: %s", err)
	}

	answer, err := o.SpeechToText(context.TODO(), speech)
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
