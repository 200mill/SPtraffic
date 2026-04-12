package ingestion

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sptraffic/backend/internal/domain"
	"github.com/sptraffic/backend/internal/ingestion/client"
	"github.com/sptraffic/backend/internal/ingestion/parser"
	"github.com/sptraffic/backend/internal/repository"
)

// Scheduler runs daily data ingestion from data.go.kr.
type Scheduler struct {
	c         *cron.Cron
	apiClient *client.Client
	termRepo  repository.TerminalRepository
	routeRepo repository.RouteRepository
	schedRepo repository.ScheduleRepository
}

func NewScheduler(
	apiClient *client.Client,
	termRepo repository.TerminalRepository,
	routeRepo repository.RouteRepository,
	schedRepo repository.ScheduleRepository,
) *Scheduler {
	return &Scheduler{
		c:         cron.New(cron.WithLocation(mustLoadKST())),
		apiClient: apiClient,
		termRepo:  termRepo,
		routeRepo: routeRepo,
		schedRepo: schedRepo,
	}
}

// Start registers the daily 02:00 KST job and starts the cron runner.
func (s *Scheduler) Start() {
	s.c.AddFunc("0 2 * * *", func() {
		log.Println("[ingestion] daily refresh started")
		if err := s.RunAll(context.Background()); err != nil {
			log.Printf("[ingestion] refresh error: %v", err)
		} else {
			log.Println("[ingestion] daily refresh complete")
		}
	})
	s.c.Start()
}

// Stop gracefully shuts down the cron runner.
func (s *Scheduler) Stop() { s.c.Stop() }

// RunAll executes a full data refresh: terminals → routes → schedules.
// It truncates all tables first (schedules → routes → terminals cascade).
func (s *Scheduler) RunAll(ctx context.Context) error {
	log.Println("[ingestion] truncating tables")
	if err := s.schedRepo.DeleteAll(ctx); err != nil {
		return fmt.Errorf("truncate schedules: %w", err)
	}
	if err := s.routeRepo.DeleteAll(ctx); err != nil {
		return fmt.Errorf("truncate routes: %w", err)
	}
	if err := s.termRepo.DeleteAll(ctx); err != nil {
		return fmt.Errorf("truncate terminals: %w", err)
	}

	log.Println("[ingestion] fetching express bus terminals")
	if err := s.fetchExpressBusTerminals(ctx); err != nil {
		log.Printf("[ingestion] express bus terminals error: %v", err)
	}

	log.Println("[ingestion] fetching intercity bus terminals")
	if err := s.fetchIntercityTerminals(ctx); err != nil {
		log.Printf("[ingestion] intercity bus terminals error: %v", err)
	}

	log.Println("[ingestion] fetching rail stations")
	if err := s.fetchRailStations(ctx); err != nil {
		log.Printf("[ingestion] rail stations error: %v", err)
	}

	today := time.Now().In(mustLoadKST()).Format("20060102")

	log.Println("[ingestion] fetching express bus schedules")
	if err := s.fetchExpressBusSchedules(ctx, today); err != nil {
		log.Printf("[ingestion] express bus schedules error: %v", err)
	}

	log.Println("[ingestion] fetching intercity bus schedules")
	if err := s.fetchIntercitySchedules(ctx, today); err != nil {
		log.Printf("[ingestion] intercity bus schedules error: %v", err)
	}

	log.Println("[ingestion] fetching rail schedules")
	if err := s.fetchRailSchedules(ctx, today); err != nil {
		log.Printf("[ingestion] rail schedules error: %v", err)
	}

	return nil
}

// ─── Express Bus ─────────────────────────────────────────────────────────────
// Base URL: https://apis.data.go.kr/1613000/ExpBusInfo

func (s *Scheduler) fetchExpressBusTerminals(ctx context.Context) error {
	params := url.Values{"numOfRows": {"1000"}, "pageNo": {"1"}}
	body, err := s.apiClient.Get(ctx, "/1613000/ExpBusInfo/GetExpBusTrminlList", params)
	if err != nil {
		return err
	}
	resp, err := client.Parse(body)
	if err != nil {
		return err
	}
	terminals, err := parser.ParseExpressBusTerminals(resp.Response.Body.Items)
	if err != nil {
		return err
	}
	for i := range terminals {
		if err := s.termRepo.Upsert(ctx, &terminals[i]); err != nil {
			log.Printf("[ingestion] upsert express terminal %s: %v", terminals[i].Code, err)
		}
	}
	log.Printf("[ingestion] upserted %d express bus terminals", len(terminals))
	return nil
}

