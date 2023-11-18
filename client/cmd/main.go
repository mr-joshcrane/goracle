package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mr-joshcrane/oracle/client"
)

//func main() {
//	prompt := strings.Join(os.Args[1:], " ")
//	token := os.Getenv("OPENAI_API_KEY")
//	image, err := client.GenerateImage(token, prompt)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println(image)
//}

func main() {
	url := "https://images.unsplash.com/photo-1481349518771-20055b2a7b24?w=500&auto=format&fit=crop&q=60&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxzZWFyY2h8NHx8cmFuZG9tfGVufDB8fDB8fHww"
	url2 := "https://images.unsplash.com/photo-1513542789411-b6a5d4f31634?w=500&auto=format&fit=crop&q=60&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxzZWFyY2h8M3x8cmFuZG9tfGVufDB8fDB8fHww"
	msg := client.CreateVisionMessage("What is similar between these two images?", url, url2)
	token := os.Getenv("OPENAI_API_KEY")
	req, err := client.CreateVisionRequest(token, msg)
	if err != nil {
		panic(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("bad status code: %d", resp.StatusCode)
		panic(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}
