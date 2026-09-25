package dadata

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ekomobile/dadata/v2"
	"github.com/ekomobile/dadata/v2/api/clean"
	"github.com/ekomobile/dadata/v2/api/suggest"
	"github.com/ekomobile/dadata/v2/client"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ApiKey    string `env:"DADATA_API_KEY"`
	SecretKey string `env:"DADATA_API_SECRET"`
}

type Geocoder struct {
	cleanAPI   *clean.Api
	suggestAPI *suggest.Api
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

	return &Geocoder{
		cleanAPI:   dadata.NewCleanApi(client.WithCredentialProvider(&creds)),
		suggestAPI: dadata.NewSuggestApi(client.WithCredentialProvider(&creds)),
	}
}

func (g *Geocoder) ParseAddress(ctx context.Context, addr string) (lat, long float64, err error) {
	res, err := g.cleanAPI.Address(ctx, addr)
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

type GeolocateRequest struct {
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
	Count        int     `json:"count,omitempty"`
	RadiusMeters int     `json:"radius_meters,omitempty"`
}

func (g *Geocoder) ParseCoordinates(ctx context.Context, lat, long float64) (addr string, err error) {
	result := &suggest.AddressResponse{}

	req := &GeolocateRequest{
		Lat: lat,
		Lon: long,
	}

	if err = g.suggestAPI.Client.Post(ctx, "geolocate/address", req, result); err != nil {
		return "", err
	}

	if len(result.Suggestions) < 1 {
		return "", fmt.Errorf("address not found")
	}

	return result.Suggestions[0].Value, nil
}
