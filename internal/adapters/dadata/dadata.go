package dadata

import (
	"context"
	"fmt"
	"github.com/ekomobile/dadata/v2"
	"github.com/ekomobile/dadata/v2/api/clean"
	"github.com/ekomobile/dadata/v2/client"
	"github.com/ilyakaznacheev/cleanenv"
	"strconv"
)

type Config struct {
	ApiKey    string `env:"DADATA_API_KEY"`
	SecretKey string `env:"DADATA_API_SECRET"`
}

type Geocoder struct {
	api *clean.Api
}

func NewConfig() (*Config, error) {
	cfg := Config{}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func NewGeocoder(cfg *Config) *Geocoder {
	creds := client.Credentials{
		ApiKeyValue:    cfg.ApiKey,
		SecretKeyValue: cfg.SecretKey,
	}
	api := dadata.NewCleanApi(client.WithCredentialProvider(&creds))
	return &Geocoder{api: api}
}

func (g *Geocoder) ParseAddress(ctx context.Context, addr string) (lat, long float64, err error) {
	res, err := g.api.Address(ctx, addr)
	if err != nil {
		return 0, 0, err
	}

	if len(res) < 1 {
		return 0, 0, fmt.Errorf("address not found")
	}

	lat, err = strconv.ParseFloat(res[0].GeoLat, 64)
	if err != nil {
		return 0, 0, err
	}

	long, err = strconv.ParseFloat(res[0].GeoLon, 64)
	if err != nil {
		return 0, 0, err
	}

	return lat, long, nil
}
