package handlers

import (
	"context"
	"fmt"
	"github.com/abelith/etu-bot/internal/models"
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	"github.com/abelith/etu-bot/pkg/maxlib/core"
	"github.com/abelith/etu-bot/pkg/maxlib/filters"
	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"strconv"
	"strings"
)

const (
	stateMe1          = "StateMe1"
	stateMeChangeName = "StateMeChangeName"
)

const (
	stateCreateCluster1             = "StateCreateCluster1"
	stateCreateCluster2             = "StateCreateCluster2"
	stateCreateClusterSendAddresses = "StateCreateClusterSA"
	stateSendingGeo                 = "StateSendingGeo"
	stateSendingGeo1                = "StateSendingGeo1"
)

const (
	stateNotify1 = "StateNotify1"
	stateNotify2 = "StateNotify2"
	stateNotify3 = "StateNotify3"
	stateNotify4 = "StateNotify4"
	stateNotify5 = "StateNotify5"
	stateNotify6 = "StateNotify6"
)

type OrgMenuUseCases interface {
	GetUserOrg(ctx context.Context, id int) (*models.OrgMemberMe, error)
	ChangeUserName(ctx context.Context, id int, name string) error
	ActivateUser(ctx context.Context, id int) error
	DeactivateUser(ctx context.Context, id int) error
	ParseAddress(ctx context.Context, addr string) (lat float64, long float64, err error)
	CreateCluster(ctx context.Context, name string, coords []models.Coordinates) error
	GetClusterNames(ctx context.Context, userID int) ([]string, error)
	Notify(ctx context.Context, n *models.Notification, content string) error
}

type OrgMenuHandlers struct {
	UseCases OrgMenuUseCases
}

func (omh *OrgMenuHandlers) Router() *core.Router {
	rt := &core.Router{}
	rt.HandleFunc(omh.MenuHandler, filters.MessageText("Меню"))
	rt.HandleFunc(omh.MeHandler, filters.MessageText("Мои данные"))
	rt.HandleFunc(omh.NotifyHandler, filters.MessageText("Создать объявление"))
	rt.HandleFunc(omh.AnalyticsHandler, filters.MessageText("Просмотреть аналитику"))
	rt.HandleFunc(omh.RequestsHandlers, filters.MessageText("Просмотреть заявки"))
	rt.HandleFunc(omh.meChangeNameHandler, filters.State(stateMe1), filters.MessageText("Изменить имя"))
	rt.HandleFunc(omh.meActivateHandler, filters.State(stateMe1), filters.MessageText("Активировать"))
	rt.HandleFunc(omh.meDeactivateHandler, filters.State(stateMe1), filters.MessageText("Деактивировать"))
	rt.HandleFunc(omh.meCreateClusterHandler, filters.State(stateMe1), filters.MessageText("Создать кластер"))
	rt.HandleFunc(omh.meChangeNameHandler1, filters.State(stateMeChangeName))
	rt.HandleFunc(omh.meCreateClusterHandler1, filters.State(stateCreateCluster1))
	rt.HandleFunc(omh.createClusterReceiveAddresses, filters.State(stateCreateCluster2), filters.MessageText("Отправить адреса"))
	rt.HandleFunc(omh.receiveAddressesHandler, filters.State(stateCreateClusterSendAddresses))
	rt.HandleFunc(omh.receiveAddrFileHandler, filters.State(stateCreateCluster2), filters.MessageText("Отправить файл с адресами"))
	rt.HandleFunc(omh.sendGeoHandler, filters.State(stateSendingGeo), filters.MessageText("Отправить геолокацию"))
	rt.HandleFunc(omh.receiveGeoHandler, filters.State(stateSendingGeo1), filters.MessageText("Геолокация дома"), filters.MessageText("Закончить"))
	return rt
}

func (omh *OrgMenuHandlers) MenuHandler(c *mcontext.Context) error {
	msg := maxbot.NewMessage().
		SetText("Меню").
		AddKeyboard(orgMenuKb)
	return c.Respond(msg)
}

