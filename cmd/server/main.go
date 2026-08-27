package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"r1rpc/internal/app"
	"r1rpc/internal/config"
	"r1rpc/internal/store"
	"r1rpc/internal/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := cfg.ApplyTimeZone(); err != nil {
		log.Fatalf("apply time zone: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := store.BootstrapSchema(ctx, cfg); err != nil {
		log.Fatalf("bootstrap schema: %v", err)
	}

	st, err := store.New(cfg)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}

	application := app.New(cfg, st)
	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			log.Printf("close application: %v", closeErr)
		}
	}()
	if err := application.EnsureDeviceGroup(context.Background()); err != nil {
		log.Fatalf("ensure device group: %v", err)
	}
	if err := application.Store.EnsureBootstrapAdmin(context.Background(), cfg.BootstrapAdminUser, cfg.BootstrapAdminPass); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}
	rebuildCtx, rebuildCancel := context.WithTimeout(context.Background(), 20*time.Second)
	if err := application.Store.RebuildRecentMetricsFromRequests(rebuildCtx, cfg.RawRetentionDays); err != nil {
		log.Printf("rebuild recent device metrics failed: %v", err)
	}
	rebuildCancel()
	runCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	application.StartBackgroundJobs(runCtx)
	webServer := web.New(application)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           webServer.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("server listening on %s", cfg.HTTPAddr)
	log.Printf("time zone: %s", cfg.TimeZone)
	log.Printf("bootstrap admin: %s / %s", cfg.BootstrapAdminUser, cfg.BootstrapAdminPass)
	log.Printf("invoke auth: 按分组配置（none / apikey）")
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.ListenAndServe()
	}()
	select {
	case <-runCtx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		shutdownErr := server.Shutdown(shutdownCtx)
		shutdownCancel()
		if shutdownErr != nil {
			log.Printf("graceful shutdown: %v", shutdownErr)
			_ = server.Close()
		}
		wsShutdownCtx, wsShutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		if wsErr := webServer.ShutdownClientConnections(wsShutdownCtx); wsErr != nil {
			log.Printf("websocket drain: %v", wsErr)
		}
		wsShutdownCancel()
		if err := <-serveDone; err != nil && err != http.ErrServerClosed {
			log.Printf("listen and serve: %v", err)
		}
		if closeErr := application.Close(); closeErr != nil {
			log.Printf("close application: %v", closeErr)
		}
	case err := <-serveDone:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("listen and serve: %v", err)
		}
		wsShutdownCtx, wsShutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		if wsErr := webServer.ShutdownClientConnections(wsShutdownCtx); wsErr != nil {
			log.Printf("websocket drain: %v", wsErr)
		}
		wsShutdownCancel()
		if closeErr := application.Close(); closeErr != nil {
			log.Printf("close application: %v", closeErr)
		}
	}
}
