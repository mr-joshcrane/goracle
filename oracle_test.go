package oracle_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"strings"
	"testing"
	"testing/iotest"

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

func TestAskWithNewDocumentProvidesCorrectPrompt(t *testing.T) {
	t.Parallel()
	r := strings.NewReader("It's time to shine")
	o, c := createTestOracle("", nil)
	document := oracle.NewDocuments(r)
	_, err := o.Ask(ctx(), "Hello World", document)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	readers, err := c.P.GetPages()
	if err != nil {
		t.Errorf("Error getting pages: %s", err)
	}
	if len(readers) != 1 {
		t.Errorf("Expected 1 reference, got %d", len(readers))
	}
	compareReaders(t, readers[0], r)
}

func TestAskWithNewDocumentsProvidesCorrectPrompt(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	r1 := strings.NewReader("It's time to shine")
	r2 := strings.NewReader("It's time to shine again")
	documents := oracle.NewDocuments(r1, r2)

	_, err := o.Ask(ctx(), "Hello World", documents)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	readers, err := c.P.GetPages()
	if err != nil {
		t.Errorf("Error getting pages: %s", err)
	}
	if len(readers) != 2 {
		t.Errorf("Expected 2 references, got %d", len(readers))
	}
	compareReaders(t, readers[0], r1)
}

func TestAskWithNewVisualsProvidesCorrectPrompt(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	v := image.NewRGBA(image.Rect(0, 0, 100, 100))
	visuals := oracle.NewVisuals(v)

	_, err := o.Ask(ctx(), "Hello World", visuals)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	readers, err := c.P.GetPages()
	if err != nil {
		t.Errorf("Error getting pages: %s", err)
	}
	if len(readers) != 1 {
		t.Errorf("Expected 1 reference, got %d", len(readers))
	}
	var want bytes.Buffer
	err = png.Encode(&want, v)
	if err != nil {
		t.Errorf("Error encoding image: %s", err)
	}
	got := readers[0]
	compareReaders(t, got, &want)
}

func TestAskWithNewArtifactProvidesCorrectPrompt(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	want := new(bytes.Buffer)
	artifacts := oracle.NewArtifacts(want)

	_, err := o.Ask(ctx(), "Please create an artifact", artifacts)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	art, err := c.P.GetArtifacts()
	if err != nil {
		t.Errorf("Error getting artifacts: %s", err)
	}
	if len(art) != 1 {
		t.Errorf("Expected 1 artifact, got %d", len(artifacts))
	}
	got := art[0]
	if got == nil {
		t.Fatal("Expected artifact, got nil")
	}
	if got != want {
		t.Fatalf("Artifact pointers do not match")
	}
}

func TestImageRefCanBeRead(t *testing.T) {
	t.Parallel()
	v := image.NewRGBA(image.Rect(0, 0, 100, 100))
	page := oracle.ImagePage{
		Image: v,
	}

	got, err := page.GetContent()
	if err != nil {
		t.Errorf("Error reading image: %s", err)
	}
	if len(got) != 298 {
		t.Errorf("Expected 40000 bytes, got %d", len(got))
	}
}

func TestAskWithNewArtifactsCanProvideCorrectPrompt(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	buf1 := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	artifacts := oracle.NewArtifacts(buf1, buf2)
	_, err := o.Ask(ctx(), "Please create two artifacts", artifacts)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	art, err := c.P.GetArtifacts()
	if err != nil {
		t.Errorf("Error getting artifacts: %s", err)
	}
	if len(art) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(art))
	}
	if art[0] != buf1 {
		t.Errorf("Expected artifact 1 to be buf1")
	}
	if art[1] != buf2 {
		t.Errorf("Expected artifact 2 to be buf2")
	}
}

func TestPromptWithGetArtifactsProvidesCorrectPrompt(t *testing.T) {
	t.Parallel()
	buf1 := bytes.NewBufferString("It's time to shine")
	buf2 := bytes.NewBufferString("It's time to shine again")
	prompt := oracle.Prompt{
		Purpose:   "You are a test Oracle",
		Question:  "Please create two artifacts",
		Artifacts: oracle.NewArtifacts(buf1, buf2),
	}
	artifacts, _ := prompt.GetArtifacts()
	if len(artifacts) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(artifacts))
	}
	if artifacts[0] != buf1 {
		t.Errorf("Expected artifact 1 to be buf1")
	}
	if artifacts[1] != buf2 {
		t.Errorf("Expected artifact 2 to be buf2")
	}
}

func TestPromptWithFaultyReferencesGivesErrorFeedback(t *testing.T) {
	t.Parallel()
	badReader := iotest.ErrReader(errors.New("Error reading page"))
	badWriter, err := os.CreateTemp("", "oracle_test")
	if err != nil {
		t.Fatal(err)
	}
	badWriter.Close()
	prompt := oracle.Prompt{
		Purpose:   "You are a test Oracle",
		Question:  "Please create two artifacts",
		Pages:     oracle.NewDocuments(badReader),
		Artifacts: oracle.NewArtifacts(badWriter),
	}
	_, err = prompt.GetArtifacts()
	if err == nil {
		t.Errorf("Expected error reading artifacts, got %v", err)
	}
	_, err = prompt.GetPages()
	if err == nil {
		t.Errorf("Expected error reading pages, got %v", err)
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

func ExampleOracle_Ask_visualsToText() {
	// Text request with reference to an [image.Image]
	c := client.NewDummyClient("This is a black square!", nil)
	o := oracle.NewOracle("", oracle.WithClient(c))
	ctx := context.Background()

	v := image.NewRGBA(image.Rect(0, 0, 100, 100))
	visuals := oracle.NewVisuals(v)
	answer, err := o.Ask(ctx, "What color and shape is this image?", visuals)
	if err != nil {
		panic(err)
	}
	fmt.Println(answer)
	// Output: This is a black square!
}

func ExampleOracle_Ask_textToImage() {
	// Request an image based on a text prompt
	// Requires an [Artifact] to write the image to
	c := client.NewDummyClient("I drew you an image!", nil)
	o := oracle.NewOracle("", oracle.WithClient(c))
	ctx := context.Background()

	buf := new(bytes.Buffer)
	artifacts := oracle.NewArtifacts(buf)
	answer, err := o.Ask(ctx, "Please create a simple red square on a black background, nothing else", artifacts)
	if err != nil {
		panic(err)
	}
	fmt.Println(answer)
	// Output: I drew you an image!
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

func ExampleOracle_Ask_chainCalls() {
	// Single API calls are great, but chaining them together is where the magic happens
	c := client.NewDummyClient("A response!", nil)
	o := oracle.NewOracle("", oracle.WithClient(c))
	ctx := context.Background()

	//

	name, _ := o.Ask(ctx, "I'm of opening a coffee shop. I want it to be Tucan themed. What are some ideas for branding and names?")
	fmt.Println(name)
	notes := oracle.NewDocuments(strings.NewReader("Brand notes: Bright colored, hipster, retro geek chic"))
	theme, _ := o.Ask(ctx, "What should the theme be? Whats my gimmick? Work with my existing notes", notes)
	fmt.Println(theme)
	// In reality, we would use real images for inspiration
	inspirations := oracle.NewVisuals(image.Image(nil), image.Image(nil))
	prompt, _ := o.Ask(ctx,
		"Given what we've talked about, create an LLM prompt that would create a logo for my coffee shop. Take inspiration from these photos",
		inspirations,
	)

	f, _ := os.Create("logo.png")
	logo := oracle.NewArtifacts(f)
	o.Ask(ctx, prompt, logo)
}
