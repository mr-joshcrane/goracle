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
	RemainingRequests string
	RemainingTokens   string
	ResetRequests     time.Duration
	ResetTokens       time.Duration
}

func parseRateLimit(resp http.Response) rateLimit {
	resetRequest := resp.Header.Get("X-Ratelimit-Reset-Requests")
	parsedResetRequest, err := time.ParseDuration(resetRequest)
	if err != nil {
		return rateLimit{}
	}

	resetTokens := resp.Header.Get("X-Ratelimit-Reset-Tokens")
	parsedResetTokens, err := time.ParseDuration(resetTokens)
	if err != nil {
		return rateLimit{}
	}
	return rateLimit{
		RemainingRequests: resp.Header.Get("X-Ratelimit-Remaining-Requests"),
		RemainingTokens:   resp.Header.Get("X-Ratelimit-Remaining-Tokens"),
		ResetRequests:     parsedResetRequest,
		ResetTokens:       parsedResetTokens,
	}
}

func ErrorRateLimitExceeded(r http.Response) error {
	rateLimit := parseRateLimit(r)
	var err RateLimitError
	if rateLimit.RemainingTokens == "0" {
		err.RetryAfter = rateLimit.ResetTokens
	} else if rateLimit.RemainingRequests == "0" {
		err.RetryAfter = rateLimit.ResetRequests
	}
	return errors.Join(ClientError{
		Status:     r.Status,
		StatusCode: http.StatusTooManyRequests,
	}, err)
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
		return ErrorRateLimitExceeded(*r)
	}
	return fmt.Errorf("%w", ClientError{
		Status:     r.Status,
		StatusCode: r.StatusCode,
	})
}
