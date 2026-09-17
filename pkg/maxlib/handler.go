package maxlib

import "github.com/max-messenger/max-bot-api-client-go/v2/model"

type HandlerFunc func(upd model.Update) error

func (h HandlerFunc) HandleUpdate(upd model.Update) error {
	return h(upd)
}

func (h HandlerFunc) Use(mws ...Middleware) {
	panic("HandlerFunc.Use must never be called")
}
