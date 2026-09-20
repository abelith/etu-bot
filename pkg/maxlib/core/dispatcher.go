package core

import (
	"context"
	"errors"
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	errors2 "github.com/abelith/etu-bot/pkg/maxlib/errors"
	"github.com/abelith/etu-bot/pkg/maxlib/state"
	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"log"
	"time"
)

type FSMStorage interface {
	Get(ctx context.Context, userID int) (*state.Context, error)
	Put(ctx context.Context, state *state.Context) error
	Delete(ctx context.Context, state *state.Context) error
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
			case <-time.Tick(200 * time.Millisecond):
				updates, marker, err = d.Api.Subscriptions.GetUpdates(ctx, marker)
				if err != nil && !errors.Is(err, context.Canceled) {
					log.Println(err)
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
		go d.handleUpdate(update)
	}
}

func (d *Dispatcher) StartPolling(ctx context.Context) {
	d.handleUpdates(d.getUpdates(ctx))
	<-ctx.Done()
}

func (d *Dispatcher) handleUpdate(upd model.Update) {
	ctx := context.Background()
	st, err := d.FSMStorage.Get(ctx, int(upd.UserID))
	if err != nil && errors.Is(err, errors2.ErrStateNotFound) {
		st = state.New(int(upd.UserID))
	} else if err != nil {
		log.Println(err)
		return
	}

	c := mcontext.New(ctx, upd, st, d.Api)

	if err := d.Handler.HandleUpdate(c); err != nil {
		log.Println(err)
		return
	}

	st = c.State()
	if st.GetState() == state.ClearState {
		if err := d.FSMStorage.Delete(ctx, st); err != nil {
			log.Println(err)
			return
		}
	}
	if st.GetState() != "" {
		if err := d.FSMStorage.Put(ctx, st); err != nil {
			log.Println(err)
			return
		}
	}
}
