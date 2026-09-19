package errors

import "fmt"

var (
	ErrRouteNotFound = fmt.Errorf("route not found")
)
