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
	ctx := context.Background()

	answer, err := o.Ask(ctx, "What are my favourite beans?",
		oracle.Folder("./cmd/beans/"),
	)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(answer)
}
