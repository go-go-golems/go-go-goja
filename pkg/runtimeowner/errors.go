package runtimeowner

import "errors"

var (
	ErrClosed           = errors.New("runtime owner closed")
	ErrScheduleRejected = errors.New("runtime schedule rejected")
	ErrCanceled         = errors.New("runtime call canceled")
	ErrPanicked         = errors.New("runtime call panicked")
)
