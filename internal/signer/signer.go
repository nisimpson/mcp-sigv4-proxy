package signer

import (
	"context"
	"net/http"
)

// Signer signs HTTP requests with AWS credentials
type Signer interface {
	// SignRequest adds AWS signature headers to the request
	SignRequest(ctx context.Context, req *http.Request, payloadHash string) error
}
