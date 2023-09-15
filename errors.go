package oracle

import (
	"github.com/mr-joshcrane/oracle/client"
)

type ClientError struct {
	client.ClientError
}

type RateLimitError struct {
	client.RateLimitError
}

type BadRequestError struct {
	client.BadRequestError
}
