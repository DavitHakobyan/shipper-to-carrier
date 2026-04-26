package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/app"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/carrieridentity"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/externalevidence"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/identity"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/platform/config"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/platform/store/postgres"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/trust"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	if err := postgres.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	repo := identity.NewPostgresRepository(pool)
	authenticator := identity.NewService(repo, cfg.SessionTTL)
	carrierRepo := carrieridentity.NewPostgresRepository(pool)
	carrierService := carrieridentity.NewService(carrierRepo)
	evidenceRepo := externalevidence.NewPostgresRepository(pool)
	evidenceService := externalevidence.NewService(evidenceRepo, externalevidence.NewMockProvider())
	trustRepo := trust.NewPostgresRepository(pool)
	trustService := trust.NewService(trustRepo)

	handler, err := app.NewServer(cfg, authenticator, carrierService, evidenceService, trustService)
	if err != nil {
		log.Fatalf("build server: %v", err)
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", cfg.Addr)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
