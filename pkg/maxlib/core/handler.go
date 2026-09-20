package core

import mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"

type HandlerFunc func(c *mcontext.Context) error

func (h HandlerFunc) HandleUpdate(c *mcontext.Context) error {
	return h(c)
}

func (h HandlerFunc) Use(_ ...Middleware) {
	panic("HandlerFunc.Use must never be called")
}
