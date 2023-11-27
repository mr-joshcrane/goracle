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
	//f1 := strings.NewReader("I love tomatos")
	//f2 := strings.NewReader("I am a big fan of tomatos")
	//f3 := strings.NewReader("I hate tomatoes")
	f1, _ := os.Open("file1.txt")
	f2, _ := os.Open("file2.txt")
	f3, _ := os.Open("file3.txt")

	docs := oracle.NewDocuments(f1, f2, f3)

	answer, err := o.Ask(context.Background(), "I recieved letters. What do they say?", docs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(answer)
}
