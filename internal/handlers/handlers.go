package handlers

import (
	"context"
	"errors"
	"fmt"
	errors2 "github.com/abelith/etu-bot/internal/errors"
	"github.com/abelith/etu-bot/internal/models"
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	"github.com/abelith/etu-bot/pkg/maxlib/core"
	"github.com/abelith/etu-bot/pkg/maxlib/filters"
	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"regexp"
)

const (
	stateStart1 = "StateStart1"
	stateStart2 = "StateStart2"
	stateStart3 = "StateStart3"
	stateStart4 = "StateStart4"
	stateStart5 = "StateStart5"
)

const helloMsg = `Добро пожаловать в систему умного города. Так как вы являетесь новым пользователем, заполните следующие данные`
const startMsg0 = "Как к Вам обращаться?"
const startMsg1 = "Если вы организация, а не жилец, можете отправить в чат свой ключ авторизации, подтвердив свой статус"
const startMsg2WithName = "%s, добро пожаловать в систему умного города. Отправьте свою геолокацию, воспользуйтесь госуслугами или введите адрес вручную, чтобы мы могли подключить вас к нужным системам"
const startMsg2WithoutName = "Добро пожаловать в систему умного города. Отправьте свою геолокацию, воспользуйтесь госуслугами или введите адрес вручную, чтобы мы могли подключить вас к нужным системам"
const fallbackMsg = "Сообщение не распознано. Попробуйте снова"
const leaveCurrent = "Оставить текущее"

