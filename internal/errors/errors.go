package errors

import "fmt"

var (
	ErrInvalidToken   = fmt.Errorf("invalid org token")
	ErrNoNearbyHouses = fmt.Errorf("nearby houses are not found")
)
