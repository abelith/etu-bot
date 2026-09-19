package core

import (
	"context"
	"errors"
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	"github.com/abelith/etu-bot/pkg/maxlib/state"
	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"time"
)

type FSMStorage interface {
	Put(ctx *state.Context) error
	Delete(ctx *state.Context) error
}

type UpdateHandler interface {
	HandleUpdate(c *mcontext.Context) error
	Use(mw ...Middleware)
}

type Dispatcher struct {
	Api        *maxbot.Api
	Handler    UpdateHandler
	FSMStorage FSMStorage
}

func (d *Dispatcher) getUpdates(ctx context.Context) <-chan model.Update {
	var updates []model.Update
	var marker int64
	var err error

	ch := make(chan model.Update)

	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.Tick(1 * time.Second):
				updates, marker, err = d.Api.Subscriptions.GetUpdates(ctx, marker)
				if err != nil && !errors.Is(err, context.Canceled) {
					panic(err)
				}

				for _, update := range updates {
					select {
					case <-ctx.Done():
						return
					case ch <- update:
					}
				}
			}
		}
	}()

	return ch
}

func (d *Dispatcher) handleUpdates(ch <-chan model.Update) {
	for update := range ch {
		go func() {
			ctx := context.Background()
			c := mcontext.New(ctx, update)

			d.Handler.HandleUpdate(c)

			st := c.State()
			if st.GetState() == state.ClearState {
				d.FSMStorage.Delete(st)
			}
			if st.GetState() != "" {
				d.FSMStorage.Put(st)
			}
		}()
	}
}

func (d *Dispatcher) StartPolling(ctx context.Context) {
	d.handleUpdates(d.getUpdates(ctx))
	<-ctx.Done()
}