var tokenRe = regexp.MustCompile(`^token:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type UseCases interface {
	VerifyOrganization(ctx context.Context, token string) (string, error)
	GetHouses(ctx context.Context, latitude, longitude float64) ([]*models.House, error)
	AddInhabitant(ctx context.Context, inhabitant *models.Inhabitant) error
	DeleteUser(ctx context.Context, id int) error
}

type Handlers struct {
	UseCases UseCases
}

func (h *Handlers) Router() *core.Router {
	rt := &core.Router{}
	rt.HandleFunc(h.StartHandler, filters.StartCommand)
	rt.HandleFunc(h.StartHandler1, filters.State(stateStart1))
	rt.HandleFunc(h.StartHandler2Organization, filters.State(stateStart2), filters.MessageTextRe(tokenRe))
	rt.HandleFunc(h.StartHandler2Inhabitant, filters.State(stateStart2), filters.MessageText("Я жилец"))
	rt.HandleFunc(h.StartHandler3, filters.State(stateStart3)) // debug stub
	rt.HandleFunc(h.StartHandler4, filters.State(stateStart4))
	rt.HandleFunc(h.StartHandler5, filters.State(stateStart5), filters.Or(filters.MessageText("Да"), filters.MessageText("Нет")))
	rt.HandleFunc(h.BotStoppedHandler, filters.BotStopped)
	rt.HandleFunc(h.FallbackHandler)
	return rt
}

func (h *Handlers) StartHandler(c *mcontext.Context) error {
	if err := c.Respond(maxbot.NewMessage().SetText(helloMsg)); err != nil {
		return err
	}

	c.State().SetState(stateStart1)

	kb := model.NewKeyboard()
	kb.AddRow().AddMessage(leaveCurrent)

	if err := c.Respond(maxbot.NewMessage().SetText(startMsg0).AddKeyboard(kb)); err != nil {
		return err
	}

	return nil
}

func (h *Handlers) StartHandler1(c *mcontext.Context) error {
	txt := c.Update().Message.Body.Text
	if txt != leaveCurrent {
		c.State().Set("name", txt)
	} else {
		name := c.Update().User.Name
		c.State().Set("name", name)
	}

	name, _ := c.State().Get("name")
	ok, err := ValidateName(name)
	if err != nil {
		return err
	}
	if !ok {
		return c.Respond(maxbot.NewMessage().SetText("Пожалуйста, укажите корректное имя"))
	}

	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Я жилец")
	c.State().SetState(stateStart2)
	return c.Respond(maxbot.NewMessage().SetText(startMsg1).AddKeyboard(kb))
}

func (h *Handlers) StartHandler2Organization(c *mcontext.Context) error {
	txt := c.Update().Message.Body.Text
	org, err := h.UseCases.VerifyOrganization(c.Context(), txt)
	if err != nil {
		if errors.Is(err, errors2.ErrInvalidToken) {
			kb := model.NewKeyboard()
			kb.AddRow().AddMessage("Я жилец")
			msg := maxbot.NewMessage().
				SetText("Ваш ключ недействителен. Вы можете попробовать снова или воспользоваться системой как жилец").
				AddKeyboard(kb)
			return c.Respond(msg)
		}

		return err
	}
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Открыть меню управления")
	msg := maxbot.NewMessage().
		SetText(fmt.Sprintf("Добро пожаловать в систему, %s", org)).
		AddKeyboard(kb)
	c.State().Clear()
	return c.Respond(msg)
}

func (h *Handlers) StartHandler2Inhabitant(c *mcontext.Context) error {
	c.State().SetState(stateStart3)
	kb := model.NewKeyboard()
	kb.AddRow().AddGeoLocation("Мой адрес", false)
	kb.AddRow().AddMessage("Адрес из госуслуг")
	msg := maxbot.NewMessage()
	msg.AddKeyboard(kb)

	name, ok := c.State().Get("name")
	if !ok {
		msg.SetText(startMsg2WithoutName)
		return c.Respond(msg)
	}

	msg.SetText(fmt.Sprintf(startMsg2WithName, name))
	return c.Respond(msg)
}

func (h *Handlers) StartHandler3(c *mcontext.Context) error {
	if txt := c.Update().Message; txt != nil && txt.Body.Text == "Адрес из госуслуг" {
		msg := maxbot.NewMessage().SetText("WIP: Будет добавлено позже")
		return c.Respond(msg)
	}

	var latitude float64
	var longitude float64

	for _, attachment := range c.Update().Message.Body.Attachments {
		if attachment.Type == model.AttachLocation {
			latitude = attachment.Latitude
			longitude = attachment.Longitude
		}
	}

	if latitude == 0 && longitude == 0 {
		// todo: распарсить текстовый адрес, получить из него координаты
	}

	c.State().Set("latitude", fmt.Sprint(latitude))
	c.State().Set("longitude", fmt.Sprint(longitude))

	houses, err := h.UseCases.GetHouses(c, latitude, longitude)
	if err != nil {
		return err
	}

	kb := model.NewKeyboard()
	for _, house := range houses {
		kb.AddRow().AddMessage(house.Address)
	}

	msg := maxbot.NewMessage().
		SetText("Выберите свой дом").
		AddKeyboard(kb)

	c.State().SetState(stateStart4)
	return c.Respond(msg)
}

func (h *Handlers) StartHandler4(c *mcontext.Context) error {
	house := c.Update().Message.Body.Text
	c.State().Set("house", house)

	name, ok := c.State().Get("name")
	if !ok {
		return fmt.Errorf("no name in state")
	}

	txt := "Вас зовут: %s\nВаш адрес: %s\nВсё верно?"
	kb := model.NewKeyboard()
	kb.AddRow().
		AddMessage("Да").
		AddMessage("Нет")
	msg := maxbot.NewMessage().
		SetText(fmt.Sprintf(txt, name, house)).
		AddKeyboard(kb)

	c.State().SetState(stateStart5)
	return c.Respond(msg)
}

func (h *Handlers) StartHandler5(c *mcontext.Context) error {
	msg := maxbot.NewMessage()
	switch c.Update().Message.Body.Text {
	case "Да":
		state := c.State()
		name, ok := state.Get("name")
		if !ok {
			return fmt.Errorf("name not in state")
		}

		house, ok := state.Get("house")
		if !ok {
			return fmt.Errorf("houst not in state")
		}

		inhabitant := &models.Inhabitant{
			Id:           int(c.Update().User.UserID),
			Name:         name,
			HouseAddress: house,
		}

		if err := h.UseCases.AddInhabitant(c, inhabitant); err != nil {
			return err
		}

		msg.SetText("Вы подключены к системе. Поздравляем")
		state.Clear()
		return c.Respond(msg)
	case "Нет":
		msg.SetText("Введите данные повторно. Как к вам обращаться?")
		c.State().SetState(stateStart1)
		return c.Respond(msg)
	}

	return nil
}

func (h *Handlers) BotStoppedHandler(c *mcontext.Context) error {
	return h.UseCases.DeleteUser(c, int(c.Update().UserID))
}

func (h *Handlers) FallbackHandler(c *mcontext.Context) error {
	return c.Respond(maxbot.NewMessage().SetText(fallbackMsg))
}
