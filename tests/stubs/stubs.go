package stubs

import (
	"context"
	"github.com/abelith/etu-bot/internal/models"
)

type UseCaseStub struct{}

func (us *UseCaseStub) VerifyOrganization(_ context.Context, _ string) (string, error) {
	return "OrgName", nil
}

func (us *UseCaseStub) GetHouses(_ context.Context, _, _ float64) ([]*models.House, error) {
	houses := []*models.House{
		{
			Address:     "1/1/1",
			Longitude:   0,
			Latitude:    0,
			Inhabitants: 123,
		},
		{
			Address:     "2/2/2",
			Longitude:   0,
			Latitude:    0,
			Inhabitants: 1234,
		},
	}
	return houses, nil
}

func (us *UseCaseStub) AddInhabitant(_ context.Context, _ *models.Inhabitant) error {
	return nil
}
