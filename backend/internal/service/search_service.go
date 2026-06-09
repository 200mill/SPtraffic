package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/sptraffic/backend/internal/domain"
	"github.com/sptraffic/backend/internal/repository"
)

const (
	minTransferWaitMins = 30
	maxResults          = 20
)

// SearchService finds direct and one-transfer journeys between two terminals.
type SearchService struct {
	termRepo  repository.TerminalRepository
	routeRepo repository.RouteRepository
	schedRepo repository.ScheduleRepository
}

func NewSearchService(
	termRepo repository.TerminalRepository,
	routeRepo repository.RouteRepository,
	schedRepo repository.ScheduleRepository,
) *SearchService {
	return &SearchService{termRepo: termRepo, routeRepo: routeRepo, schedRepo: schedRepo}
}

func (s *SearchService) Search(ctx context.Context, q domain.SearchQuery) ([]domain.SearchResult, error) {
	from, err := s.termRepo.FindByCode(ctx, q.FromCode)
	if err != nil {
		return nil, fmt.Errorf("departure terminal %q not found", q.FromCode)
	}
	to, err := s.termRepo.FindByCode(ctx, q.ToCode)
	if err != nil {
		return nil, fmt.Errorf("arrival terminal %q not found", q.ToCode)
	}

	afterMins := q.Date.Hour()*60 + q.Date.Minute()
	weekday := q.Date.Weekday()

	var results []domain.SearchResult

	// 1. Direct routes
	directRoutes, err := s.routeRepo.FindByOriginAndDest(ctx, from.ID, to.ID)
	if err != nil {
		return nil, err
	}
	for _, rt := range directRoutes {
		if !modeMatches(rt.Type, q.Mode) {
			continue
		}
		scheds, err := s.schedRepo.FindByRouteFrom(ctx, rt.ID, afterMins)
		if err != nil {
			continue
		}
		for _, sc := range scheds {
			if !sc.RunsOn(weekday) {
				continue
			}
			results = append(results, domain.SearchResult{
				Legs: []domain.Leg{{
					From:          *from,
					To:            *to,
					DepartureMins: sc.DepartureMins,
					ArrivalMins:   sc.ArrivalMins,
					RouteType:     string(rt.Type),
					Operator:      rt.Operator,
				}},
				TotalMinutes: sc.DurationMin,
				Transfers:    0,
			})
		}
	}

	// 2. One-transfer routes
	if q.MaxLegs >= 2 {
		transfers, err := s.oneTransfer(ctx, from, to, afterMins, weekday, q.Mode)
		if err == nil {
			results = append(results, transfers...)
		}
	}

	// 3. Two-transfer routes
	if q.MaxLegs >= 3 {
		transfers, err := s.twoTransfer(ctx, from, to, afterMins, weekday, q.Mode)
		if err == nil {
			results = append(results, transfers...)
		}
	}

	// Sort: direct first, then total time, then departure time
	sort.Slice(results, func(i, j int) bool {
		ri, rj := results[i], results[j]
		if ri.Transfers != rj.Transfers {
			return ri.Transfers < rj.Transfers
		}
		if ri.TotalMinutes != rj.TotalMinutes {
			return ri.TotalMinutes < rj.TotalMinutes
		}
		return ri.Legs[0].DepartureMins < rj.Legs[0].DepartureMins
	})

	if len(results) > maxResults {
		results = results[:maxResults]
	}
	return results, nil
}

func (s *SearchService) oneTransfer(
	ctx context.Context,
	from, to *domain.Terminal,
	afterMins int,
	weekday time.Weekday,
	mode string,
) ([]domain.SearchResult, error) {
	// Routes departing from origin
	leg1Routes, err := s.routeRepo.FindByOrigin(ctx, from.ID)
	if err != nil {
		return nil, err
	}
	// Routes arriving at destination
	leg2Routes, err := s.routeRepo.FindByDest(ctx, to.ID)
	if err != nil {
		return nil, err
	}

	// Index leg1 routes by their destination terminal ID
	leg1ByDest := make(map[int][]domain.Route)
	for _, rt := range leg1Routes {
		if !modeMatches(rt.Type, mode) {
			continue
		}
		leg1ByDest[rt.DestID] = append(leg1ByDest[rt.DestID], rt)
	}

	var results []domain.SearchResult

	for _, r2 := range leg2Routes {
		if !modeMatches(r2.Type, mode) {
			continue
		}
		midID := r2.OriginID
		r1List, ok := leg1ByDest[midID]
		if !ok {
			continue
		}

		mid, err := s.termRepo.FindByID(ctx, midID)
		if err != nil {
			continue
		}

		scheds2, err := s.schedRepo.FindByRoute(ctx, r2.ID)
		if err != nil {
			continue
		}

		for _, r1 := range r1List {
			scheds1, err := s.schedRepo.FindByRouteFrom(ctx, r1.ID, afterMins)
			if err != nil {
				continue
			}

			for _, s1 := range scheds1 {
				if !s1.RunsOn(weekday) {
					continue
				}
				for _, s2 := range scheds2 {
					if !s2.RunsOn(weekday) {
						continue
					}
					// Transfer wait must be at least minTransferWaitMins.
					// Apply overnight wrap so crossing midnight gives a positive value.
					wait := s2.DepartureMins - s1.ArrivalMins
					if wait < 0 {
						wait += 24 * 60
					}
					if wait < minTransferWaitMins {
						continue
					}
					total := s2.ArrivalMins - s1.DepartureMins
					if total <= 0 {
						total += 24 * 60
					}
					results = append(results, domain.SearchResult{
						Legs: []domain.Leg{
							{
								From:          *from,
								To:            *mid,
								DepartureMins: s1.DepartureMins,
								ArrivalMins:   s1.ArrivalMins,
								RouteType:     string(r1.Type),
								Operator:      r1.Operator,
							},
							{
								From:          *mid,
								To:            *to,
								DepartureMins: s2.DepartureMins,
								ArrivalMins:   s2.ArrivalMins,
								RouteType:     string(r2.Type),
								Operator:      r2.Operator,
							},
						},
						TotalMinutes: total,
						Transfers:    1,
					})
				}
			}
		}
	}
	return results, nil
}

