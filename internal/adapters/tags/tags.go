package tags

import (
	"context"
	"fmt"
	"github.com/abelith/etu-bot/pkg/api"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ServerUrl string `env:"TAG_PARSER_URL"`
}

func NewConfig() (*Config, error) {
	cfg := Config{}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Parser struct {
	api *api.Client
}

func NewParser(cfg *Config) (*Parser, error) {
	client, err := api.NewClient(cfg.ServerUrl)
	if err != nil {
		return nil, err
	}

	return &Parser{api: client}, nil
}

func (p *Parser) ParseTags(ctx context.Context, text string) ([]string, error) {
	res, err := p.api.AssignTags(ctx, &api.AssignTagsRequest{Message: text})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.AssignTagsResponse:
		return r.Data, nil
	case *api.AssignTagsBadRequest:
		return nil, fmt.Errorf(r.Message)
	case *api.AssignTagsInternalServerError:
		return nil, fmt.Errorf(r.Message)
	case *api.AssignTagsServiceUnavailable:
		return nil, fmt.Errorf(r.Message)
	}

	return nil, fmt.Errorf("failed to parse tags")
}
