package maxlib

import (
	"context"
	"errors"
	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"time"
)

type UpdateHandler interface {
	HandleUpdate(update model.Update) error
	Use(mw ...Middleware)
}

type Dispatcher struct {
	Api     *maxbot.Api
	Handler UpdateHandler
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
			_ = d.Handler.HandleUpdate(update)
		}()
	}
}

func (d *Dispatcher) StartPolling(ctx context.Context) {
	d.handleUpdates(d.getUpdates(ctx))
	<-ctx.Done()
}
