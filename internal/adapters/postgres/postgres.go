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
			Values(inhabitant.Id, inhabitant.Name).
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

		insertLink := sq.Insert("user_houses").
			Columns("user_id", "house_id").
			Values(inhabitant.Id, houseID).
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
			// токен уже использован параллельным запросом
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
