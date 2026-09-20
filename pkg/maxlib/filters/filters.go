package filters

import (
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	"github.com/abelith/etu-bot/pkg/maxlib/core"
	"github.com/abelith/etu-bot/pkg/maxlib/state"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"regexp"
	"strings"
)

func StartCommand(c *mcontext.Context) bool {
	upd := c.Update()
	return upd.UpdateType == model.UpdateBotStarted
}

func Command(cmd string) core.Filter {
	return func(c *mcontext.Context) bool {
		upd := c.Update()
		return upd.UpdateType == model.UpdateMessageCreated && strings.HasPrefix(upd.GetCommand().Command, cmd)
	}
}

func State(state state.State) core.Filter {
	return func(c *mcontext.Context) bool {
		return c.State().GetState() == state
	}
}

func MessageText(s string) core.Filter {
	return func(c *mcontext.Context) bool {
		if msg := c.Update().Message; msg != nil {
			return s == msg.Body.Text
		}
		return false
	}
}

func MessageTextRe(re *regexp.Regexp) core.Filter {
	return func(c *mcontext.Context) bool {
		if msg := c.Update().Message; msg != nil {
			return re.MatchString(msg.Body.Text)
		}
		return false
	}
}

func MessageGeo(c *mcontext.Context) bool {
	for _, attachment := range c.Update().Message.Body.Attachments {
		if attachment.Type == model.AttachLocation {
			return true
		}
	}

	return false
}
