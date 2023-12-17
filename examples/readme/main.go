package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/mr-joshcrane/goracle"
	"github.com/mr-joshcrane/goracle/client"
)

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go iterateOnReadme(i, &wg)
	}
	wg.Wait()
}

func iterateOnReadme(i int, wg *sync.WaitGroup) {
	ctx := context.Background()
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		fmt.Println("OPENAI_API_KEY is not set")
		return
	}
	// --- Generate
	c := client.NewChatGPT(token)
	// c := client.VertexAI() // We could easily swap out the client here

	generator := goracle.NewOracle(c)

	response, err := generator.Ask(ctx,
		` I am editing my README.md file. I want to make it better, and I included the source code so you can read it.
	  	Rewrite this README.md file. I will be piping it directly to a file, so please write it in a suitable style.
		  Here are my requirements:
			  Explain that this is only intended to be a user friendly convenience library for working with LLMs in Golang applications.
		    Have a section on the Ask method does and how you can use it.
			  Have a section on what a Reference is conceptually, the different types supported, and how you can use it.
		`,
		goracle.File("README.md"),
		goracle.File("goracle.go"),
		goracle.File("oracle_test.go"),
		goracle.File("client/client_integration_test.go"),
	)
	if err != nil {
		fmt.Println("Error:", err)
		wg.Done()
		return
	}
	err = os.WriteFile(fmt.Sprintf("README%d.md", i), []byte(response), 0644)
	if err != nil {
		fmt.Println("Error:", err)
	}
	wg.Done()
}