// twoTransfer finds journeys with exactly two transfers (three legs).
// Strategy: enumerate mid1 candidates from leg1, look up mid1→* routes,
// intersect destinations with terminals that have a direct route to `to`.
func (s *SearchService) twoTransfer(
	ctx context.Context,
	from, to *domain.Terminal,
	afterMins int,
	weekday time.Weekday,
	mode string,
) ([]domain.SearchResult, error) {
	leg1Routes, err := s.routeRepo.FindByOrigin(ctx, from.ID)
	if err != nil {
		return nil, err
	}
	leg3Routes, err := s.routeRepo.FindByDest(ctx, to.ID)
	if err != nil {
		return nil, err
	}

	// Index leg3 routes by their origin (= mid2)
	leg3ByOrigin := make(map[int][]domain.Route)
	for _, rt := range leg3Routes {
		if modeMatches(rt.Type, mode) {
			leg3ByOrigin[rt.OriginID] = append(leg3ByOrigin[rt.OriginID], rt)
		}
	}

	// Group leg1 routes by dest (= mid1)
	leg1ByDest := make(map[int][]domain.Route)
	for _, rt := range leg1Routes {
		if modeMatches(rt.Type, mode) {
			leg1ByDest[rt.DestID] = append(leg1ByDest[rt.DestID], rt)
		}
	}

	var results []domain.SearchResult

	for mid1ID, r1List := range leg1ByDest {
		// Find routes departing from mid1
		leg2Routes, err := s.routeRepo.FindByOrigin(ctx, mid1ID)
		if err != nil {
			continue
		}

		// Keep only leg2 routes whose destination is a valid mid2
		for _, r2 := range leg2Routes {
			if !modeMatches(r2.Type, mode) {
				continue
			}
			mid2ID := r2.DestID
			r3List, ok := leg3ByOrigin[mid2ID]
			if !ok {
				continue
			}
			// Don't allow mid1 == mid2 or mid == from/to
			if mid2ID == mid1ID || mid2ID == from.ID || mid1ID == to.ID {
				continue
			}

			mid1, err := s.termRepo.FindByID(ctx, mid1ID)
			if err != nil {
				continue
			}
			mid2, err := s.termRepo.FindByID(ctx, mid2ID)
			if err != nil {
				continue
			}

			scheds2, err := s.schedRepo.FindByRoute(ctx, r2.ID)
			if err != nil {
				continue
			}

			for _, r3 := range r3List {
				scheds3, err := s.schedRepo.FindByRoute(ctx, r3.ID)
				if err != nil {
					continue
				}

				for _, r1 := range r1List {
					scheds1, err := s.schedRepo.FindByRouteFrom(ctx, r1.ID, afterMins)
					if err != nil {
						continue
					}

					for _, s1 := range scheds1 {
						if !s1.RunsOn(weekday) {
							continue
						}
						for _, s2 := range scheds2 {
							if !s2.RunsOn(weekday) {
								continue
							}
							wait1 := s2.DepartureMins - s1.ArrivalMins
							if wait1 < 0 {
								wait1 += 24 * 60
							}
							if wait1 < minTransferWaitMins {
								continue
							}

							for _, s3 := range scheds3 {
								if !s3.RunsOn(weekday) {
									continue
								}
								wait2 := s3.DepartureMins - s2.ArrivalMins
								if wait2 < 0 {
									wait2 += 24 * 60
								}
								if wait2 < minTransferWaitMins {
									continue
								}
								total := s3.ArrivalMins - s1.DepartureMins
								if total <= 0 {
									total += 24 * 60
								}
								results = append(results, domain.SearchResult{
									Legs: []domain.Leg{
										{
											From:          *from,
											To:            *mid1,
											DepartureMins: s1.DepartureMins,
											ArrivalMins:   s1.ArrivalMins,
											RouteType:     string(r1.Type),
											Operator:      r1.Operator,
										},
										{
											From:          *mid1,
											To:            *mid2,
											DepartureMins: s2.DepartureMins,
											ArrivalMins:   s2.ArrivalMins,
											RouteType:     string(r2.Type),
											Operator:      r2.Operator,
										},
										{
											From:          *mid2,
											To:            *to,
											DepartureMins: s3.DepartureMins,
											ArrivalMins:   s3.ArrivalMins,
											RouteType:     string(r3.Type),
											Operator:      r3.Operator,
										},
									},
									TotalMinutes: total,
									Transfers:    2,
								})
								if len(results) >= maxResults {
									return results, nil
								}
							}
						}
					}
				}
			}
		}
	}
	return results, nil
}

func modeMatches(routeType domain.RouteType, mode string) bool {
	switch mode {
	case "bus":
		return routeType == domain.RouteTypeExpress || routeType == domain.RouteTypeIntercity
	case "rail":
		return routeType == domain.RouteTypeRail
	case "metro":
		return routeType == domain.RouteTypeMetro
	default: // "all"
		return true
	}
}
