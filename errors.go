package goracle

import "github.com/mr-joshcrane/goracle/client/openai"

type ClientError struct {
	openai.ClientError
}

type RateLimitError struct {
	openai.RateLimitError
}

type BadRequestError struct {
	openai.BadRequestError
}
