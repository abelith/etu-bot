package usecases

import (
	"context"
	"fmt"
	"github.com/abelith/etu-bot/internal/infrastructure/events"
	"github.com/abelith/etu-bot/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"strings"
	"sync"
	"time"
)

const (
	defaultBatchSize      = 128
	defaultNotifyInterval = 30 * time.Second
)

type HousingCommand interface {
	AddInhabitant(ctx context.Context, inhabitant *models.Inhabitant) error
	CreateClusterWithHouses(ctx context.Context, cluster *models.Cluster) error
}

type HousingQuery interface {
	GetNearbyHouses(ctx context.Context, lat, long, eps float64) ([]*models.House, error)
	GetClusterNamesByOrgID(ctx context.Context, orgID int, cursor string, limit int) ([]string, error)
}

type OrgCommand interface {
	AddEmployee(ctx context.Context, userID int, token uuid.UUID) (string, error)
}

type OrgQuery interface {
	GetIDByUserID(ctx context.Context, userID int) (int, error)
	GetOrgMemberMe(ctx context.Context, userID int) (*models.OrgMemberMe, error)
}

type UserCommand interface {
	Update(ctx context.Context, update models.UserUpdate) error
	Delete(ctx context.Context, id int) error
}

type GeoCoder interface {
	ParseAddress(ctx context.Context, addr string) (lat, long float64, err error)
	ParseCoordinates(ctx context.Context, lat, long float64) (addr string, err error)
}

type TagParser interface {
	ParseTags(ctx context.Context, text string) ([]string, error)
}

type NotificationsCommand interface {
	AddNotification(ctx context.Context, n *models.Notification) (int, error)
	AddNotifyTasks(ctx context.Context, notificationID int) error
	AckBatch(ctx context.Context, ids []int) error
}

type NotificationsQuery interface {
	GetBatch(ctx context.Context, batchSize int) ([]models.NotificationTask, error)
}

type Notifier interface {
	Notify(ctx context.Context, task models.NotificationTask) error
}

type UseCases struct {
	hc  HousingCommand
	hq  HousingQuery
	oc  OrgCommand
	oq  OrgQuery
	uc  UserCommand
	geo GeoCoder
	tp  TagParser
	nc  NotificationsCommand
	nq  NotificationsQuery
	n   Notifier
	wg  sync.WaitGroup
}

func NewUseCases(
	hc HousingCommand,
	hq HousingQuery,
	oc OrgCommand,
	oq OrgQuery,
	uc UserCommand,
	geo GeoCoder,
	tp TagParser,
	nc NotificationsCommand,
	nq NotificationsQuery,
	n Notifier,
) *UseCases {
	return &UseCases{
		hc:  hc,
		hq:  hq,
		oc:  oc,
		oq:  oq,
		uc:  uc,
		geo: geo,
		tp:  tp,
		nc:  nc,
		nq:  nq,
		n:   n,
	}
}

func (uc *UseCases) Init(ctx context.Context) {
	uc.wg.Go(func() {
		uc.runNotificationsWorker(ctx)
	})
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
	return uc.oq.GetOrgMemberMe(ctx, id)
}

func (uc *UseCases) ChangeUserName(ctx context.Context, id int, name string) error {
	return uc.uc.Update(ctx, models.UserUpdate{
		Id:   id,
		Name: &name,
	})
}

func (uc *UseCases) ActivateUser(ctx context.Context, id int) error {
	t := true
	return uc.uc.Update(ctx, models.UserUpdate{
		Id:     id,
		Active: &t,
	})
}

func (uc *UseCases) DeactivateUser(ctx context.Context, id int) error {
	f := false
	return uc.uc.Update(ctx, models.UserUpdate{
		Id:     id,
		Active: &f,
	})
}

func (uc *UseCases) CreateCluster(ctx context.Context, userID int, name string, coords []models.Coordinates) error {
	orgID, err := uc.oq.GetIDByUserID(ctx, userID)
	if err != nil {
		return err
	}

	houses := make([]*models.House, 0, len(coords))
	for _, coord := range coords {
		addr, err := uc.geo.ParseCoordinates(ctx, coord.Latitude, coord.Longitude)
		if err != nil {
			return fmt.Errorf("failed to parse coordinates: %w", err)
		}
		houses = append(houses, &models.House{
			Address:   addr,
			Latitude:  coord.Latitude,
			Longitude: coord.Longitude,
		})
	}

	cluster := &models.Cluster{
		OrgID:  orgID,
		Name:   name,
		Houses: houses,
	}

	if err := uc.hc.CreateClusterWithHouses(ctx, cluster); err != nil {
		return err
	}

	events.Log.Write(map[string]any{"type": "cluster_created", "cluster_name": cluster.Name, "org_id": cluster.OrgID})
	return nil
}

func (uc *UseCases) GetClusterNames(ctx context.Context, userID int) ([]string, error) {
	orgID, err := uc.oq.GetIDByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	clusterNames, err := uc.hq.GetClusterNamesByOrgID(ctx, orgID, "", 0)
	if err != nil {
		return nil, err
	}
	return clusterNames, nil
}

func (uc *UseCases) Notify(ctx context.Context, n *models.Notification, content string) error {
	tags, err := uc.tp.ParseTags(ctx, content)
	if err != nil {
		return err
	}
	n.Tags = tags
	id, err := uc.nc.AddNotification(ctx, n)
	if err != nil {
		return err
	}

	if err := uc.nc.AddNotifyTasks(ctx, id); err != nil {
		return err
	}

	events.Log.Write(map[string]any{"type": "notification_created", "tags": n.Tags, "id": n.Id})
	return fmt.Errorf("unimplemented")
}

func (uc *UseCases) processBatch(ctx context.Context) error {
	tasks, err := uc.nq.GetBatch(ctx, defaultBatchSize)
	if err != nil {
		return err
	}

	acks := make([]int, 0, len(tasks))
	for _, task := range tasks {
		if err := uc.n.Notify(ctx, task); err != nil {
			log.Ctx(ctx).Warn().
				Err(err).
				Int("task_id", task.TaskID).
				Int("user_id", task.UserID).
				Msg("failed to notify, will be retried on next iteration")
			continue
		}
		acks = append(acks, task.TaskID)
	}

	if err := uc.nc.AckBatch(ctx, acks); err != nil {
		return err
	}

	return nil
}

func (uc *UseCases) runNotificationsWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.Tick(defaultNotifyInterval):
			if err := uc.processBatch(ctx); err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to process notifications batch")
			}
		}
	}
}

func (uc *UseCases) Wait() {
	uc.wg.Wait()
}
