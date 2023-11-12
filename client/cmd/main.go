package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mr-joshcrane/oracle/client"
)

func main() {
	prompt := strings.Join(os.Args[1:], " ")
	token := os.Getenv("OPENAI_API_KEY")
	image, err := client.GenerateImage(token, prompt)
	if err != nil {
		panic(err)
	}
	fmt.Println(image)
}
