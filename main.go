package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/abelith/etu-bot/pkg/maxlib"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
)

func HandleUpdate(_ context.Context, update model.Update) {
	fmt.Println(update)
}

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

	router := &maxlib.Router{}
	router.HandleFunc(func(upd model.Update) error {
		fmt.Println("start command hit")
		return nil
	}, maxlib.StartCommand)
	router.HandleFunc(func(upd model.Update) error {
		fmt.Println("fallback handler")
		return nil
	})
	router.Use(func(next maxlib.UpdateHandler) maxlib.UpdateHandler {
		return maxlib.HandlerFunc(func(upd model.Update) error {
			fmt.Println("mw triggered")
			return next.HandleUpdate(upd)
		})
	})

	dp := &maxlib.Dispatcher{
		Api:     api,
		Handler: router,
	}
	dp.StartPolling(ctx)
}
