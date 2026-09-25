package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	errors2 "github.com/abelith/etu-bot/internal/errors"
	"github.com/abelith/etu-bot/internal/infrastructure/events"
	"github.com/abelith/etu-bot/internal/models"
	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"time"
)

const (
	notificationTaskStatusSent = "sent"
	clusterNamesDefaultLimit   = 50
)

type Config struct {
	BatchCap     int           `env:"EVENT_LOG_BATCH_SIZE" env-default:"128"`
	WriteTimeout time.Duration `env:"EVENT_LOG_WRITE_TIMEOUT" env-default:"200ms"`
	FlushTimeout time.Duration `env:"EVENT_LOG_FLUSH_TIMEOUT" env-default:"10s"`
}

func NewConfig() (*Config, error) {
	cfg := Config{}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Repository struct {
	cfg  *Config
	pool *pgxpool.Pool
	ch   chan events.Event
}

func NewRepo(cfg *Config, pool *pgxpool.Pool) *Repository {
	return &Repository{
		cfg:  cfg,
		pool: pool,
		ch:   make(chan events.Event, 1),
	}
}

func (r *Repository) AddInhabitant(ctx context.Context, inhabitant *models.Inhabitant) error {
	return r.runTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		insertUser := sq.Insert("users").
			Columns("id", "user_name").
			Values(inhabitant.User.Id, inhabitant.User.Name).
			Suffix("ON CONFLICT (id) DO UPDATE SET user_name = EXCLUDED.user_name").
			PlaceholderFormat(sq.Dollar)

		query, args, err := insertUser.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build insert user query: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}

		selectHouse := sq.Select("id").
			From("houses").
			Where(sq.Eq{"addr": inhabitant.HouseAddress}).
			PlaceholderFormat(sq.Dollar)

		query, args, err = selectHouse.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build select house query: %w", err)
		}

		var houseID int
		if err := tx.QueryRow(ctx, query, args...).Scan(&houseID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("house not found")
			}
			return fmt.Errorf("failed to find house: %w", err)
		}

		insertLink := sq.Insert("users_houses").
			Columns("user_id", "house_id").
			Values(inhabitant.User.Id, houseID).
			Suffix("ON CONFLICT (user_id) DO UPDATE SET house_id = EXCLUDED.house_id").
			PlaceholderFormat(sq.Dollar)

		query, args, err = insertLink.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build insert user_houses query: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to link user to house: %w", err)
		}

		return nil
	})
}

