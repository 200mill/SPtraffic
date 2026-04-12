package service

import (
	"context"
	"fmt"

	"github.com/sptraffic/backend/internal/domain"
	"github.com/sptraffic/backend/internal/repository"
)

// RouteSchedule bundles a route with its origin, destination, and schedules.
type RouteSchedule struct {
	Route     domain.Route      `json:"route"`
	Origin    domain.Terminal   `json:"origin"`
	Dest      domain.Terminal   `json:"destination"`
	Schedules []domain.Schedule `json:"schedules"`
}

// ScheduleService retrieves full timetable data for a given route.
type ScheduleService struct {
	routeRepo repository.RouteRepository
	termRepo  repository.TerminalRepository
	schedRepo repository.ScheduleRepository
}

func NewScheduleService(
	routeRepo repository.RouteRepository,
	termRepo repository.TerminalRepository,
	schedRepo repository.ScheduleRepository,
) *ScheduleService {
	return &ScheduleService{routeRepo: routeRepo, termRepo: termRepo, schedRepo: schedRepo}
}

func (s *ScheduleService) GetByRoute(ctx context.Context, routeID int) (*RouteSchedule, error) {
	route, err := s.routeRepo.FindByID(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("route %d not found", routeID)
	}

	origin, err := s.termRepo.FindByID(ctx, route.OriginID)
	if err != nil {
		return nil, fmt.Errorf("origin terminal %d not found", route.OriginID)
	}

	dest, err := s.termRepo.FindByID(ctx, route.DestID)
	if err != nil {
		return nil, fmt.Errorf("destination terminal %d not found", route.DestID)
	}

	scheds, err := s.schedRepo.FindByRoute(ctx, routeID)
	if err != nil {
		return nil, err
	}

	return &RouteSchedule{
		Route:     *route,
		Origin:    *origin,
		Dest:      *dest,
		Schedules: scheds,
	}, nil
}
