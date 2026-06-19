package api

import (
	"errors"
	"net/url"
)

// CreateBatchRequest represents the incoming JSON for a new batch.
type CreateBatchRequest struct {
	URLs    []string `json:"urls"`
	Options *Options `json:"options,omitempty"`
}

type Options struct {
	Concurrency int `json:"concurrency,omitempty"`
	TimeoutMs   int `json:"timeout_ms,omitempty"`
}

// ErrorResponse matches the exact error contract from the exam.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Validate applies the exam's strict business rules.
func (req *CreateBatchRequest) Validate() error {
	if len(req.URLs) == 0 || len(req.URLs) > 100 {
		return errors.New("urls must contain between 1 and 100 entries")
	}

	for _, u := range req.URLs {
		parsed, err := url.ParseRequestURI(u)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return errors.New("invalid url format: must be http or https")
		}
	}

	// Apply defaults and bounds
	if req.Options == nil {
		req.Options = &Options{}
	}

	if req.Options.Concurrency == 0 {
		req.Options.Concurrency = 8
	} else if req.Options.Concurrency < 1 || req.Options.Concurrency > 50 {
		return errors.New("concurrency must be between 1 and 50")
	}

	if req.Options.TimeoutMs == 0 {
		req.Options.TimeoutMs = 5000
	} else if req.Options.TimeoutMs < 100 || req.Options.TimeoutMs > 30000 {
		return errors.New("timeout_ms must be between 100 and 30000")
	}

	return nil
}