func (omh *OrgMenuHandlers) MeHandler(c *mcontext.Context) error {
	userID := int(c.Update().UserID)
	me, err := omh.UseCases.GetUserOrg(c, userID)
	if err != nil {
		return err
	}

	status := "Активен"
	if !me.OrgMember.User.Active {
		status = "Не активен"
	}

	txt := fmt.Sprintf("Информация о Вас:\nИмя: %s\nСтатус: %s\nРоль в организации: %s\n"+
		"Информация об организации:\nНазвание: %s\nОписание: %s\n"+
		"Информация о кластере:\nНазвание: %s\nКоличество Домов: %d\n",
		me.OrgMember.User.Name, status, me.OrgMember.Role, me.Org.Name, me.Org.Description, me.ClusterStat.Name, me.ClusterStat.HousesNumber)
	msg := maxbot.NewMessage().
		SetText(txt).
		AddKeyboard(orgMeKb)
	c.State().SetState(stateMe1)
	return c.Respond(msg)
}

func (omh *OrgMenuHandlers) NotifyHandler(c *mcontext.Context) error {
	c.State().SetState(stateNotify1)
	return c.Respond(maxbot.NewMessage().SetText("Создать обращение к жильцам"))
}

func (omh *OrgMenuHandlers) notifyHandler1(c *mcontext.Context) error {
	clusterNames, err := omh.UseCases.GetClusterNames(c, int(c.Update().UserID))
	if err != nil {
		return err
	}

	clustersKb := model.NewKeyboard()
	for _, clusterName := range clusterNames {
		clustersKb.AddRow().AddMessage(clusterName)
	}

	c.State().SetState(stateNotify2)
	return c.Respond(maxbot.NewMessage().SetText("Выберите кластер").AddKeyboard(clustersKb))
}
func (omh *OrgMenuHandlers) notifyHandler2(c *mcontext.Context) error {
	m := c.Update().Message
	if m == nil {
		return fmt.Errorf("empty message")
	}

	c.State().Set("notification_cluster", m.Body.Text)
	c.State().SetState(stateNotify3)
	return c.Respond(maxbot.NewMessage().SetText("Введите текст обращения"))
}

func (omh *OrgMenuHandlers) notifyHandler3(c *mcontext.Context) error {
	m := c.Update().Message
	if m == nil {
		return fmt.Errorf("empty message")
	}

	c.State().Set("notification_mid", m.Body.Mid)
	c.State().Set("notification_content", m.Body.Text)
	c.State().SetState(stateNotify4)

	msg := maxbot.NewMessage().
		SetText("Всё верно?").
		AddKeyboard(confirmKb)
	return c.Respond(msg)
}

func (omh *OrgMenuHandlers) notifyHandler4(c *mcontext.Context) error {
	m := c.Update().Message
	if m == nil {
		return fmt.Errorf("empty message")
	}

	switch m.Body.Text {
	case "Подтвердить":
		c.State().SetState(stateNotify5)
		return c.Respond(maxbot.NewMessage().SetText("Укажите приоритет"))
	case "Отмена":
		c.State().Clear()
		return c.Respond(maxbot.NewMessage().SetText("Отменено"))
	}

	return nil
}

func (omh *OrgMenuHandlers) notifyHandler5(c *mcontext.Context) error {
	m := c.Update().Message
	if m == nil {
		return fmt.Errorf("empty message")
	}

	switch m.Body.Text {
	case "3":
		c.State().Set("notification_priority", "3")
	case "4":
		c.State().Set("notification_priority", "4")
	case "5":
		c.State().Set("notification_priority", "5")
	default:
		return c.Respond(maxbot.NewMessage().SetText("Вы указали некорректный приоритет"))
	}

	c.State().SetState(stateNotify6)
	msg := maxbot.NewMessage().
		SetText("Отправить?").
		AddKeyboard(confirmKb)
	return c.Respond(msg)
}

