package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sptraffic/backend/internal/domain"
)

// TerminalRepository abstracts terminal/station persistence.
type TerminalRepository interface {
	FindByID(ctx context.Context, id int) (*domain.Terminal, error)
	FindByCode(ctx context.Context, code string) (*domain.Terminal, error)
	FindAll(ctx context.Context) ([]domain.Terminal, error)
	// FindAllPaginated returns a page of terminals ordered by name.
	FindAllPaginated(ctx context.Context, limit, offset int) ([]domain.Terminal, int, error)
	FindByRegion(ctx context.Context, regionCode string) ([]domain.Terminal, error)
	// SearchByName returns terminals whose name contains the query string (case-insensitive).
	SearchByName(ctx context.Context, query string) ([]domain.Terminal, error)
	FindRegions(ctx context.Context) ([]string, error)
	Upsert(ctx context.Context, t *domain.Terminal) error
	DeleteAll(ctx context.Context) error
}

type pgTerminalRepo struct{ db *pgxpool.Pool }

func NewTerminalRepository(db *pgxpool.Pool) TerminalRepository {
	return &pgTerminalRepo{db: db}
}

const terminalCols = `id, code, name, type, region_code, lat, lon`

func scanTerminal(row pgx.Row) (*domain.Terminal, error) {
	t := &domain.Terminal{}
	err := row.Scan(&t.ID, &t.Code, &t.Name, &t.Type, &t.RegionCode, &t.Lat, &t.Lon)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *pgTerminalRepo) FindByID(ctx context.Context, id int) (*domain.Terminal, error) {
	return scanTerminal(r.db.QueryRow(ctx,
		`SELECT `+terminalCols+` FROM terminals WHERE id = $1`, id))
}

func (r *pgTerminalRepo) FindByCode(ctx context.Context, code string) (*domain.Terminal, error) {
	return scanTerminal(r.db.QueryRow(ctx,
		`SELECT `+terminalCols+` FROM terminals WHERE code = $1`, code))
}

func (r *pgTerminalRepo) FindAll(ctx context.Context) ([]domain.Terminal, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+terminalCols+` FROM terminals ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectTerminals(rows)
}

func (r *pgTerminalRepo) FindAllPaginated(ctx context.Context, limit, offset int) ([]domain.Terminal, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM terminals`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+terminalCols+` FROM terminals ORDER BY name LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out, err := collectTerminals(rows)
	return out, total, err
}

func (r *pgTerminalRepo) FindByRegion(ctx context.Context, regionCode string) ([]domain.Terminal, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+terminalCols+` FROM terminals WHERE region_code = $1 ORDER BY name`,
		regionCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectTerminals(rows)
}

func (r *pgTerminalRepo) SearchByName(ctx context.Context, query string) ([]domain.Terminal, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+terminalCols+` FROM terminals
		 WHERE name ILIKE '%' || $1 || '%'
		 ORDER BY name LIMIT 20`,
		query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectTerminals(rows)
}

func (r *pgTerminalRepo) FindRegions(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT region_code FROM terminals ORDER BY region_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regions []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		regions = append(regions, code)
	}
	return regions, rows.Err()
}

func (r *pgTerminalRepo) Upsert(ctx context.Context, t *domain.Terminal) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO terminals (code, name, type, region_code, lat, lon)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (code) DO UPDATE SET
		   name        = EXCLUDED.name,
		   type        = EXCLUDED.type,
		   region_code = EXCLUDED.region_code,
		   lat         = EXCLUDED.lat,
		   lon         = EXCLUDED.lon`,
		t.Code, t.Name, t.Type, t.RegionCode, t.Lat, t.Lon)
	return err
}

func (r *pgTerminalRepo) DeleteAll(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM terminals`)
	return err
}

func collectTerminals(rows pgx.Rows) ([]domain.Terminal, error) {
	var out []domain.Terminal
	for rows.Next() {
		var t domain.Terminal
		if err := rows.Scan(&t.ID, &t.Code, &t.Name, &t.Type, &t.RegionCode, &t.Lat, &t.Lon); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
