package usecases

import (
	"context"
	"fmt"
	"github.com/abelith/etu-bot/internal/infrastructure/events"
	"github.com/abelith/etu-bot/internal/models"
	"github.com/google/uuid"
	"strings"
)

type HousingCommand interface {
	AddInhabitant(ctx context.Context, inhabitant *models.Inhabitant) error
}

type HousingQuery interface {
	GetNearbyHouses(ctx context.Context, lat, long, eps float64) ([]*models.House, error)
}

type OrgCommand interface {
	AddEmployee(ctx context.Context, userID int, token uuid.UUID) (string, error)
}

type UserCommand interface {
	Delete(ctx context.Context, id int) error
}

type GeoCoder interface {
	ParseAddress(ctx context.Context, addr string) (lat, long float64, err error)
}

type UseCases struct {
	hc  HousingCommand
	hq  HousingQuery
	oc  OrgCommand
	uc  UserCommand
	geo GeoCoder
}

func NewUseCases(
	hc HousingCommand,
	hq HousingQuery,
	oc OrgCommand,
	uc UserCommand,
	geo GeoCoder,
) *UseCases {
	return &UseCases{
		hc:  hc,
		hq:  hq,
		oc:  oc,
		uc:  uc,
		geo: geo,
	}
}

func (uc *UseCases) GetHouses(ctx context.Context, latitude, longitude float64) ([]*models.House, error) {
	return uc.hq.GetNearbyHouses(ctx, latitude, longitude, 5.0)
}

func (uc *UseCases) AddInhabitant(ctx context.Context, inhabitant *models.Inhabitant) error {
	if err := uc.hc.AddInhabitant(ctx, inhabitant); err != nil {
		return err
	}

	events.Log.Write(map[string]any{"type": "inhabitant_added", "id": inhabitant.Id, "addr": inhabitant.HouseAddress})
	return nil
}

func (uc *UseCases) AddEmployee(ctx context.Context, id int, token string) (string, error) {
	parts := strings.Split(token, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid token format")
	}
	token = parts[1]
	tokenUUID, err := uuid.Parse(token)
	if err != nil {
		return "", fmt.Errorf("failed to parse uuid: %w", err)
	}

	name, err := uc.oc.AddEmployee(ctx, id, tokenUUID)
	if err != nil {
		return "", fmt.Errorf("failed to add employee: %w", err)
	}

	events.Log.Write(map[string]any{"type": "employee_added", "id": id, "org_name": name})
	return name, nil
}

func (uc *UseCases) DeleteUser(ctx context.Context, id int) error {
	if err := uc.uc.Delete(ctx, id); err != nil {
		return err
	}

	events.Log.Write(map[string]any{"type": "user_deleted", "id": id})
	return nil
}

func (uc *UseCases) ParseAddress(ctx context.Context, addr string) (lat float64, long float64, err error) {
	return uc.geo.ParseAddress(ctx, addr)
}