func (r *Repository) AddEmployee(ctx context.Context, userID int, token uuid.UUID) (string, error) {
	selectToken := sq.Select("organizations.id", "organizations.name").
		From("tokens").
		Join("organizations ON organizations.id = tokens.org_id").
		Where(sq.Eq{"tokens.id": token}).
		PlaceholderFormat(sq.Dollar)

	query, args, err := selectToken.ToSql()
	if err != nil {
		return "", fmt.Errorf("failed to build query: %w", err)
	}

	var orgID int
	var orgName string
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&orgID, &orgName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors2.ErrInvalidToken
		}
		return "", fmt.Errorf("failed to find token: %w", err)
	}

	err = r.runTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		deleteToken := sq.Delete("tokens").
			Where(sq.Eq{"id": token}).
			PlaceholderFormat(sq.Dollar)

		query, args, err := deleteToken.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build query: %w", err)
		}

		res, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to delete token: %w", err)
		}
		if res.RowsAffected() == 0 {
			return errors2.ErrInvalidToken
		}

		insertUser := sq.Insert("users").
			Columns("id", "user_name").
			Values(userID, "").
			Suffix("ON CONFLICT (id) DO NOTHING").
			PlaceholderFormat(sq.Dollar)

		query, args, err = insertUser.ToSql()
		if err != nil {
			return fmt.Errorf("failed build query: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}

		insertUserOrg := sq.Insert("users_organizations").
			Columns("user_id", "org_id", "role").
			Values(userID, orgID, "employee").
			Suffix("ON CONFLICT (user_id, org_id) DO NOTHING").
			PlaceholderFormat(sq.Dollar)

		query, args, err = insertUserOrg.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build query: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to link user to organization: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return orgName, nil
}

func (r *Repository) GetNearbyHouses(ctx context.Context, lat, long, eps float64) ([]*models.House, error) {
	point := fmt.Sprintf("SRID=4326;POINT(%f %f)", long, lat)

	selectHouses := sq.Select(
		"addr",
		"ST_Y(location::geometry)",
		"ST_X(location::geometry)",
		"(SELECT COUNT(*) FROM user_houses WHERE user_houses.house_id = houses.id)",
	).
		From("houses").
		Where("ST_DWithin(location::geography, ?::geography, ?)", point, eps).
		PlaceholderFormat(sq.Dollar)

	query, args, err := selectHouses.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query nearby houses: %w", err)
	}
	defer rows.Close()

	houses := make([]*models.House, 0)
	for rows.Next() {
		house := &models.House{}
		if err := rows.Scan(&house.Address, &house.Latitude, &house.Longitude, &house.Inhabitants); err != nil {
			return nil, fmt.Errorf("failed to scan house: %w", err)
		}
		houses = append(houses, house)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	if len(houses) < 1 {
		return nil, errors2.ErrNoNearbyHouses
	}

	return houses, nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	deleteUser := sq.Delete("users").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	query, args, err := deleteUser.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (r *Repository) GetBatch(ctx context.Context, batchSize int) ([]models.NotificationTask, error) {
	selectTasks := sq.Select(
		"nt.task_id",
		"nt.user_id",
		"n.source_id",
	).
		From("notifications_tasks nt").
		Join("notifications n ON n.id = nt.notification_id").
		Where(sq.Eq{"nt.status": "pending"}).
		OrderBy("nt.created_at ASC").
		Limit(uint64(batchSize)).
		PlaceholderFormat(sq.Dollar)

	query, args, err := selectTasks.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query notification tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]models.NotificationTask, 0, batchSize)
	for rows.Next() {
		var task models.NotificationTask
		if err := rows.Scan(&task.TaskID, &task.UserID, &task.SourceID); err != nil {
			return nil, fmt.Errorf("failed to scan notification task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return tasks, nil
}

func (r *Repository) AddNotification(ctx context.Context, n *models.Notification) (int, error) {
	var notificationID int

	err := r.runTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		selectCluster := sq.Select("id").
			From("clusters").
			Where(sq.Eq{"name": n.ClusterName}).
			PlaceholderFormat(sq.Dollar)

		query, args, err := selectCluster.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build select cluster query: %w", err)
		}

		var clusterID int
		if err := tx.QueryRow(ctx, query, args...).Scan(&clusterID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("cluster %q not found", n.ClusterName)
			}
			return fmt.Errorf("failed to find cluster: %w", err)
		}

		insertNotification := sq.Insert("notifications").
			Columns("cluster_id", "source_id", "priority").
			Values(clusterID, n.SourceID, int32(n.Priority)).
			Suffix("RETURNING id").
			PlaceholderFormat(sq.Dollar)

		query, args, err = insertNotification.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build insert notification query: %w", err)
		}
		if err := tx.QueryRow(ctx, query, args...).Scan(&notificationID); err != nil {
			return fmt.Errorf("failed to insert notification: %w", err)
		}

		for _, tag := range n.Tags {
			tagID, err := r.upsertTag(ctx, tx, tag)
			if err != nil {
				return fmt.Errorf("failed to upsert tag %q: %w", tag, err)
			}

			insertTag := sq.Insert("notifications_tags").
				Columns("notification_id", "tag_id").
				Values(notificationID, tagID).
				PlaceholderFormat(sq.Dollar)

			query, args, err = insertTag.ToSql()
			if err != nil {
				return fmt.Errorf("failed to build insert notifications_tags query: %w", err)
			}
			if _, err := tx.Exec(ctx, query, args...); err != nil {
				return fmt.Errorf("failed to link notification to tag: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return notificationID, nil
}

func (r *Repository) upsertTag(ctx context.Context, tx pgx.Tx, tag string) (int, error) {
	selectTag := sq.Select("id").
		From("tags").
		Where(sq.Eq{"tag": tag}).
		PlaceholderFormat(sq.Dollar)

	query, args, err := selectTag.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build select tag query: %w", err)
	}

	var tagID int
	err = tx.QueryRow(ctx, query, args...).Scan(&tagID)
	if err == nil {
		return tagID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("failed to find tag: %w", err)
	}

	insertTag := sq.Insert("tags").
		Columns("tag").
		Values(tag).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)

	query, args, err = insertTag.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build insert tag query: %w", err)
	}
	if err := tx.QueryRow(ctx, query, args...).Scan(&tagID); err != nil {
		return 0, fmt.Errorf("failed to insert tag: %w", err)
	}

	return tagID, nil
}

