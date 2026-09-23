package handlers

import (
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
)

var (
	menuKb                = model.NewKeyboard()
	inhabitantMenuKb      = model.NewKeyboard()
	orgMenuKb             = model.NewKeyboard()
	orgMeKb               = model.NewKeyboard()
	addClusterAddressesKb = model.NewKeyboard()
	geoKb                 = model.NewKeyboard()
	confirmKb             = model.NewKeyboard()
	orgPriorityKb         = model.NewKeyboard()
)

func init() {
	menuKb.AddRow().AddMessage("Меню")
	inhabitantMenuKb.AddRow().
		AddMessage("Мои данные").
		AddMessage("Настройки уведомлений").
		AddMessage("Создать объявление").
		AddMessage("Создать заявку").
		AddMessage("Просмотреть аналитику")
	orgMenuKb.AddRow().
		AddMessage("Мои данные").
		AddMessage("Просмотреть аналитику").
		AddMessage("Создать уведомление").
		AddMessage("Просмотреть заявки")
	orgMeKb.AddRow().
		AddMessage("Изменить имя").
		AddMessage("Активировать").
		AddMessage("Деактивировать").
		AddMessage("Создать кластер")
	addClusterAddressesKb.AddRow().
		AddMessage("Отправить адреса").
		AddMessage("Отправить файл с адресами").
		AddMessage("Отправить геолокацию")
	geoKb.AddRow().
		AddGeoLocation("Геолокация дома", false).
		AddMessage("Закончить")
	confirmKb.AddRow().
		AddMessage("Подтвердить").
		AddMessage("Отмена")
	orgPriorityKb.AddRow().
		AddMessage("3").
		AddMessage("4").
		AddMessage("5")
}
