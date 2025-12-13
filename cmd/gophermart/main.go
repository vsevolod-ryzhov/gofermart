package main

import (
	"net/http"
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
	orders := service.NewOrdersService(repo)
	handlerInstance := handler.NewHandler(auth, orders)

	srv := &http.Server{
		Addr:         configInstance.AppPort,
		Handler:      handlerInstance.MakeHandler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err := srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
