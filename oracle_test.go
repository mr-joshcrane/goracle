package oracle_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/oracle"
	"github.com/mr-joshcrane/oracle/client"
)

func compareReaders(t *testing.T, x, y io.Reader) {
	xBytes, err := io.ReadAll(x)
	if err != nil {
		t.Fatalf("Error reading reference X: %s", err)
	}
	yBytes, err := io.ReadAll(y)
	if err != nil {
		t.Fatalf("Error reading reference Y: %s", err)
	}
	if !bytes.Equal(xBytes, yBytes) {
		t.Fatalf("Expected %s, got %s", string(xBytes), string(yBytes))
	}
}

func ctx() context.Context {
	return context.TODO()
}

func createTestOracle(fixedResponse string, err error) (*oracle.Oracle, *client.Dummy) {
	c := client.NewDummyClient(fixedResponse, err)
	o := oracle.NewOracle("", oracle.WithClient(c))
	o.SetPurpose("You are a test Oracle")
	return o, c
}

func TestTextToSpeechWithTrivialTransformerReturnsItsInput(t *testing.T) {
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

func TestSpeechToTextWithTrivialTransformerReturnsItsInput(t *testing.T) {
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

func TestAsk_ProvidesWellFormedPromptToLLM(t *testing.T) {
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

func TestResetReturnsABlankOracle(t *testing.T) {
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

func TestAskWithStringLiteralReferenceReturnsCorrectPrompt(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	_, err := o.Ask(ctx(), "Hello World", "It's time to shine")
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	got := c.P.GetPages()
	if len(got) != 1 {
		t.Errorf("Expected 1 reference, got %d", len(got))
	}
	if string(got[0]) != "It's time to shine" {
		t.Errorf("Expected It's time to shine, got %s", string(got[0]))
	}
}

func TestAskWithStringLiteralReferneceProvidesCorrectPrompt(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	_, err := o.Ask(ctx(), "Hello World",
		"It's time to shine",
		"It's time to shine again",
	)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	got := c.P.GetPages()
	if len(got) != 2 {
		t.Errorf("Expected 2 references, got %d", len(got))
	}
	if string(got[0]) != "It's time to shine" {
		t.Errorf("Expected It's time to shine, got %s", string(got[0]))
	}
	if string(got[1]) != "It's time to shine again" {
		t.Errorf("Expected It's time to shine again, got %s", string(got[1]))
	}
}

func TestFileReference_ValidFileReturnsByteContents(t *testing.T) {
	t.Parallel()
	want := []byte("cheese is made from milk")
	path := t.TempDir() + "/text.txt"
	err := os.WriteFile(path, want, 0644)
	if err != nil {
		t.Fatal(err)
	}
	got := oracle.File(path)
	if !bytes.Equal(got, want) {
		t.Errorf("Expected %s, got %s", string(want), string(got))
	}
}

func TestFileReference_InvalidFileReturnsEmptyBytes(t *testing.T) {
	t.Parallel()
	got := oracle.File("invalid/path")
	if len(got) != 0 {
		t.Errorf("Expected empty bytes, got %s", string(got))
	}
}

func TestFolderReference_ValidFolderReturnsByteContentsOfFiles(t *testing.T) {
	t.Parallel()
	content1 := []byte("cheese is made from milk")
	content2 := []byte("the sky is blue")
	dir := t.TempDir()
	err := os.WriteFile(dir+"/text1.txt", content1, 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(dir+"/text2.txt", content2, 0644)
	if err != nil {
		t.Fatal(err)
	}
	got := oracle.Folder(dir)
	want := []byte("cheese is made from milk\nthe sky is blue\n")
	if !cmp.Equal(got, want) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestFolderReference_InvalidFolderReturnsEmptyBytes(t *testing.T) {
	t.Parallel()
	got := oracle.Folder("invalid/path")
	if len(got) != 0 {
		t.Errorf("Expected empty bytes, got %s", string(got))
	}
}

// Examples

func ExampleOracle_Ask_standardTextCompletion() {
	// Basic request response text flow
	c := client.NewDummyClient("A friendly LLM response!", nil)
	o := oracle.NewOracle("", oracle.WithClient(c))
	ctx := context.Background()
	answer, err := o.Ask(ctx, "A user question")
	if err != nil {
		panic(err)
	}
	fmt.Println(answer)
	// Output: A friendly LLM response!
}

func ExampleOracle_Transform_textToSpeech() {
	// Transform text to audio
	// Requires a [strings.Reader] as the source
	// Returns an [io.Reader] as the target
	c := client.NewDummyClient("", nil)
	o := oracle.NewOracle("", oracle.WithClient(c))
	ctx := context.Background()

	speech, err := o.TextToSpeech(ctx, "Hello, world!")
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(speech)
	if err != nil {
		panic(err)
	}
	_ = os.WriteFile("hello_world.wav", data, 0644)
}

func ExampleOracle_Transform_speechToText() {
	// Transform audio to text
	// Requires an [io.Reader] as the source
	// Returns a [string] as the target
	c := client.NewDummyClient("A transcript of your audio data", nil)
	o := oracle.NewOracle("", oracle.WithClient(c))
	ctx := context.Background()

	// In reality, this will be some reader containing audio data
	r := bytes.NewReader([]byte(c.FixedResponse))
	answer, err := o.SpeechToText(ctx, r)
	if err != nil {
		panic(err)
	}
	fmt.Println(answer)
	// Output: A transcript of your audio data
}
