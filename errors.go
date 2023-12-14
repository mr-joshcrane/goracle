package oracle

import "github.com/mr-joshcrane/oracle/client/openai"

type ClientError struct {
	openai.ClientError
}

type RateLimitError struct {
	openai.RateLimitError
}

type BadRequestError struct {
	openai.BadRequestError
}
