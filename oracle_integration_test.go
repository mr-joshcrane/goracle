//go:build integration

package oracle_test

import (
	"context"
	"image/jpeg"
	"os"

	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/oracle"
	"github.com/mr-joshcrane/oracle/client"
)

type testCase struct {
	Oracle      *oracle.Oracle
	Description string
}

func testCases(t *testing.T) []testCase {
	return []testCase{
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
			got, err := o.Ask(context.Background(), "6")
			if err != nil {
				t.Errorf("Error asking question: %s", err)
			}

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
			answer, err := o.Ask(context.TODO(), question)
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

			answer, err := o.Ask(context.TODO(), "Can you repeat my two facts?",
				"the sky is blue",
				oracle.File(f.Name()),
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
	t.Parallel()
	cases := testCases(t)
	gopher, err := os.Open("testdata/quokka.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer gopher.Close()
	image, err := jpeg.Decode(gopher)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			o := c.Oracle
			answer, err := o.Ask(context.TODO(), "What species is this?",
				image,
			)
			if err != nil {
				t.Errorf("Error asking question: %s", err)
			}
			answer = strings.ToLower(answer)
			if !strings.Contains(answer, "quokka") {
				t.Errorf("Error reading from image, expected 'gopher', got %s", answer)
			}
		})
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

func newVertexTestOracle(t *testing.T) *oracle.Oracle {
	t.Helper()
	token := os.Getenv("VERTEX_API_KEY")
	if token == "" {
		t.Fatal("VERTEX_API_KEY is not set")
	}
	project := os.Getenv("VERTEX_PROJECT")
	if project == "" {
		t.Fatal("VERTEX_PROJECT is not set")
	}
	c := client.NewVertex(token, project)
	return oracle.NewOracle("", oracle.WithClient(c))
}
