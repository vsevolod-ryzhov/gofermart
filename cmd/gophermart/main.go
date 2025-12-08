package main

import (
	"net/http"
	"time"

	"github.com/vsevolod-ryzhov/gofermart/internal/config"
	"github.com/vsevolod-ryzhov/gofermart/internal/handler"
)

func main() {
	config := config.NewConfig()
	srv := &http.Server{
		Addr:         config.AppPort,
		Handler:      handler.MakeHandler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err := srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
