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
// It truncates all tables first (routes cascade-deletes schedules).
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

	log.Println("[ingestion] fetching express bus schedules")
	if err := s.fetchExpressBusSchedules(ctx); err != nil {
		log.Printf("[ingestion] express bus schedules error: %v", err)
	}

	log.Println("[ingestion] fetching intercity bus schedules")
	if err := s.fetchIntercitySchedules(ctx); err != nil {
		log.Printf("[ingestion] intercity bus schedules error: %v", err)
	}

	log.Println("[ingestion] fetching rail schedules")
	if err := s.fetchRailSchedules(ctx); err != nil {
		log.Printf("[ingestion] rail schedules error: %v", err)
	}

	return nil
}

// --- Express Bus ---

func (s *Scheduler) fetchExpressBusTerminals(ctx context.Context) error {
	params := url.Values{"numOfRows": {"1000"}, "pageNo": {"1"}}
	body, err := s.apiClient.Get(ctx,
		"/B551177/BusSttnInfoInqireService/getSttnList", params)
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
			log.Printf("[ingestion] upsert terminal %s: %v", terminals[i].Code, err)
		}
	}
	log.Printf("[ingestion] upserted %d express bus terminals", len(terminals))
	return nil
}

func (s *Scheduler) fetchExpressBusSchedules(ctx context.Context) error {
	// Fetch all terminal pairs — in practice you'd paginate and iterate
	// across (depTerminalId, arrTerminalId) combinations.
	// This skeleton fetches the full schedule list; adjust endpoint + params as needed.
	params := url.Values{"numOfRows": {"1000"}, "pageNo": {"1"}}
	body, err := s.apiClient.Get(ctx,
		"/B551177/BusSttnInfoInqireService/getRouteList", params)
	if err != nil {
		return err
	}
	resp, err := client.Parse(body)
	if err != nil {
		return err
	}
	schedules, err := parser.ParseExpressBusSchedules(resp.Response.Body.Items)
	if err != nil {
		return err
	}
	return s.saveSchedules(ctx, schedules, domain.RouteTypeExpress)
}

// --- Intercity Bus ---

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
			log.Printf("[ingestion] upsert terminal %s: %v", terminals[i].Code, err)
		}
	}
	log.Printf("[ingestion] upserted %d intercity bus terminals", len(terminals))
	return nil
}

func (s *Scheduler) fetchIntercitySchedules(ctx context.Context) error {
	params := url.Values{"numOfRows": {"1000"}, "pageNo": {"1"}}
	body, err := s.apiClient.Get(ctx,
		"/1613000/SuburbsBusInfoService/getRouteList", params)
	if err != nil {
		return err
	}
	resp, err := client.Parse(body)
	if err != nil {
		return err
	}
	schedules, err := parser.ParseIntercitySchedules(resp.Response.Body.Items)
	if err != nil {
		return err
	}
	return s.saveIntercitySchedules(ctx, schedules)
}

// --- Rail ---

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

func (s *Scheduler) fetchRailSchedules(ctx context.Context) error {
	params := url.Values{"numOfRows": {"1000"}, "pageNo": {"1"}}
	body, err := s.apiClient.Get(ctx,
		"/1613000/TrainInfoService/getTrainStationList", params)
	if err != nil {
		return err
	}
	resp, err := client.Parse(body)
	if err != nil {
		return err
	}
	schedules, err := parser.ParseRailSchedules(resp.Response.Body.Items)
	if err != nil {
		return err
	}
	return s.saveRailSchedules(ctx, schedules)
}

// --- Helpers ---

// saveSchedules persists express-bus (parser.ExpressBusSchedule) records.
func (s *Scheduler) saveSchedules(ctx context.Context, schedules []parser.ExpressBusSchedule, rType domain.RouteType) error {
	for _, sc := range schedules {
		dep, err := s.termRepo.FindByCode(ctx, sc.DepCode)
		if err != nil {
			continue
		}
		arr, err := s.termRepo.FindByCode(ctx, sc.ArrCode)
		if err != nil {
			continue
		}
		routeID, err := s.routeRepo.Insert(ctx, &domain.Route{
			Type:     rType,
			OriginID: dep.ID,
			DestID:   arr.ID,
			Operator: sc.Operator,
		})
		if err != nil {
			continue
		}
		dur := sc.ArrMins - sc.DepMins
		if dur < 0 {
			dur += 24 * 60 // overnight
		}
		_ = s.schedRepo.Insert(ctx, &domain.Schedule{
			RouteID:       routeID,
			DepartureMins: sc.DepMins,
			ArrivalMins:   sc.ArrMins,
			DurationMin:   dur,
			DaysOfWeek:    127,
		})
	}
	return nil
}

func (s *Scheduler) saveIntercitySchedules(ctx context.Context, schedules []parser.IntercitySchedule) error {
	for _, sc := range schedules {
		dep, err := s.termRepo.FindByCode(ctx, sc.DepCode)
		if err != nil {
			continue
		}
		arr, err := s.termRepo.FindByCode(ctx, sc.ArrCode)
		if err != nil {
			continue
		}
		routeID, err := s.routeRepo.Insert(ctx, &domain.Route{
			Type:     domain.RouteTypeIntercity,
			OriginID: dep.ID,
			DestID:   arr.ID,
			Operator: sc.Operator,
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
	return nil
}

func (s *Scheduler) saveRailSchedules(ctx context.Context, schedules []parser.RailSchedule) error {
	for _, sc := range schedules {
		dep, err := s.termRepo.FindByCode(ctx, sc.DepCode)
		if err != nil {
			continue
		}
		arr, err := s.termRepo.FindByCode(ctx, sc.ArrCode)
		if err != nil {
			continue
		}
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
	return nil
}

func mustLoadKST() *time.Location {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic("failed to load Asia/Seoul timezone: " + err.Error())
	}
	return loc
}
