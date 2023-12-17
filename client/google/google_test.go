package google_test

import (
	"github.com/mr-joshcrane/oracle"
	"github.com/mr-joshcrane/oracle/client/google"
)

func testPrompt() oracle.Prompt {
	return oracle.Prompt{
		Purpose:       "A test purpose",
		InputHistory:  []string{"GivenInput", "GivenInput2"},
		OutputHistory: []string{"IdealOutput", "IdealOutput2"},
		Question:      "A test question",
		References:    [][]byte{[]byte("page1"), []byte("page2")},
	}
}

func testMessages() []google.ChatMessage {
	return google.MessagesFromPrompt(testPrompt())
}
