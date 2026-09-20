package errors

import "fmt"

var (
	ErrInvalidToken = fmt.Errorf("invalid org token")
)
