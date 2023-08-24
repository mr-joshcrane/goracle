package oracle_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/oracle"
)

func TestGeneratePrompt_GeneratesExpectedPrompt(t *testing.T) {
	t.Parallel()
	o := oracle.NewOracle("dummy-token-openai")
	o.SetPurpose("To answer if a number is odd or even in a specific format")
	o.GiveExample("2", "+++even+++")
	o.GiveExample("3", "---odd---")
	got := o.GeneratePrompt("4")
	want := oracle.Prompt{
		Purpose: "To answer if a number is odd or even in a specific format",
		ExampleInputs: []string{
			"2",
			"3",
		},
		IdealOutputs: []string{
			"+++even+++",
			"---odd---",
		},
		Question: "4",
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}