func (r *Repository) AddNotifyTasks(ctx context.Context, notificationID int) error {
	recipients := sq.Select("uh.user_id").
		Column("? AS notification_id", notificationID).
		Distinct().
		From("clusters_houses ch").
		Join("users_houses uh ON uh.house_id = ch.house_id").
		Join("users u ON u.id = uh.user_id AND u.active = true").
		Where("ch.cluster_id = (SELECT cluster_id FROM notifications WHERE id = ?)", notificationID).
		Where(`NOT EXISTS (
			SELECT 1
			FROM notifications_tags nt
			JOIN users_tags ut ON ut.tag_id = nt.tag_id AND ut.user_id = uh.user_id AND ut.excluded = true
			WHERE nt.notification_id = ?
		)`, notificationID)

	insertTasks := sq.Insert("notifications_tasks").
		Columns("user_id", "notification_id").
		Select(recipients).
		Suffix("ON CONFLICT (user_id, notification_id) DO NOTHING").
		PlaceholderFormat(sq.Dollar)

	query, args, err := insertTasks.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to add notify tasks: %w", err)
	}

	return nil
}

func (r *Repository) AckBatch(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	updateTasks := sq.Update("notifications_tasks").
		Set("status", notificationTaskStatusSent).
		Where(sq.Eq{"task_id": ids}).
		PlaceholderFormat(sq.Dollar)

	query, args, err := updateTasks.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to ack batch: %w", err)
	}

	return nil
}

func (r *Repository) Update(ctx context.Context, update models.UserUpdate) error {
	builder := sq.Update("users").
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": update.Id}).
		PlaceholderFormat(sq.Dollar)

	if update.Name != nil {
		builder = builder.Set("user_name", *update.Name)
	}
	if update.Active != nil {
		builder = builder.Set("active", *update.Active)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *Repository) GetIDByUserID(ctx context.Context, userID int) (int, error) {
	selectOrg := sq.Select("org_id").
		From("users_organizations").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("created_at ASC").
		Limit(1).
		PlaceholderFormat(sq.Dollar)

	query, args, err := selectOrg.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build query: %w", err)
	}

	var orgID int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&orgID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("user %d is not a member of any organization", userID)
		}
		return 0, fmt.Errorf("failed to find organization: %w", err)
	}

	return orgID, nil
}

func (r *Repository) GetOrgMemberMe(ctx context.Context, userID int) (*models.OrgMemberMe, error) {
	selectMember := sq.Select(
		"u.id", "u.user_name", "u.active",
		"uo.org_id", "uo.role",
		"o.id", "o.name", "o.description",
	).
		From("users_organizations uo").
		Join("users u ON u.id = uo.user_id").
		Join("organizations o ON o.id = uo.org_id").
		Where(sq.Eq{"uo.user_id": userID}).
		OrderBy("uo.created_at ASC").
		Limit(1).
		PlaceholderFormat(sq.Dollar)

	query, args, err := selectMember.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	result := &models.OrgMemberMe{}
	var description *string

	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&result.OrgMember.User.Id, &result.OrgMember.User.Name, &result.OrgMember.User.Active,
		&result.OrgMember.OrgID, &result.OrgMember.Role,
		&result.Org.Id, &result.Org.Name, &description,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user %d is not a member of any organization", userID)
		}
		return nil, fmt.Errorf("failed to find org member: %w", err)
	}
	if description != nil {
		result.Org.Description = *description
	}

	selectCluster := sq.Select("c.name", "COUNT(ch.house_id)").
		From("clusters c").
		LeftJoin("clusters_houses ch ON ch.cluster_id = c.id").
		Where(sq.Eq{"c.org_id": result.OrgMember.OrgID}).
		GroupBy("c.id", "c.name").
		OrderBy("c.created_at ASC").
		Limit(1).
		PlaceholderFormat(sq.Dollar)

	query, args, err = selectCluster.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build cluster stat query: %w", err)
	}

	err = r.pool.QueryRow(ctx, query, args...).Scan(&result.ClusterStat.Name, &result.ClusterStat.HousesNumber)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to find cluster stat: %w", err)
	}

	return result, nil
}

