package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sptraffic/backend/internal/domain"
)

// ScheduleRepository abstracts schedule persistence.
type ScheduleRepository interface {
	FindByRoute(ctx context.Context, routeID int) ([]domain.Schedule, error)
	// FindByRouteFrom returns schedules on routeID in chronological order starting
	// from afterMins, wrapping past midnight so overnight routes are included.
	FindByRouteFrom(ctx context.Context, routeID int, afterMins int) ([]domain.Schedule, error)
	Insert(ctx context.Context, s *domain.Schedule) error
	DeleteAll(ctx context.Context) error
}

type pgScheduleRepo struct{ db *pgxpool.Pool }

func NewScheduleRepository(db *pgxpool.Pool) ScheduleRepository {
	return &pgScheduleRepo{db: db}
}

const schedCols = `id, route_id, departure_mins, COALESCE(arrival_mins,0),
                   COALESCE(duration_min,0), days_of_week`

func (r *pgScheduleRepo) FindByRoute(ctx context.Context, routeID int) ([]domain.Schedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+schedCols+`
		 FROM schedules WHERE route_id = $1 ORDER BY departure_mins`,
		routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSchedules(rows)
}

// FindByRouteFrom returns schedules sorted chronologically from afterMins.
// Schedules before afterMins are treated as "next day" (departure_mins + 1440)
// so that a search at 23:00 also surfaces early-morning services the next day.
func (r *pgScheduleRepo) FindByRouteFrom(ctx context.Context, routeID int, afterMins int) ([]domain.Schedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+schedCols+`
		 FROM schedules
		 WHERE route_id = $1
		 ORDER BY CASE
		     WHEN departure_mins >= $2 THEN departure_mins
		     ELSE departure_mins + 1440
		 END
		 LIMIT 60`,
		routeID, afterMins)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSchedules(rows)
}

func (r *pgScheduleRepo) Insert(ctx context.Context, s *domain.Schedule) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO schedules
		   (route_id, departure_mins, arrival_mins, duration_min, days_of_week, valid_from, valid_until)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.RouteID, s.DepartureMins, s.ArrivalMins, s.DurationMin,
		s.DaysOfWeek, s.ValidFrom, s.ValidUntil)
	return err
}

func (r *pgScheduleRepo) DeleteAll(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM schedules`)
	return err
}

func collectSchedules(rows pgx.Rows) ([]domain.Schedule, error) {
	var out []domain.Schedule
	for rows.Next() {
		var s domain.Schedule
		if err := rows.Scan(&s.ID, &s.RouteID, &s.DepartureMins, &s.ArrivalMins,
			&s.DurationMin, &s.DaysOfWeek); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
