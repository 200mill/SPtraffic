package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sptraffic/backend/internal/domain"
)

// RouteRepository abstracts route persistence.
type RouteRepository interface {
	FindByID(ctx context.Context, id int) (*domain.Route, error)
	FindByOriginAndDest(ctx context.Context, originID, destID int) ([]domain.Route, error)
	FindByOrigin(ctx context.Context, originID int) ([]domain.Route, error)
	FindByDest(ctx context.Context, destID int) ([]domain.Route, error)
	// Insert inserts a new route and returns its generated ID.
	Insert(ctx context.Context, r *domain.Route) (int, error)
	DeleteAll(ctx context.Context) error
}

type pgRouteRepo struct{ db *pgxpool.Pool }

func NewRouteRepository(db *pgxpool.Pool) RouteRepository {
	return &pgRouteRepo{db: db}
}

const routeSelect = `
SELECT id, COALESCE(code,''), type, origin_id, dest_id, COALESCE(operator,'')
FROM routes`

func (r *pgRouteRepo) FindByID(ctx context.Context, id int) (*domain.Route, error) {
	row := r.db.QueryRow(ctx, routeSelect+` WHERE id = $1`, id)
	return scanRoute(row)
}

func (r *pgRouteRepo) FindByOriginAndDest(ctx context.Context, originID, destID int) ([]domain.Route, error) {
	rows, err := r.db.Query(ctx,
		routeSelect+` WHERE origin_id = $1 AND dest_id = $2`, originID, destID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRoutes(rows)
}

func (r *pgRouteRepo) FindByOrigin(ctx context.Context, originID int) ([]domain.Route, error) {
	rows, err := r.db.Query(ctx, routeSelect+` WHERE origin_id = $1`, originID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRoutes(rows)
}

func (r *pgRouteRepo) FindByDest(ctx context.Context, destID int) ([]domain.Route, error) {
	rows, err := r.db.Query(ctx, routeSelect+` WHERE dest_id = $1`, destID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRoutes(rows)
}

func (r *pgRouteRepo) Insert(ctx context.Context, route *domain.Route) (int, error) {
	var id int
	var err error
	if route.Code == "" {
		err = r.db.QueryRow(ctx,
			`INSERT INTO routes (type, origin_id, dest_id, operator)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			route.Type, route.OriginID, route.DestID, route.Operator,
		).Scan(&id)
	} else {
		err = r.db.QueryRow(ctx,
			`INSERT INTO routes (code, type, origin_id, dest_id, operator)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (code) DO UPDATE SET
			   type      = EXCLUDED.type,
			   origin_id = EXCLUDED.origin_id,
			   dest_id   = EXCLUDED.dest_id,
			   operator  = EXCLUDED.operator
			 RETURNING id`,
			route.Code, route.Type, route.OriginID, route.DestID, route.Operator,
		).Scan(&id)
	}
	return id, err
}

func (r *pgRouteRepo) DeleteAll(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM routes`)
	return err
}

func scanRoute(row pgx.Row) (*domain.Route, error) {
	rt := &domain.Route{}
	err := row.Scan(&rt.ID, &rt.Code, &rt.Type, &rt.OriginID, &rt.DestID, &rt.Operator)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func collectRoutes(rows pgx.Rows) ([]domain.Route, error) {
	var out []domain.Route
	for rows.Next() {
		var rt domain.Route
		if err := rows.Scan(&rt.ID, &rt.Code, &rt.Type, &rt.OriginID, &rt.DestID, &rt.Operator); err != nil {
			return nil, err
		}
		out = append(out, rt)
	}
	return out, rows.Err()
}