func (r *Repository) CreateClusterWithHouses(ctx context.Context, cluster *models.Cluster) error {
	return r.runTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		insertCluster := sq.Insert("clusters").
			Columns("org_id", "name").
			Values(cluster.OrgID, cluster.Name).
			Suffix("RETURNING id").
			PlaceholderFormat(sq.Dollar)

		query, args, err := insertCluster.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build insert cluster query: %w", err)
		}

		var clusterID int
		if err := tx.QueryRow(ctx, query, args...).Scan(&clusterID); err != nil {
			return fmt.Errorf("failed to insert cluster: %w", err)
		}

		for _, house := range cluster.Houses {
			houseID, err := r.upsertHouse(ctx, tx, house)
			if err != nil {
				return fmt.Errorf("failed to upsert house %q: %w", house.Address, err)
			}

			insertLink := sq.Insert("clusters_houses").
				Columns("cluster_id", "house_id").
				Values(clusterID, houseID).
				Suffix("ON CONFLICT (cluster_id, house_id) DO NOTHING").
				PlaceholderFormat(sq.Dollar)

			query, args, err = insertLink.ToSql()
			if err != nil {
				return fmt.Errorf("failed to build insert clusters_houses query: %w", err)
			}
			if _, err := tx.Exec(ctx, query, args...); err != nil {
				return fmt.Errorf("failed to link cluster to house: %w", err)
			}
		}

		return nil
	})
}

func (r *Repository) upsertHouse(ctx context.Context, tx pgx.Tx, house *models.House) (int, error) {
	selectHouse := sq.Select("id").
		From("houses").
		Where(sq.Eq{"addr": house.Address}).
		PlaceholderFormat(sq.Dollar)

	query, args, err := selectHouse.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build select house query: %w", err)
	}

	var houseID int
	err = tx.QueryRow(ctx, query, args...).Scan(&houseID)
	if err == nil {
		return houseID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("failed to find house: %w", err)
	}

	insertHouse := sq.Insert("houses").
		Columns("addr", "location").
		Values(house.Address, sq.Expr("ST_SetSRID(ST_MakePoint(?, ?), 4326)", house.Longitude, house.Latitude)).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)

	query, args, err = insertHouse.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build insert house query: %w", err)
	}
	if err := tx.QueryRow(ctx, query, args...).Scan(&houseID); err != nil {
		return 0, fmt.Errorf("failed to insert house: %w", err)
	}

	return houseID, nil
}

func (r *Repository) GetClusterNamesByOrgID(ctx context.Context, orgID int, cursor string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = clusterNamesDefaultLimit
	}

	builder := sq.Select("name").
		From("clusters").
		Where(sq.Eq{"org_id": orgID}).
		OrderBy("name ASC").
		Limit(uint64(limit)).
		PlaceholderFormat(sq.Dollar)

	if cursor != "" {
		builder = builder.Where(sq.Gt{"name": cursor})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query cluster names: %w", err)
	}
	defer rows.Close()

	names := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan cluster name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return names, nil
}

func (r *Repository) Write(event events.Event) {
	select {
	case r.ch <- event:
	case <-time.After(r.cfg.WriteTimeout):
	}
}

func (r *Repository) InitLogger(ctx context.Context) {
	go r.readEvents(ctx)
}

func (r *Repository) readEvents(ctx context.Context) {
	batch := make([]events.Event, 0, r.cfg.BatchCap)

	var flushBatch = func() {
		if len(batch) == 0 {
			return
		}

		flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.cfg.FlushTimeout)
		defer cancel()

		if err := r.sendBatch(flushCtx, batch); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to send event batch")
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flushBatch()
			return
		case event, ok := <-r.ch:
			if !ok {
				flushBatch()
				return
			}

			batch = append(batch, event)
			if len(batch) >= r.cfg.BatchCap {
				flushBatch()
			}
		case <-time.Tick(r.cfg.FlushTimeout):
			flushBatch()
		}
	}
}

func (r *Repository) sendBatch(ctx context.Context, batch []events.Event) error {
	if len(batch) == 0 {
		return nil
	}

	objects := make([]string, 0, len(batch))
	for _, event := range batch {
		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}
		objects = append(objects, string(data))
	}

	builder := sq.Insert("event_log").
		Columns("json_data").
		PlaceholderFormat(sq.Dollar)

	for _, obj := range objects {
		builder = builder.Values(obj)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to exec batch insert: %w", err)
	}

	return nil
}

func (r *Repository) runTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			return fmt.Errorf("failed to rollback: %w %w", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}
