package oracle_test

import (
	"testing"

	"github.com/mr-joshcrane/oracle"
)

func TestAsk(t *testing.T) {
	question := "What is the meaning of life?"
	oracle := oracle.NewOracle()
	answer := oracle.Ask(question)
	if answer != "42" {
		t.Errorf("Expected 42, got %s", answer)
	}
}
