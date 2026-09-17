package maxlib

import "slices"

func Chain(h UpdateHandler, mws ...Middleware) UpdateHandler {
	slices.Reverse(mws)
	for _, mw := range mws {
		h = mw(h)
	}
	return h
}
