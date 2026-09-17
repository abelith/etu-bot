package maxlib

import (
	"errors"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
)

type Middleware func(next UpdateHandler) UpdateHandler

type Filter func(upd model.Update) bool

type route struct {
	h       UpdateHandler
	filters []Filter
}

type Router struct {
	routes []route
	mw     []Middleware
}

func (r *Router) HandleUpdate(upd model.Update) error {
	var current UpdateHandler = HandlerFunc(r.dispatch)

	for i := len(r.mw) - 1; i >= 0; i-- {
		current = r.mw[i](current)
	}

	return current.HandleUpdate(upd)
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

func (r *Router) dispatch(upd model.Update) error {
	for _, rt := range r.routes {
		if match(rt.filters, upd) {
			err := rt.h.HandleUpdate(upd)
			if errors.Is(err, ErrRouteNotFound) {
				continue
			}
			return err
		}
	}

	return ErrRouteNotFound
}

func match(filters []Filter, upd model.Update) bool {
	for _, filter := range filters {
		if !filter(upd) {
			return false
		}
	}

	return true
}
