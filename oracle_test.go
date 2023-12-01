package oracle_test

import (
	"bytes"
	"context"
	"image"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mr-joshcrane/oracle"
	"github.com/mr-joshcrane/oracle/client"
)

var IgnoreReader = cmpopts.IgnoreUnexported(bytes.Buffer{}, oracle.Artifact{}, oracle.ImagePage{}, strings.Reader{}, bytes.Reader{}, oracle.DocumentPage{})

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

func TestAsk_NewDocument(t *testing.T) {
	t.Parallel()
	r := strings.NewReader("It's time to shine")
	o, c := createTestOracle("", nil)
	document := oracle.NewDocuments(r)
	_, err := o.Ask(ctx(), "Hello World", document)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	want := oracle.Prompt{
		Purpose:  "You are a test Oracle",
		Question: "Hello World",
		Pages:    document,
	}
	if !cmp.Equal(c.P, want, IgnoreReader) {
		t.Fatal(cmp.Diff(want, c.P, IgnoreReader))
	}
	data := c.P.GetPages()[0]
	got, err := io.ReadAll(data)
	if err != nil {
		t.Errorf("Error reading reference: %s", err)
	}
	if !cmp.Equal(got, []byte("It's time to shine")) {
		t.Errorf("Expected It's time to shine, got %s", got)
	}
}

func TestAsk_NewDocuments(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	r1 := strings.NewReader("It's time to shine")
	r2 := strings.NewReader("It's time to shine again")
	documents := oracle.NewDocuments(r1, r2)

	_, err := o.Ask(ctx(), "Hello World", documents)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	want := oracle.Prompt{
		Purpose:  "You are a test Oracle",
		Question: "Hello World",
		Pages:    documents,
	}
	if !cmp.Equal(c.P, want, IgnoreReader) {
		t.Fatal(cmp.Diff(want, c.P, IgnoreReader))
	}
	refs := c.P.GetPages()
	if len(refs) < 1 {
		t.Fatal("Expected 1 reference, got 0")
	}
	got, err := io.ReadAll(refs[0])
	if err != nil {
		t.Errorf("Error reading reference: %s", err)
	}
	if !cmp.Equal(got, []byte("It's time to shine"), IgnoreReader) {
		t.Errorf("Expected It's time to shine, got %s", got)
	}
	if len(refs) < 2 {
		t.Fatal("Expected 2 references, got 1")
	}
	got, err = io.ReadAll(refs[1])
	if err != nil {
		t.Errorf("Error reading reference: %s", err)
	}
	if !cmp.Equal(got, []byte("It's time to shine again"), IgnoreReader) {
		t.Errorf("Expected It's time to shine again, got %s", got)
	}
}

func TestAsk_NewVisuals(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	v := image.NewRGBA(image.Rect(0, 0, 100, 100))
	visuals := oracle.NewVisuals(v)

	_, err := o.Ask(ctx(), "Hello World", visuals)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	want := oracle.Prompt{
		Purpose:  "You are a test Oracle",
		Question: "Hello World",
		Pages:    visuals,
	}
	if !cmp.Equal(c.P, want, IgnoreReader) {
		t.Fatal(cmp.Diff(want, c.P, IgnoreReader))
	}
}

func TestAsk_NewArtifact(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	buf := new(bytes.Buffer)
	artifacts := oracle.NewArtifacts(buf)

	_, err := o.Ask(ctx(), "Please create an artifact", artifacts)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	want := oracle.Prompt{
		Purpose:   "You are a test Oracle",
		Question:  "Please create an artifact",
		Artifacts: artifacts,
	}
	if !cmp.Equal(c.P, want, IgnoreReader) {
		t.Fatal(cmp.Diff(want, c.P, IgnoreReader))
	}
}

func TestImageRef_Read(t *testing.T) {
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

func TestAsk_NewArtifacts(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	buf1 := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	artifacts := oracle.NewArtifacts(buf1, buf2)
	_, err := o.Ask(ctx(), "Please create two artifacts", artifacts)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	want := oracle.Prompt{
		Purpose:   "You are a test Oracle",
		Question:  "Please create two artifacts",
		Artifacts: artifacts,
	}
	if !cmp.Equal(c.P, want, IgnoreReader) {
		t.Fatal(cmp.Diff(want, c.P, IgnoreReader))
	}
}

func TestPrompt_GetArtifacts(t *testing.T) {
	t.Parallel()
	buf1 := bytes.NewBufferString("It's time to shine")
	buf2 := bytes.NewBufferString("It's time to shine again")
	prompt := oracle.Prompt{
		Purpose:   "You are a test Oracle",
		Question:  "Please create two artifacts",
		Artifacts: oracle.NewArtifacts(buf1, buf2),
	}
	artifacts := prompt.GetArtifacts()
	if len(artifacts) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(artifacts))
	}
	got, err := io.ReadAll(artifacts[0])
	if err != nil {
		t.Errorf("Error reading artifact: %s", err)
	}
	if !cmp.Equal(got, []byte("It's time to shine"), IgnoreReader) {
		t.Errorf("Expected It's time to shine, got %s", got)
	}
	got, err = io.ReadAll(artifacts[1])
	if err != nil {
		t.Errorf("Error reading artifact: %s", err)
	}
	if !cmp.Equal(got, []byte("It's time to shine again"), IgnoreReader) {
		t.Errorf("Expected It's time to shine again, got %s", got)
	}
}
