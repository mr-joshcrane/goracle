package client

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

type ClientError struct {
	Status     string
	StatusCode int
}

func (e ClientError) Error() string {
	return fmt.Sprintf("Client error: %s", e.Status)
}

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e RateLimitError) Error() string {
	return fmt.Sprintf("Rate limit exceeded. Retry after %s", e.RetryAfter)
}

type BadRequestError struct {
	RequestTokens int
	TokenLimit    int
}

func (e BadRequestError) Error() string {
	return fmt.Sprintf("Bad request. Requested %d tokens, limit is %d", e.RequestTokens, e.TokenLimit)
}

type rateLimit struct {
	ResetReq          time.Duration
	ResetTkns         time.Duration
	TokensRemaining   int
	RequestsRemaining int
}

func parseRateLimit(resp http.Response) rateLimit {
	var r rateLimit
	return r
}

func errorRateLimitExceeded(r http.Response) error {
	rateLimit := parseRateLimit(r)
	var retryIn time.Duration
	if rateLimit.TokensRemaining == 0 {
		retryIn = rateLimit.ResetTkns
	} else if rateLimit.RequestsRemaining == 0 {
		retryIn = rateLimit.ResetReq
	}
	rle := RateLimitError{
		RetryAfter: retryIn,
	}
	ce := ClientError{
		Status:     r.Status,
		StatusCode: http.StatusTooManyRequests,
	}
	return errors.Join(rle, ce)
}
func errorBadRequest(r http.Response) error {
	brqe := BadRequestError{
		RequestTokens: 0,
		TokenLimit:    0,
	}
	ce := ClientError{
		Status:     r.Status,
		StatusCode: http.StatusBadRequest,
	}
	return errors.Join(brqe, ce)
}

func NewClientError(r *http.Response) error {
	if r.StatusCode == http.StatusBadRequest {
		return errorBadRequest(*r)
	}
	if r.StatusCode == http.StatusTooManyRequests {
		return errorRateLimitExceeded(*r)
	}
	return fmt.Errorf("%w", ClientError{
		Status:     r.Status,
		StatusCode: r.StatusCode,
	})
}
