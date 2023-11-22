package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mr-joshcrane/oracle"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY is not set")
		os.Exit(1)
	}
	o := oracle.NewOracle(token)
	answer, err := o.Ask(context.Background(), "What is the meaning of life?")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(answer)
}