func (omh *OrgMenuHandlers) notifyHandler6(c *mcontext.Context) error {
	m := c.Update().Message
	if m == nil {
		return fmt.Errorf("empty message")
	}

	switch m.Body.Text {
	case "Подтвердить":
		clusterName, ok := c.State().Get("notification_cluster")
		if !ok {
			return fmt.Errorf("no clusterName")
		}

		mid, ok := c.State().Get("notification_mid")
		if !ok {
			return fmt.Errorf("no mid")
		}

		content, ok := c.State().Get("notification_content")
		if !ok {
			return fmt.Errorf("no content")
		}

		priorityStr, ok := c.State().Get("notification_priority")
		if !ok {
			return fmt.Errorf("no priority")
		}

		priority, err := strconv.ParseInt(priorityStr, 10, 32)
		if err != nil {
			return err
		}

		n := &models.Notification{
			SourceID:    mid,
			ClusterName: clusterName,
			Priority:    models.Priority(priority),
		}
		if err := omh.UseCases.Notify(c, n, content); err != nil {
			return err
		}
		return c.Respond(maxbot.NewMessage().SetText("Уведомление создано"))
	case "Отмена":
		c.State().Clear()
		return c.Respond(maxbot.NewMessage().SetText("Отменено"))
	}

	return nil
}

func (omh *OrgMenuHandlers) AnalyticsHandler(c *mcontext.Context) error {
	panic("unimplemented")
}

func (omh *OrgMenuHandlers) RequestsHandlers(c *mcontext.Context) error {
	panic("unimplemented")
}

func (omh *OrgMenuHandlers) meChangeNameHandler(c *mcontext.Context) error {
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Оставить текущее")
	msg := maxbot.NewMessage().
		SetText("Введите новое имя").
		AddKeyboard(kb)
	c.State().SetState(stateMeChangeName)
	return c.Respond(msg)
}

func (omh *OrgMenuHandlers) meChangeNameHandler1(c *mcontext.Context) error {
	m := c.Update().Message
	if m == nil {
		return fmt.Errorf("invalid message")
	}

	switch m.Body.Text {
	case "Оставить текущее":
		c.State().Clear()
		return nil
	default:
		newName := m.Body.Text
		valid, err := ValidateName(newName)
		if err != nil {
			return err
		}

		if !valid {
			return fmt.Errorf("invalid name")
		}

		if err := omh.UseCases.ChangeUserName(c, int(c.Update().UserID), newName); err != nil {
			return err
		}

		c.State().Clear()
		return c.Respond(maxbot.NewMessage().SetText("Имя изменено"))
	}
}

func (omh *OrgMenuHandlers) meActivateHandler(c *mcontext.Context) error {
	if err := omh.UseCases.ActivateUser(c, int(c.Update().UserID)); err != nil {
		return err
	}
	return c.Respond(maxbot.NewMessage().SetText("Пользователь активирован"))
}

func (omh *OrgMenuHandlers) meDeactivateHandler(c *mcontext.Context) error {
	if err := omh.UseCases.DeactivateUser(c, int(c.Update().UserID)); err != nil {
		return err
	}
	return c.Respond(maxbot.NewMessage().SetText("Пользователь деактивирован"))
}

func (omh *OrgMenuHandlers) meCreateClusterHandler(c *mcontext.Context) error {
	c.State().SetState(stateCreateCluster1)
	return c.Respond(maxbot.NewMessage().SetText("Введите название кластера"))
}

func (omh *OrgMenuHandlers) meCreateClusterHandler1(c *mcontext.Context) error {
	m := c.Update().Message
	if m == nil {
		return fmt.Errorf("empty message")
	}

	clusterName := m.Body.Text
	ok, err := ValidateName(clusterName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("invalid cluster name: %s", clusterName)
	}

	c.State().Set("cluster_name", clusterName)
	c.State().SetState(stateCreateCluster2)
	msg := maxbot.NewMessage().
		SetText("Введите адреса домов кластера").
		AddKeyboard(addClusterAddressesKb)
	return c.Respond(msg)
}

func (omh *OrgMenuHandlers) createClusterReceiveAddresses(c *mcontext.Context) error {
	c.State().SetState(stateCreateClusterSendAddresses)
	return c.Respond(maxbot.NewMessage().SetText("Отправляйте адреса"))
}

