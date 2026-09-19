package main

import (
	"context"
	"crypto/tls"
	"fmt"
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	"github.com/abelith/etu-bot/pkg/maxlib/core"
	"github.com/abelith/etu-bot/pkg/maxlib/filters"
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

	router := &core.Router{}
	router.HandleFunc(func(c *mcontext.Context) error {
		fmt.Println("start command hit")
		return nil
	}, filters.StartCommand)
	router.HandleFunc(func(c *mcontext.Context) error {
		fmt.Println("fallback handler")
		return nil
	})
	router.Use(func(next core.UpdateHandler) core.UpdateHandler {
		return core.HandlerFunc(func(c *mcontext.Context) error {
			fmt.Println("mw triggered")
			return next.HandleUpdate(c)
		})
	})

	dp := &core.Dispatcher{
		Api:     api,
		Handler: router,
	}
	dp.StartPolling(ctx)
}
