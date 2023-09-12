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
	question := "How much wood would a woodchuck chuck?"
	o := oracle.NewOracle(token)
	q, err := o.Ask(context.Background(), question)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Fprintln(os.Stdout, q)
}
