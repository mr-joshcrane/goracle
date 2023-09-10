package client

import (
	"errors"
	"net/http"
	"strconv"
	"time"
)

type ClientError struct {
	err           error
	statusCode    int
	retryInterval time.Duration
}

type RateLimit struct {
	ResetReq          time.Duration
	ResetTkns         time.Duration
	TokensRemaining   int
	RequestsRemaining int
}

func ParseRateLimit(resp http.Response) RateLimit {
	var r RateLimit
	r.ResetReq, _ = time.ParseDuration(resp.Header.Get("X-RateLimit-Reset-Requests"))
	r.ResetTkns, _ = time.ParseDuration(resp.Header.Get("X-RateLimit-Reset-Token"))
	remainingRequests := resp.Header.Get("X-RateLimit-Remaining-Requests")
	r.TokensRemaining, _ = strconv.Atoi(remainingRequests)
	remainingTokens := resp.Header.Get("X-RateLimit-Remaining-Tokens")
	r.TokensRemaining, _ = strconv.Atoi(remainingTokens)
	return r
}

func (c ClientError) Error() string {
	if c.err == nil {
		return ""
	}
	return c.err.Error()
}

func (c ClientError) RetryIn() time.Duration {
	return c.retryInterval
}

func (c ClientError) StatusCode() int {
	return c.statusCode
}

func ErrorRateLimitExceeded(r http.Response) ClientError {
	rateLimit := ParseRateLimit(r)
	var retryIn time.Duration
	if rateLimit.TokensRemaining == 0 {
		retryIn = rateLimit.ResetTkns
	} else if rateLimit.RequestsRemaining == 0 {
		retryIn = rateLimit.ResetReq
	}
	return ClientError{
		err:           errors.New("rate limit exceeded"),
		statusCode:    429,
		retryInterval: retryIn,
	}
}

func ErrorBadRequest() ClientError {
	return ClientError{
		err:        errors.New("bad request"),
		statusCode: 400,
	}
}

func ErrorUnauthorized() ClientError {
	return ClientError{
		err:        errors.New("unauthorized"),
		statusCode: 401,
	}
}

func GenericError(err error) ClientError {
	return ClientError{
		err:        err,
		statusCode: 500,
	}
}
