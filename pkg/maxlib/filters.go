package maxlib

import (
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"strings"
)

func StartCommand(upd model.Update) bool {
	return upd.UpdateType == model.UpdateMessageCreated && strings.HasPrefix(upd.GetMessage().Body.Text, "/start")
}

func Command(cmd string) Filter {
	return func(upd model.Update) bool {
		return upd.UpdateType == model.UpdateMessageCreated && strings.HasPrefix(upd.GetMessage().Body.Text, cmd)
	}
}
