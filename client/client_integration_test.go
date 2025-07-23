//go:build integration

package client_test

import (
	"context"
	"image/jpeg"
	"os"

	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/goracle"
	"github.com/mr-joshcrane/goracle/client"
)

type testCase struct {
	Oracle      *goracle.Oracle
	Description string
}

func testCases(t *testing.T) []testCase {
	return []testCase{
		{
			Oracle:      goracle.NewAnthropicOracle(""),
			Description: "Anthropic_Oracle",
		},
		{
			Oracle:      newTestOracle(t),
			Description: "OpenAI_Oracle",
		},
		{
			Oracle:      newVertexTestOracle(t),
			Description: "VertexAI_Oracle",
		},
	}
}

func TestOracleIntegration_ExamplesGuideOutput(t *testing.T) {
	t.Parallel()
	cases := testCases(t)
	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			o := c.Oracle
			o.GiveExample("2", "+++even+++")
			o.GiveExample("3", "---odd---")
			got, err := o.Ask("6")
			if err != nil {
				t.Errorf("Error asking question: %s", err)
			}
			got = strings.TrimSpace(got)
			if got != "+++even+++" {
				t.Fatal(cmp.Diff("+++even+++", got))
			}
		})
	}
}

func TestOracleIntegration_PurposeGuidesOutput(t *testing.T) {
	t.Parallel()
	cases := testCases(t)
	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			o := c.Oracle
			o.SetPurpose("You always answer questions with the number 42.")
			question := "What is the meaning of life?"
			answer, err := o.Ask(question)
			if err != nil {
				t.Errorf("Error asking question: %s", err)
			}
			if !strings.Contains(answer, "42") {
				t.Errorf("Expected 42, got %s", answer)
			}
		})
	}
}

func TestOracleIntegration_RefersToDocuments(t *testing.T) {
	t.Parallel()
	cases := testCases(t)
	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			o := c.Oracle
			f, err := os.CreateTemp(t.TempDir(), c.Description+".txt")
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.WriteString("cheese is made from milk")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			answer, err := o.Ask("Can you repeat my two facts?",
				"the sky is blue",
				goracle.File(f.Name()),
			)
			if err != nil {
				t.Errorf("Error asking question: %s", err)
			}
			answer = strings.ToLower(answer)
			if !strings.Contains(answer, "the sky is blue") {
				t.Errorf("Error reading from buffer, expected the sky is blue, got %s", answer)
			}
			if !strings.Contains(answer, "cheese is made from milk") {
				t.Errorf("Error reading from file, expected cheese is made from milk, got %s", answer)
			}
		})
	}
}

func TestOracleIntegration_RefersToImages(t *testing.T) {
	cases := testCases(t)
	quokka, err := os.Open("testdata/quokka.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer quokka.Close()
	image, err := jpeg.Decode(quokka)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			o := c.Oracle
			answer, err := o.Ask("What species is this?", image)
			if err != nil {
				t.Errorf("Error asking question: %s", err)
			}
			answer = strings.ToLower(answer)
			if !strings.Contains(answer, "quokka") {
				t.Errorf("Error identifying image, expected 'quokka', got %s", answer)
			}
		})
	}
}

func TestOpenAIClient_TextToSpeechandSpeechToVoice(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	c := client.NewChatGPT(os.Getenv("OPENAI_API_KEY"))
	text := "hello world"
	audio, err := c.CreateAudio(ctx, text)
	if err != nil {
		t.Fatal(err)
	}
	transcript, err := c.CreateTranscript(ctx, audio)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.ToLower(transcript)
	if !strings.Contains(got, "hello") || !strings.Contains(got, "world") {
		t.Errorf("Expected 'hello world', got %s", got)
	}

}

func newTestOracle(t *testing.T) *goracle.Oracle {
	t.Helper()
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		t.Fatal("OPENAI_API_KEY is not set")
	}
	c := client.NewChatGPT(token)
	return goracle.NewOracle(c)
}

func newVertexTestOracle(t *testing.T) *goracle.Oracle {
	t.Helper()
	token := os.Getenv("VERTEX_API_KEY")
	if token == "" {
		t.Fatal("VERTEX_API_KEY is not set")
	}
	project := os.Getenv("VERTEX_PROJECT")
	if project == "" {
		t.Fatal("VERTEX_PROJECT is not set")
	}
	c := client.NewVertex()
	return goracle.NewOracle(c)
}
