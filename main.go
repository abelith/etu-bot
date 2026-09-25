package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/abelith/etu-bot/internal/adapters/dadata"
	max2 "github.com/abelith/etu-bot/internal/adapters/max"
	"github.com/abelith/etu-bot/internal/adapters/postgres"
	"github.com/abelith/etu-bot/internal/adapters/tags"
	errors2 "github.com/abelith/etu-bot/internal/errors"
	"github.com/abelith/etu-bot/internal/handlers"
	"github.com/abelith/etu-bot/internal/infrastructure/events"
	"github.com/abelith/etu-bot/internal/usecases"
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	"github.com/abelith/etu-bot/pkg/maxlib/core"
	"github.com/abelith/etu-bot/pkg/maxlib/state"
	"github.com/abelith/etu-bot/pkg/pgpool"
	"github.com/rs/zerolog/log"
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
	botToken := os.Getenv("TOKEN")
	api, err := maxbot.NewApi(botToken, opts...)
	if err != nil {
		fmt.Println("api initial err:", err)
		return
	}

	dadataCfg, err := dadata.NewConfig()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	geo := dadata.NewGeocoder(dadataCfg)

	pool, err := pgpool.NewPool(ctx)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	pgCfg, err := postgres.NewConfig()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	repo := postgres.NewRepo(pgCfg, pool)
	repo.InitLogger(ctx)
	events.SetLogger(repo)

	cfgTags, err := tags.NewConfig()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	t, err := tags.NewParser(cfgTags)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	n := &max2.Notifier{BotToken: botToken}

	rt := (&handlers.StartHandlers{UseCases: usecases.NewUseCases(
		repo, repo, repo, repo, repo, geo, t, repo, repo, n,
	)}).Router()
	rt.Use(func(next core.UpdateHandler) core.UpdateHandler {
		return core.HandlerFunc(func(c *mcontext.Context) error {
			err := next.HandleUpdate(c)
			if errors.Is(err, errors2.ErrNoNearbyHouses) {
				return c.Respond(maxbot.NewMessage().SetText("Ближайшие дома не найдены"))
			}
			if err != nil {
				return c.Respond(maxbot.NewMessage().SetText("Произошла неизвестная ошибка. Попробуйте снова"))
			}
			return nil
		})
	})
	rt.Use(func(next core.UpdateHandler) core.UpdateHandler {
		return core.HandlerFunc(func(c *mcontext.Context) error {
			ctx := log.Logger.WithContext(c.Context())
			c.SetContext(ctx)
			return next.HandleUpdate(c)
		})
	})
	rt.Use(func(next core.UpdateHandler) core.UpdateHandler {
		return core.HandlerFunc(func(c *mcontext.Context) error {
			log.Debug().Any("state", c.State()).Any("update", c.Update()).Send()
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
