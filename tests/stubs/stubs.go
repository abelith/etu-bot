package stubs

import (
	"context"
	"fmt"
	"github.com/abelith/etu-bot/internal/models"
)

type UseCaseStub struct{}

func (us *UseCaseStub) VerifyOrganization(_ context.Context, _ string) (string, error) {
	fmt.Println("VerifyOrganization called")
	return "OrgName", nil
}

func (us *UseCaseStub) GetHouses(_ context.Context, _, _ float64) ([]*models.House, error) {
	fmt.Println("GetHouses called")
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
	fmt.Println("AddInhabitant called")
	return nil
}

func (us *UseCaseStub) DeleteUser(_ context.Context, _ int) error {
	fmt.Println("DeleteUser called")
	return nil
}
