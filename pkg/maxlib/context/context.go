package context

import (
	"context"
	"github.com/abelith/etu-bot/pkg/maxlib/state"
	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"time"
)

type Context struct {
	ctx   context.Context
	upd   model.Update
	state *state.Context
	api   *maxbot.Api
}

func New(ctx context.Context, upd model.Update, state *state.Context, api *maxbot.Api) *Context {
	return &Context{
		ctx:   ctx,
		upd:   upd,
		state: state,
		api:   api,
	}
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) SetContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *Context) State() *state.Context {
	return c.state
}

func (c *Context) Update() model.Update {
	return c.upd
}

func (c *Context) Respond(msg *maxbot.Message) error {
	msg = msg.SetChat(c.upd.ChatID).SetUser(c.upd.UserID)

	_, err := c.api.Messages.Send(c.ctx, msg)
	return err
}

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *Context) Err() error {
	return c.ctx.Err()
}

func (c *Context) Value(key any) any {
	return c.ctx.Value(key)
}
