//go:build integration

package oracle_test

import (
	"os"
	"strings"
	"testing"

	"github.com/mr-joshcrane/oracle"
)

func TestGiveExample(t *testing.T) {
	t.Parallel()
	o := newTestOracle(t)
	prompt := oracle.Prompt{
		Purpose: "To answer if a number is odd or even in a specific format",
		Examples: map[string]string{
			"2": "+++even+++",
			"3": "---odd---",
			"4": "+++even+++",
			"5": "---odd---",
		},
		Question: "6",
	}
	answer, err := o.Completion(prompt)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}

	if answer != "+++even+++" {
		t.Errorf("Expected +++even+++, got %s", answer)
	}
}

func TestAsk(t *testing.T) {
	t.Parallel()
	o := newTestOracle(t)
	o.SetPurpose("You always answer questions with the number 42.")
	question := "What is the meaning of life?"
	answer, err := o.Ask(question)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	if !strings.Contains(answer, "42") {
		t.Errorf("Expected 42, got %s", answer)
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
