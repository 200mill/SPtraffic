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
// Dataset: https://www.data.go.kr/data/15098522/openapi.do
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

			routeOperator := schedules[0].Operator
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
// Dataset: https://www.data.go.kr/data/15098541/openapi.do
// Base URL: https://apis.data.go.kr/1613000/SuburbsBusInfoService

func (s *Scheduler) fetchIntercityTerminals(ctx context.Context) error {
	params := url.Values{"numOfRows": {"1000"}, "pageNo": {"1"}}
	body, err := s.apiClient.Get(ctx,
		"/1613000/SuburbsBusInfoService/GetSuberbsBusTrminlList", params)
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
// GetStrtpntAlocFndSuberbsBusInfo requires both depTerminalId and arrTerminalId.
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
				"/1613000/SuburbsBusInfoService/GetStrtpntAlocFndSuberbsBusInfo", params)
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

			routeOperator := schedules[0].Operator
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
// Dataset: https://www.data.go.kr/data/15098552/openapi.do
// Base URL: https://apis.data.go.kr/1613000/TrainInfoService
//
// Station ingestion strategy:
//   1. GetCtyCodeList → list of city codes
//   2. GetCtyAcctoTrainSttnList?cityCode=XXX → stations per city
//   (No "get all stations" endpoint exists in this API)
//
// Schedule endpoint: GetStrtpntAlocFndTrainInfo
//   Requires depPlaceId + arrPlaceId (station node IDs, NOT names)

func (s *Scheduler) fetchRailStations(ctx context.Context) error {
	// Step 1: get city codes
	cityParams := url.Values{"numOfRows": {"100"}, "pageNo": {"1"}}
	cityBody, err := s.apiClient.Get(ctx,
		"/1613000/TrainInfoService/GetCtyCodeList", cityParams)
	if err != nil {
		return fmt.Errorf("GetCtyCodeList: %w", err)
	}
	cityResp, err := client.Parse(cityBody)
	if err != nil {
		return fmt.Errorf("parse city codes: %w", err)
	}
	cities, err := parser.ParseRailCityCodes(cityResp.Response.Body.Items)
	if err != nil {
		return fmt.Errorf("parse city codes: %w", err)
	}
	log.Printf("[ingestion] rail: got %d city codes", len(cities))

	// Step 2: for each city, fetch its stations
	seen := make(map[string]bool)
	total := 0
	for _, city := range cities {
		stParams := url.Values{
			"cityCode":   {city.Code},
			"numOfRows":  {"200"},
			"pageNo":     {"1"},
		}
		stBody, err := s.apiClient.Get(ctx,
			"/1613000/TrainInfoService/GetCtyAcctoTrainSttnList", stParams)
		if err != nil {
			log.Printf("[ingestion] rail stations city %s: %v", city.Code, err)
			continue
		}
		stResp, err := client.Parse(stBody)
		if err != nil {
			continue // no stations for this city — normal
		}
		stations, err := parser.ParseRailStations(stResp.Response.Body.Items)
		if err != nil {
			continue
		}
		for i := range stations {
			if seen[stations[i].Code] {
				continue
			}
			seen[stations[i].Code] = true
			if err := s.termRepo.Upsert(ctx, &stations[i]); err != nil {
				log.Printf("[ingestion] upsert station %s: %v", stations[i].Code, err)
			} else {
				total++
			}
		}
	}
	log.Printf("[ingestion] upserted %d rail stations", total)
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
				"depPlaceId":   {dep.Code}, // station node ID
				"arrPlaceId":   {arr.Code}, // station node ID
				"depPlandTime": {today},
				"numOfRows":    {"100"},
				"pageNo":       {"1"},
			}
			body, err := s.apiClient.Get(ctx,
				"/1613000/TrainInfoService/GetStrtpntAlocFndTrainInfo", params)
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
