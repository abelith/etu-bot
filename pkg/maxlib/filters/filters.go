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
	if upd.UpdateType == model.UpdateBotStarted {
		return true
	}

	return strings.HasPrefix(upd.GetCommand().Command, "/start")
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
			s := strings.TrimSpace(msg.Body.Text)
			return re.MatchString(s)
		}
		return false
	}
}

func MessageGeo(c *mcontext.Context) bool {
	if c.Update().Message == nil {
		return false
	}

	for _, attachment := range c.Update().Message.Body.Attachments {
		if attachment.Type == model.AttachLocation {
			return true
		}
	}

	return false
}

func BotStopped(c *mcontext.Context) bool {
	return c.Update().UpdateType == model.UpdateBotStopped
}

func Or(filters ...core.Filter) core.Filter {
	return func(c *mcontext.Context) bool {
		for _, filter := range filters {
			if filter(c) {
				return true
			}
		}
		return false
	}
}
