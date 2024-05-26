package errors

import "errors"

var (
	EmptyQueue = errors.New("queue is empty")
)
