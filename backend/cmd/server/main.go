package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sptraffic/backend/internal/api"
	"github.com/sptraffic/backend/internal/api/handler"
	"github.com/sptraffic/backend/internal/cache"
	"github.com/sptraffic/backend/internal/config"
	"github.com/sptraffic/backend/internal/db"
	"github.com/sptraffic/backend/internal/ingestion"
	ingestclient "github.com/sptraffic/backend/internal/ingestion/client"
	"github.com/sptraffic/backend/internal/repository"
	"github.com/sptraffic/backend/internal/service"
)

func main() {
	cfg := config.Load()

	// ── Database ──────────────────────────────────────────────────────────────
	pool, err := pgxpool.New(context.Background(), cfg.DB.DSN())
	if err != nil {
		log.Fatalf("cannot create DB pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("database connected")

	// Run SQL migrations on startup
	if err := db.Migrate(context.Background(), pool, cfg.Server.MigrationsDir); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	// ── Redis ─────────────────────────────────────────────────────────────────
	cacheClient := cache.New(cfg.Redis.Addr)
	if err := cacheClient.Ping(context.Background()); err != nil {
		log.Printf("redis ping failed (continuing without cache): %v", err)
	} else {
		log.Println("redis connected")
	}

	// ── Repositories ──────────────────────────────────────────────────────────
	termRepo := repository.NewTerminalRepository(pool)
	routeRepo := repository.NewRouteRepository(pool)
	schedRepo := repository.NewScheduleRepository(pool)

	// ── Services ──────────────────────────────────────────────────────────────
	searchSvc := service.NewSearchService(termRepo, routeRepo, schedRepo)
	scheduleSvc := service.NewScheduleService(routeRepo, termRepo, schedRepo)

	// ── Ingestion ─────────────────────────────────────────────────────────────
	var ingestFn handler.IngestFunc
	if cfg.DataGokr.APIKey != "" {
		apiClient := ingestclient.New(cfg.DataGokr.APIKey)
		scheduler := ingestion.NewScheduler(apiClient, termRepo, routeRepo, schedRepo)
		scheduler.Start()
		defer scheduler.Stop()
		ingestFn = func() error {
			return scheduler.RunAll(context.Background())
		}
		log.Println("ingestion scheduler started (daily 02:00 KST)")
	} else {
		log.Println("DATAGOKR_API_KEY not set — ingestion scheduler disabled")
		ingestFn = func() error { return nil }
	}

	// ── HTTP handlers ─────────────────────────────────────────────────────────
	termH := handler.NewTerminalHandler(termRepo, routeRepo, schedRepo)
	searchH := handler.NewSearchHandler(searchSvc, cacheClient)
	schedH := handler.NewScheduleHandler(scheduleSvc)
	regionH := handler.NewRegionHandler(termRepo)
	healthH := handler.NewHealthHandler(pool, cacheClient)
	adminH := handler.NewAdminHandler(ingestFn)
	metroH := handler.NewMetroHandler(termRepo, cacheClient, cfg.SeoulMetro.Key)

	router := api.NewRouter(termH, searchH, schedH, regionH, healthH, adminH, metroH, api.RouterConfig{
		CORSOrigin: cfg.Server.CORSOrigin,
		AdminKey:   cfg.Admin.Key,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on :%s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}
	log.Println("server stopped")
}
