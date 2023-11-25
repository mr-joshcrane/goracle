package oracle_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/oracle"
	"github.com/mr-joshcrane/oracle/client"
)

func ctx() context.Context {
	return context.TODO()
}

func createTestOracle(fixedResponse string, err error) (*oracle.Oracle, *client.Dummy) {
	c := client.NewDummyClient(fixedResponse, err)
	o := oracle.NewOracle("", oracle.WithClient(c))
	o.SetPurpose("You are a test Oracle")
	return o, c
}

func TestTextToSpeech(t *testing.T) {
	t.Parallel()
	o, _ := createTestOracle("Hello World", nil)
	r, err := o.TextToSpeech(ctx(), "Hello World")
	if err != nil {
		t.Errorf("Error generating speech from text: %s", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Errorf("Error reading speech: %s", err)
	}
	got := string(data)
	if got != "Hello World" {
		t.Errorf("Expected Hello World, got %s", string(data))
	}
}

func TestSpeechToText(t *testing.T) {
	t.Parallel()
	o, _ := createTestOracle("", nil)
	reader := bytes.NewReader([]byte("Hello World"))
	got, err := o.SpeechToText(ctx(), reader)
	if err != nil {
		t.Errorf("Error generating speech from text: %s", err)
	}
	if got != "Hello World" {
		t.Errorf("Expected Hello World, got %s", got)
	}
}

func TestAsk(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("Hello World", nil)
	got, err := o.Ask(ctx(), "Hello World")
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	if got != "Hello World" {
		t.Errorf("Expected Hello World, got %s", got)
	}
	want := oracle.Prompt{
		Purpose:  "You are a test Oracle",
		Question: "Hello World",
	}
	if !cmp.Equal(c.P, want) {
		t.Fatal(cmp.Diff(want, c.P))
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("Hello World", nil)
	o.GiveExample("An example that should be forgotten", "And the ideal response that should be forgotten")
	o.SetPurpose("Setting a purpose that should be forgotten")
	_, err := o.Ask(ctx(), "A question that should be forgotten")
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	o.Reset()
	_, err = o.Ask(ctx(), "Hello World")
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	want := oracle.Prompt{
		InputHistory:  []string{},
		OutputHistory: []string{},
		Question:      "Hello World",
	}
	if !cmp.Equal(c.P, want) {
		t.Fatal(cmp.Diff(want, c.P))
	}
}

func TestPromptAccessorMethods(t *testing.T) {
	t.Parallel()
	prompt := oracle.Prompt{
		Purpose:       "You are a test Oracle",
		Question:      "Hello World",
		InputHistory:  []string{"Hello World"},
		OutputHistory: []string{"Hello World"},
	}
	if prompt.GetPurpose() != "You are a test Oracle" {
		t.Errorf("Expected You are a test Oracle, got %s", prompt.GetPurpose())
	}
	if prompt.GetQuestion() != "Hello World" {
		t.Errorf("Expected Hello World, got %s", prompt.GetQuestion())
	}
	inHistory, outHistory := prompt.GetHistory()
	if !cmp.Equal(inHistory, []string{"Hello World"}) {
		t.Fatal(cmp.Diff([]string{"Hello World"}, inHistory))
	}
	if !cmp.Equal(outHistory, []string{"Hello World"}) {
		t.Fatal(cmp.Diff([]string{"Hello World"}, outHistory))
	}
}
