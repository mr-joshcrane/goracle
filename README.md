```markdown
# Oracle - Advanced Go Library for LLM Abstraction

## Introduction
Oracle is a Go library designed to streamline interactions with large language models (LLMs) by providing a unified API layer for different services.

Supported clients:
- OpenAI's GPT
- Google's VertexAI models (including Gemini)

## Core Features
- Unified API layer for LLMs.
- Consistent interactions with OpenAI and VertexAI.

## Supported Clients
- Modular design for scalability.
- The library defines a 'Prompt' struct that allows for versatile request generation, supporting text and image data.

## Error Handling
- Handles common errors and service-specific issues such as rate limiting and invalid requests.

## Prerequisites
- Installation of Go.
- Required OpenAI and VertexAI API keys and VertexAI project ID.

## Installation and Setup

Install the library:

```sh
go get github.com/mr-joshcrane/oracle
```

Configure the required environment variables:

For OpenAI:

```sh
export OPENAI_API_KEY='your_api_key_here'
```

For Google Cloud, be sure to have set up your authentication with the `gcloud`
CLI. If you can get a valid token with the below command, you should be in
business.

```sh
gcloud auth --print-access-token
```

## Usage

Initialize Oracle with OpenAI's GPT:

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/mr-joshcrane/oracle"
)

func main() {
    ctx := context.Background()
    openaiToken := os.Getenv("OPENAI_API_KEY")
    c := oracle.NewOracle(openaiToken)
    response, err := oracleGPT.Ask(ctx, "Sample query for GPT.")
    if err != nil {
        fmt.Println("Error:", err)
        return
    }

    fmt.Println(response)
}
```

To switch to VertexAI, initialize with VertexAI's client configuration.

Run Tests:
The `oracle_test.go` MainTest function has been modified to run all subtests
collecting coverage information. To run them...

```sh
# For unit tests only and file coverage
go test 
# Include integration tests
go test --integration
```

## Contributing
We appreciate contributions. To contribute:
- Include tests for new features.
- Pass all the tests before submitting a PR.
- Clearly describe changes in your PR.
Feedback, especially on documentation, is highly valued.

## Licensing and Support
The project is under the MIT License. For support, submit inquiries through our [issues tracker](https://github.com/mr-joshcrane/oracle/issues), providing detailed context for quicker resolution.

*Oracle is third-party software and not officially affiliated with OpenAI or Google Cloud.*
```
This README removes overly descriptive language aligning with critique reference 1 and corrects inaccuracies and clarifies ambiguity as per points laid out in critique reference 3.
