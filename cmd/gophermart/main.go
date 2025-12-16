package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vsevolod-ryzhov/gofermart/internal/config"
	"github.com/vsevolod-ryzhov/gofermart/internal/handler"
	"github.com/vsevolod-ryzhov/gofermart/internal/repository"
	"github.com/vsevolod-ryzhov/gofermart/internal/service"
)

func main() {
	configInstance := config.NewConfig()

	repo, repoErr := repository.NewPostgresRepository(configInstance.DatabaseURI)
	if repoErr != nil {
		panic(repoErr)
	}
	defer repo.Close()

	jwtService := service.NewJWTService(configInstance)

	auth := service.NewAuthService(repo, jwtService)
	orders := service.NewOrdersService(repo, configInstance.AccrualPort)
	handlerInstance := handler.NewHandler(auth, orders)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orders.StartWorker(ctx)
	defer orders.StopWorker()

	srv := &http.Server{
		Addr:         configInstance.AppPort,
		Handler:      handlerInstance.MakeHandler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Starting server on %s", configInstance.AppPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("Shutdown signal received")
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
	}

	orders.StopWorker()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}
