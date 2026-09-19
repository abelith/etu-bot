package core

import (
	"errors"
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	merrors "github.com/abelith/etu-bot/pkg/maxlib/errors"
)

type Middleware func(next UpdateHandler) UpdateHandler

type Filter func(c *mcontext.Context) bool

type route struct {
	h       UpdateHandler
	filters []Filter
}

type Router struct {
	routes []route
	mw     []Middleware
}

func (r *Router) HandleUpdate(c *mcontext.Context) error {
	var current UpdateHandler = HandlerFunc(r.dispatch)

	for i := len(r.mw) - 1; i >= 0; i-- {
		current = r.mw[i](current)
	}

	return current.HandleUpdate(c)
}

func (r *Router) Use(mws ...Middleware) {
	r.mw = append(r.mw, mws...)
}

func (r *Router) HandleFunc(h HandlerFunc, filters ...Filter) {
	r.routes = append(r.routes, route{
		h:       h,
		filters: filters,
	})
}

func (r *Router) Handle(h UpdateHandler, filters ...Filter) {
	r.routes = append(r.routes, route{
		h:       h,
		filters: filters,
	})
}

func (r *Router) dispatch(c *mcontext.Context) error {
	for _, rt := range r.routes {
		if match(rt.filters, c) {
			err := rt.h.HandleUpdate(c)
			if errors.Is(err, merrors.ErrRouteNotFound) {
				continue
			}
			return err
		}
	}

	return merrors.ErrRouteNotFound
}

func match(filters []Filter, c *mcontext.Context) bool {
	for _, filter := range filters {
		if !filter(c) {
			return false
		}
	}

	return true
}
