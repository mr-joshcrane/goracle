package goracle_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/goracle"
	"github.com/mr-joshcrane/goracle/client"
	"golang.org/x/tools/cover"
)

func TestMain(m *testing.M) {
	lastArg := os.Args[len(os.Args)-1]
	if lastArg == "ALL" {
		os.Exit(m.Run())
	}
	path := os.TempDir() + "/coverage.out"
	f, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	coverProfile := fmt.Sprintf("-coverprofile=%s", path)
	defer f.Close()
	tags := []string{"test", "-coverpkg=./...", "./...", coverProfile, "-args", "ALL"}
	if lastArg == "--integration" {
		tags = []string{"test", "-coverpkg=./...", "./...", "--tags=integration", coverProfile, "-args", "ALL"}
	}
	cmd := exec.Command("go", tags...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		os.Exit(1)
	}
	if strings.Contains(string(out), "FAIL") {
		fmt.Println(string(out))
	}
	profiles, err := cover.ParseProfiles(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var globalTested, globalTestable int
	for _, profile := range profiles {
		var tested, testable int
		for _, block := range profile.Blocks {
			lineCount := block.EndLine - block.StartLine
			if block.NumStmt > 0 {
				testable += lineCount
			}
			if block.Count > 0 && block.NumStmt > 0 {
				tested += lineCount
			}
		}
		percentageTested := float64(tested) / float64(testable) * 100
		fmt.Printf("%.2f%% - %s\n", percentageTested, profile.FileName)
		globalTested += tested
		globalTestable += testable
	}

	percentageTested := float64(globalTested) / float64(globalTestable) * 100
	fmt.Printf("\nOverall Coverage: %.2f%%\n", percentageTested)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func ctx() context.Context {
	return context.TODO()
}

func createTestOracle(fixedResponse string, err error) (*goracle.Oracle, *client.Dummy) {
	c := client.NewDummyClient(fixedResponse, err)
	o := goracle.NewOracle(c)
	o.SetPurpose("You are a test Oracle")
	return o, c
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
	want := goracle.Prompt{
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
	want := goracle.Prompt{
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
	prompt := goracle.Prompt{
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
	got := c.P.GetReferences()
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
		t.Fatalf("Error asking question: %s", err)
	}
	got := c.P.GetReferences()
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

func TestAskWithBytesReferenceReturnsCorrectPrompt(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	_, err := o.Ask(ctx(), "Hello World?",
		[]byte("It's time to shine"),
	)
	if err != nil {
		t.Fatalf("Error asking question: %s", err)
	}
	got := c.P.GetReferences()
	if len(got) != 1 {
		t.Errorf("Expected 1 reference, got %d", len(got))
	}
	if !bytes.Equal(got[0], []byte("It's time to shine")) {
		t.Errorf("Expected It's time to shine, got %s", string(got[0]))
	}
}

func TestAskWithImageReferenceProvidesCorrectPrompt(t *testing.T) {
	t.Parallel()
	o, c := createTestOracle("", nil)
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	_, err := o.Ask(ctx(), "Whats in this image?", img)
	if err != nil {
		t.Fatalf("Error asking question: %s", err)
	}
	got := c.P.GetReferences()
	if len(got) != 1 {
		t.Errorf("Expected 1 reference, got %d", len(got))
	}
	if !bytes.Equal(got[0], goracle.Image(img)) {
		t.Errorf("Expected %v, got %v", goracle.Image(img), got[0])
	}
}

func TestAskWithSomeUnknownReferenceReturnsError(t *testing.T) {
	t.Parallel()
	o, _ := createTestOracle("", nil)
	type Unsupported struct{}
	_, err := o.Ask(ctx(), "Whats in this image?", Unsupported{})
	if err == nil {
		t.Fatalf("Expected error asking question, got nil")
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
	got := goracle.File(path)
	if !bytes.Equal(got, want) {
		t.Errorf("Expected %s, got %s", string(want), string(got))
	}
}

func TestFileReference_InvalidFileReturnsEmptyBytes(t *testing.T) {
	t.Parallel()
	got := goracle.File("invalid/path")
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
	got := goracle.Folder(dir)
	want := []byte("cheese is made from milk\nthe sky is blue\n")
	if !cmp.Equal(got, want) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestFolderReference_InvalidFolderReturnsEmptyBytes(t *testing.T) {
	t.Parallel()
	got := goracle.Folder("invalid/path")
	if len(got) != 0 {
		t.Errorf("Expected empty bytes, got %s", string(got))
	}
}

func TestImageReference_ValidImageReturnsPNGEncodingasBytes(t *testing.T) {
	t.Parallel()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	got := goracle.Image(img)
	want := []byte{137, 80, 78, 71, 13, 10, 26, 10}
	if !bytes.Equal(got[:8], want) {
		t.Errorf("Expected %v, got %v", want, got[:8])
	}
}

func TestImageReference_InvalidImageReturnsPNGEncodingasBytes(t *testing.T) {
	t.Parallel()
	img := image.NewRGBA64(image.Rect(0, 0, 100, 100))
	img.Rect = image.Rect(0, 0, 0, -1)
	got := goracle.Image(img)
	want := []byte{}
	if !bytes.Equal(got, want) {
		t.Errorf("Expected %v, got %v", want, got[:8])
	}
}

// Examples

func ExampleOracle_Ask_standardTextCompletion() {
	// Basic request response text flow
	c := client.NewDummyClient("A friendly LLM response!", nil)
	o := goracle.NewOracle(c)
	ctx := context.Background()
	answer, err := o.Ask(ctx, "A user question")
	if err != nil {
		panic(err)
	}
	fmt.Println(answer)
	// Output: A friendly LLM response!
}

func ExampleOracle_Ask_withReferences() {
	// Basic request response text flow with multi-modal references
	c := client.NewDummyClient("Yes. There is a reference to swiss cheese in cheeseDocs/swiss.txt", nil)
	o := goracle.NewOracle(c)
	ctx := context.Background()
	nonCheeseImage := image.NewRGBA(image.Rect(0, 0, 100, 100))
	answer, err := o.Ask(ctx,
		"My question for you is, do any of my references make mention of swiss cheese?",
		"Some long chunk of text, that is notably non related",
		goracle.File("invoice.txt"),
		nonCheeseImage,
		goracle.Folder("~/cheeseDocs/"),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(answer)
	// Output: Yes. There is a reference to swiss cheese in cheeseDocs/swiss.txt
}

func ExampleOracle_Ask_withExamples() {
	// Examples allow you to guide the LLM with n-shot learning
	c := client.NewDummyClient("42", nil)
	o := goracle.NewOracle(c)
	o.GiveExample("Fear is the...", "mind killer")
	o.GiveExample("With great power comes...", "great responsibility")
	ctx := context.Background()
	answer, err := o.Ask(ctx, "What is the answer to life, the universe, and everything?")
	if err != nil {
		panic(err)
	}
	fmt.Println(answer)
	// Output: 42
}

func ExampleOracle_Ask_withConversationMemory() {
	// For when you want the responses to be Stateful
	// and depend on the previous answers
	// this is the default and matches the typical chatbot experience
	c := client.NewDummyClient("We talked about the answer to life, the universe, and everything", nil)
	o := goracle.NewOracle(c)
	ctx := context.Background()

	// This is the default, but can be set manually
	goracle.Stateful(o)
	_, err := o.Ask(ctx, "What is the answer to life, the universe, and everything?")
	if err != nil {
		panic(err)
	}
	answer, err := o.Ask(ctx, "What have we already talked about?")
	if err != nil {
		panic(err)
	}
	fmt.Println(answer)
	// Output: We talked about the answer to life, the universe, and everything
}

func ExampleOracle_Ask_withoutConversationMemory() {
	// For when you want the responses to be Stateless
	// and not depend on the previous answers/examples
	c := client.NewDummyClient("Nothing so far", nil)
	o := goracle.NewOracle(c)
	ctx := context.Background()

	//
	goracle.Stateless(o)
	_, err := o.Ask(ctx, "What is the answer to life, the universe, and everything?")
	if err != nil {
		panic(err)
	}
	answer, err := o.Ask(ctx, "What have we already talked about?")
	if err != nil {
		panic(err)
	}
	fmt.Println(answer)
	// Output: Nothing so far
}
