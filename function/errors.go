package webhook

import "errors"

// Video processing errors
var (
	ErrInvalidEntry      = errors.New("invalid entry")
	ErrMissingVideoID    = errors.New("missing video ID")
	ErrMissingChannelID  = errors.New("missing channel ID")
)