package filters

import (
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	"github.com/abelith/etu-bot/pkg/maxlib/core"
	"github.com/abelith/etu-bot/pkg/maxlib/state"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"strings"
)

func StartCommand(c *mcontext.Context) bool {
	upd := c.Update()
	return upd.UpdateType == model.UpdateMessageCreated && strings.HasPrefix(upd.GetMessage().Body.Text, "/start")
}

func Command(cmd string) core.Filter {
	return func(c *mcontext.Context) bool {
		upd := c.Update()
		return upd.UpdateType == model.UpdateMessageCreated && strings.HasPrefix(upd.GetMessage().Body.Text, cmd)
	}
}

func State(state state.State) core.Filter {
	return func(c *mcontext.Context) bool {
		return c.State().GetState() == state
	}
}
