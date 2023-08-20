package oracle_test

import (
	"testing"

	"github.com/mr-joshcrane/oracle"
)

func TestAsk(t *testing.T) {
	t.Parallel()
	oracle := oracle.NewOracle()
	oracle.SetPurpose("You always answer questions with the number 42.")
	question := "What is the meaning of life?"
	answer, err := oracle.Ask(question)
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}
	if answer != "42" {
		t.Errorf("Expected 42, got %s", answer)
	}
}

func TestGiveExample(t *testing.T) {
	t.Parallel()
	oracle := oracle.NewOracle()
	oracle.SetPurpose("To answer if a number is odd or even in a specific format")
	oracle.GiveExamplePrompt("2", "+++even+++")
	oracle.GiveExamplePrompt("3", "---odd---")
	oracle.GiveExamplePrompt("4", "+++even+++")
	oracle.GiveExamplePrompt("5", "---odd---")
	answer, err := oracle.Ask("6")
	if err != nil {
		t.Errorf("Error asking question: %s", err)
	}

	if answer != "+++even+++" {
		t.Errorf("Expected +++even+++, got %s", answer)
	}
}