// fetchExpressBusSchedules queries schedules for every terminal-pair combination.
// The TAGO ExpBusInfo API requires both depTerminalId and arrTerminalId, so we
// iterate all pairs. Korea has ~30–50 express bus terminals, making this feasible.
func (s *Scheduler) fetchExpressBusSchedules(ctx context.Context, today string) error {
	// Load all express bus terminals that were just inserted
	allTerminals, err := s.termRepo.FindAll(ctx)
	if err != nil {
		return err
	}
	var busTerminals []domain.Terminal
	for _, t := range allTerminals {
		if t.Type == domain.TerminalTypeBus {
			busTerminals = append(busTerminals, t)
		}
	}

	total := 0
	for _, dep := range busTerminals {
		for _, arr := range busTerminals {
			if dep.ID == arr.ID {
				continue
			}
			params := url.Values{
				"depTerminalId": {dep.Code},
				"arrTerminalId": {arr.Code},
				"depPlandTime":  {today},
				"numOfRows":     {"100"},
				"pageNo":        {"1"},
			}
			body, err := s.apiClient.Get(ctx,
				"/1613000/ExpBusInfo/GetStrtpntAlocFndExpbusInfo", params)
			if err != nil {
				continue
			}
			resp, err := client.Parse(body)
			if err != nil {
				continue // no routes for this pair — normal
			}
			schedules, err := parser.ParseExpressBusSchedules(resp.Response.Body.Items)
			if err != nil || len(schedules) == 0 {
				continue
			}

			// Use the grade (operator) from the first schedule as the route's operator.
			routeOperator := ""
			if len(schedules) > 0 {
				routeOperator = schedules[0].Operator
			}
			routeID, err := s.routeRepo.Insert(ctx, &domain.Route{
				Type:     domain.RouteTypeExpress,
				OriginID: dep.ID,
				DestID:   arr.ID,
				Operator: routeOperator,
			})
			if err != nil {
				continue
			}
			for _, sc := range schedules {
				dur := sc.ArrMins - sc.DepMins
				if dur < 0 {
					dur += 24 * 60
				}
				_ = s.schedRepo.Insert(ctx, &domain.Schedule{
					RouteID:       routeID,
					DepartureMins: sc.DepMins,
					ArrivalMins:   sc.ArrMins,
					DurationMin:   dur,
					DaysOfWeek:    127, // all days
				})
			}
			total += len(schedules)
		}
	}
	log.Printf("[ingestion] inserted %d express bus schedules", total)
	return nil
}

// ─── Intercity Bus ────────────────────────────────────────────────────────────
// Base URL: https://apis.data.go.kr/1613000/SuburbsBusInfoService

func (s *Scheduler) fetchIntercityTerminals(ctx context.Context) error {
	params := url.Values{"numOfRows": {"1000"}, "pageNo": {"1"}}
	body, err := s.apiClient.Get(ctx,
		"/1613000/SuburbsBusInfoService/getSttnList", params)
	if err != nil {
		return err
	}
	resp, err := client.Parse(body)
	if err != nil {
		return err
	}
	terminals, err := parser.ParseIntercityTerminals(resp.Response.Body.Items)
	if err != nil {
		return err
	}
	for i := range terminals {
		if err := s.termRepo.Upsert(ctx, &terminals[i]); err != nil {
			log.Printf("[ingestion] upsert intercity terminal %s: %v", terminals[i].Code, err)
		}
	}
	log.Printf("[ingestion] upserted %d intercity bus terminals", len(terminals))
	return nil
}

