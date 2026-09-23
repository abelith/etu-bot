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

	events.Log.Write(map[string]any{"type": "inhabitant_added", "id": inhabitant.User.Id, "addr": inhabitant.HouseAddress})
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

func (uc *UseCases) GetUserOrg(ctx context.Context, id int) (*models.OrgMemberMe, error) {
	// get user org data agg
	return nil, fmt.Errorf("unimplemented")
}

func (uc *UseCases) ChangeUserName(ctx context.Context, id int, name string) error {
	// update user: set username = name
	return fmt.Errorf("unimplemented")
}

func (uc *UseCases) ActivateUser(ctx context.Context, id int) error {
	// update user: set active = true
	return fmt.Errorf("unimplemented")
}

func (uc *UseCases) DeactivateUser(ctx context.Context, id int) error {
	// update user: set active = false
	return fmt.Errorf("unimplemented")
}

func (uc *UseCases) CreateCluster(ctx context.Context, name string, coords []models.Coordinates) error {
	// create cluster with name name from set of coords
	return fmt.Errorf("unimplemented")
}

func (uc *UseCases) GetClusterNames(ctx context.Context, userID int) ([]string, error) {
	// identify org id by user id
	// retrieve cluster names for org id
	return nil, fmt.Errorf("unimplemented")
}

func (uc *UseCases) Notify(ctx context.Context, n *models.Notification, content string) error {
	// parse tags via api
	// add tags to n
	// write notification to outbox
	// create and write tasks to outbox
	// start notification worker
	return fmt.Errorf("unimplemented")
}
