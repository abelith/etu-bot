package handlers

import (
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	"github.com/abelith/etu-bot/pkg/maxlib/core"
	"github.com/abelith/etu-bot/pkg/maxlib/filters"
	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
)

type UserMenuUseCases interface{}

type UserMenuHandlers struct {
	UseCases UserMenuUseCases
}

func (umh *UserMenuHandlers) Router() *core.Router {
	rt := &core.Router{}
	rt.HandleFunc(umh.MenuHandler, filters.MessageText("Меню"))
	rt.HandleFunc(umh.MeHandler, filters.MessageText("Мои данные"))
	rt.HandleFunc(umh.NotificationsSettingsHandler, filters.MessageText("Настройки уведомлений"))
	rt.HandleFunc(umh.NotifyHandler, filters.MessageText("Создать объявление"))
	rt.HandleFunc(umh.RequestHandler, filters.MessageText("Создать заявку"))
	rt.HandleFunc(umh.AnalyticsHandler, filters.MessageText("Просмотреть аналитику"))
	return rt
}

func (umh *UserMenuHandlers) MenuHandler(c *mcontext.Context) error {
	msg := maxbot.NewMessage().AddKeyboard(inhabitantMenuKb)
	return c.Respond(msg)
}

func (umh *UserMenuHandlers) MeHandler(c *mcontext.Context) error {
	panic("unimplemented")
}

func (umh *UserMenuHandlers) NotificationsSettingsHandler(c *mcontext.Context) error {
	panic("unimplemented")
}

func (umh *UserMenuHandlers) NotifyHandler(c *mcontext.Context) error {
	panic("unimplemented")
}

func (umh *UserMenuHandlers) RequestHandler(c *mcontext.Context) error {
	panic("unimplemented")
}

func (umh *UserMenuHandlers) AnalyticsHandler(c *mcontext.Context) error {
	panic("unimplemented")
}