// fetchIntercitySchedules iterates terminal pairs for the intercity bus API.
// Like the express bus API, getAlocFndSuberbsBusInfo requires both dep+arr IDs.
func (s *Scheduler) fetchIntercitySchedules(ctx context.Context, today string) error {
	allTerminals, err := s.termRepo.FindAll(ctx)
	if err != nil {
		return err
	}
	var busTerminals []domain.Terminal
	for _, t := range allTerminals {
		if t.Type == domain.TerminalTypeBus {
			busTerminals = append(busTerminals, t)
		}
	}

	total := 0
	for _, dep := range busTerminals {
		for _, arr := range busTerminals {
			if dep.ID == arr.ID {
				continue
			}
			params := url.Values{
				"depTerminalId": {dep.Code},
				"arrTerminalId": {arr.Code},
				"depPlandTime":  {today},
				"numOfRows":     {"100"},
				"pageNo":        {"1"},
			}
			body, err := s.apiClient.Get(ctx,
				"/1613000/SuburbsBusInfoService/getAlocFndSuberbsBusInfo", params)
			if err != nil {
				continue
			}
			resp, err := client.Parse(body)
			if err != nil {
				continue
			}
			schedules, err := parser.ParseIntercitySchedules(resp.Response.Body.Items)
			if err != nil || len(schedules) == 0 {
				continue
			}

			routeOperator := ""
			if len(schedules) > 0 {
				routeOperator = schedules[0].Operator
			}
			routeID, err := s.routeRepo.Insert(ctx, &domain.Route{
				Type:     domain.RouteTypeIntercity,
				OriginID: dep.ID,
				DestID:   arr.ID,
				Operator: routeOperator,
			})
			if err != nil {
				continue
			}
			for _, sc := range schedules {
				dur := sc.ArrMins - sc.DepMins
				if dur < 0 {
					dur += 24 * 60
				}
				_ = s.schedRepo.Insert(ctx, &domain.Schedule{
					RouteID:       routeID,
					DepartureMins: sc.DepMins,
					ArrivalMins:   sc.ArrMins,
					DurationMin:   dur,
					DaysOfWeek:    127,
				})
			}
			total += len(schedules)
		}
	}
	log.Printf("[ingestion] inserted %d intercity bus schedules", total)
	return nil
}

// ─── Rail ─────────────────────────────────────────────────────────────────────
// Base URL: https://apis.data.go.kr/1613000/TrainInfoService
//
// NOTE: The KORAIL/TAGO TrainInfoService endpoint and field names below need
// verification against a live API response. The schedule endpoint
// getSttnToDirctTrnList may require different parameter names.

func (s *Scheduler) fetchRailStations(ctx context.Context) error {
	params := url.Values{"numOfRows": {"1000"}, "pageNo": {"1"}}
	body, err := s.apiClient.Get(ctx,
		"/1613000/TrainInfoService/getStationList", params)
	if err != nil {
		return err
	}
	resp, err := client.Parse(body)
	if err != nil {
		return err
	}
	stations, err := parser.ParseRailStations(resp.Response.Body.Items)
	if err != nil {
		return err
	}
	for i := range stations {
		if err := s.termRepo.Upsert(ctx, &stations[i]); err != nil {
			log.Printf("[ingestion] upsert station %s: %v", stations[i].Code, err)
		}
	}
	log.Printf("[ingestion] upserted %d rail stations", len(stations))
	return nil
}

func (s *Scheduler) fetchRailSchedules(ctx context.Context, today string) error {
	allTerminals, err := s.termRepo.FindAll(ctx)
	if err != nil {
		return err
	}
	var railStations []domain.Terminal
	for _, t := range allTerminals {
		if t.Type == domain.TerminalTypeRail {
			railStations = append(railStations, t)
		}
	}

	total := 0
	for _, dep := range railStations {
		for _, arr := range railStations {
			if dep.ID == arr.ID {
				continue
			}
			params := url.Values{
				"depPlaceNm":   {dep.Name},
				"arrPlaceNm":   {arr.Name},
				"depPlandTime": {today},
				"numOfRows":    {"100"},
				"pageNo":       {"1"},
			}
			body, err := s.apiClient.Get(ctx,
				"/1613000/TrainInfoService/getSttnToDirctTrnList", params)
			if err != nil {
				continue
			}
			resp, err := client.Parse(body)
			if err != nil {
				continue
			}
			schedules, err := parser.ParseRailSchedules(resp.Response.Body.Items)
			if err != nil || len(schedules) == 0 {
				continue
			}

			for _, sc := range schedules {
				routeID, err := s.routeRepo.Insert(ctx, &domain.Route{
					Code:     sc.TrainNo,
					Type:     domain.RouteTypeRail,
					OriginID: dep.ID,
					DestID:   arr.ID,
					Operator: sc.TrainNm,
				})
				if err != nil {
					continue
				}
				dur := sc.ArrMins - sc.DepMins
				if dur < 0 {
					dur += 24 * 60
				}
				_ = s.schedRepo.Insert(ctx, &domain.Schedule{
					RouteID:       routeID,
					DepartureMins: sc.DepMins,
					ArrivalMins:   sc.ArrMins,
					DurationMin:   dur,
					DaysOfWeek:    127,
				})
			}
			total += len(schedules)
		}
	}
	log.Printf("[ingestion] inserted %d rail schedules", total)
	return nil
}

func mustLoadKST() *time.Location {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic("failed to load Asia/Seoul timezone: " + err.Error())
	}
	return loc
}
