[![Go Reference](https://pkg.go.dev/badge/github.com/mr-joshcrane/oracle.svg)](https://pkg.go.dev/github.com/mr-joshcrane/oracle)[![License: MIT](https://img.shields.io/badge/Licence-MIT)](https://opensource.org/licenses/MIT)[![Go Report Card](https://goreportcard.com/badge/github.com/mr-joshcrane/oracle)](https://goreportcard.com/report/github.com/mr-joshcrane/oracle)


# Oracle

Oracle is a Go library that provides an easy way to integrate OpenAI's GPT-3 into your application. Whether you need to generate text, answer questions, create conversational agents, or more, Oracle has got you covered.

## Features

- Simplifies interaction with the GPT-3 API
- Supports customization of prompts
- Unifies the interaction with the different models available in GPT-3
- Methods for parsing and error handling of API responses
- Client creation with OpenAI API key

## Installation

To add Oracle to your Go project, use the `go get` command:

```sh
go get github.com/mr-joshcrane/oracle
```

## Example Usage

Create a new Oracle client and ask it a question:

```go
package main

import (
    "fmt"
    "github.com/mr-joshcrane/oracle"
)

func main() {
    question := "How much wood would a woodchuck chuck if a woodchuck could chuck wood?"
    o := oracle.NewOracle()
    answer, err := o.Ask(question)
    if err != nil {
        panic(err)
    }
    fmt.Println(answer)
}
```

The Ask method will generate a proper chatlog prompt with prior messages and send it off to the GPT-3 API for producing an answer to the question. The library abstracts away the heavy lifting, enabling you to focus solely on working with the response.

## Test

To test the Oracle library, navigate to the repository's root and run:

```sh
go test ./...
```

## Contribution

Contributions to this project are welcome. Please ensure that the test suite passes before submitting your PR.

## License

Oracle is MIT licensed, as found in the [LICENSE](https://github.com/mr-joshcrane/oracle/blob/main/LICENSE) file.
