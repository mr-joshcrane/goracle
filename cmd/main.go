package main

import (
	"context"
	"fmt"
	"image"
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
	f1 := image.NewRGBA(image.Rect(0, 0, 100, 100))
	vis := oracle.NewVisuals(f1)
	answer, err := o.Ask(context.Background(), "What is this?", vis)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(answer)
}