func (omh *OrgMenuHandlers) receiveAddressesHandler(c *mcontext.Context) error {
	m := c.Update().Message
	if m == nil {
		return fmt.Errorf("empty message")
	}

	if m.Body.Text == "Закончить" {
		defer c.State().Clear()
		if err := omh.flushAddresses(c); err != nil {
			return err
		}
		return c.Respond(maxbot.NewMessage().SetText("Кластер создан"))
	}

	lat, long, err := omh.UseCases.ParseAddress(c, m.Body.Text)
	if err != nil {
		// err is not handled properly, to be changed
		return c.Respond(maxbot.NewMessage().SetText("Адрес некорректный. Попробуйте снова"))
	}

	count := 0
	if s, ok := c.State().Get("addr_count"); ok {
		i, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			count = int(i)
		}
	}

	key := fmt.Sprintf("addr_%d", count)
	val := fmt.Sprintf("%f;%f", lat, long)
	c.State().Set(key, val)
	c.State().Set("addr_count", fmt.Sprint(count+1))
	return c.Respond(maxbot.NewMessage().SetText("Закончить"))
}

// total kludge, not a handler, to be reconsidered
func (omh *OrgMenuHandlers) flushAddresses(c *mcontext.Context) error {
	count := 0
	if s, ok := c.State().Get("addr_count"); ok {
		i, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			count = int(i)
		}
	}

	if count == 0 {
		return fmt.Errorf("no addresses to flush")
	}

	coords := make([]models.Coordinates, 0, count)
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("addr_%d", i)
		addr, ok := c.State().Get(key)
		if !ok {
			return fmt.Errorf("address not found")
		}
		pieces := strings.Split(addr, ";")
		if len(pieces) < 2 {
			return fmt.Errorf("invalid coordinates")
		}

		latStr := pieces[0]
		longStr := pieces[1]

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return err
		}

		long, err := strconv.ParseFloat(longStr, 64)
		if err != nil {
			return err
		}

		latLong := models.Coordinates{
			Latitude:  lat,
			Longitude: long,
		}
		coords = append(coords, latLong)
	}

	name, ok := c.State().Get("cluster_name")
	if !ok {
		return fmt.Errorf("no cluster name")
	}

	return omh.UseCases.CreateCluster(c, name, coords)
}

func (omh *OrgMenuHandlers) receiveAddrFileHandler(c *mcontext.Context) error {
	return c.Respond(maxbot.NewMessage().SetText("Будет реализовано позднее"))
}

func (omh *OrgMenuHandlers) sendGeoHandler(c *mcontext.Context) error {
	m := c.Update().Message
	if m == nil {
		return fmt.Errorf("empty message")
	}

	msg := maxbot.NewMessage().
		SetText("Отправьте геолокацию").
		AddKeyboard(geoKb)
	return c.Respond(msg)
}

func (omh *OrgMenuHandlers) receiveGeoHandler(c *mcontext.Context) error {
	m := c.Update().Message
	if m == nil {
		return fmt.Errorf("empty message")
	}

	switch m.Body.Text {
	case "Геолокация дома":
		for _, attachment := range m.Body.Attachments {
			if attachment.Type == model.AttachLocation {
				lat, long := attachment.Latitude, attachment.Longitude
				count := 0
				if s, ok := c.State().Get("addr_count"); ok {
					i, err := strconv.ParseInt(s, 10, 64)
					if err == nil {
						count = int(i)
					}
				}

				key := fmt.Sprintf("addr_%d", count)
				val := fmt.Sprintf("%f;%f", lat, long)
				c.State().Set(key, val)
				c.State().Set("addr_count", fmt.Sprint(count+1))
				return c.Respond(maxbot.NewMessage().SetText("Координаты получены"))
			}
		}
	case "Закончить":
		defer c.State().Clear()
		if err := omh.flushAddresses(c); err != nil {
			return err
		}
		return c.Respond(maxbot.NewMessage().SetText("Координаты сохранены"))
	}

	return nil
}
