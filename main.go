package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/abelith/etu-bot/internal/handlers"
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	"github.com/abelith/etu-bot/pkg/maxlib/core"
	"github.com/abelith/etu-bot/pkg/maxlib/state"
	"github.com/abelith/etu-bot/tests/stubs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	opts := []maxbot.Opt{
		maxbot.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // for dev purposes only
				},
			},
			Timeout: 10 * time.Second,
		}),
	}
	api, err := maxbot.NewApi(os.Getenv("TOKEN"), opts...)
	if err != nil {
		fmt.Println("api initial err:", err)
		return
	}

	rt := (&handlers.Handlers{UseCases: &stubs.UseCaseStub{}}).Router()
	rt.Use(func(next core.UpdateHandler) core.UpdateHandler {
		return core.HandlerFunc(func(c *mcontext.Context) error {
			fmt.Println(c.State(), c.Update())
			return next.HandleUpdate(c)
		})
	})

	dp := &core.Dispatcher{
		Api:        api,
		Handler:    rt,
		FSMStorage: state.NewMemoryFSMStorage(),
	}
	dp.StartPolling(ctx)
}
