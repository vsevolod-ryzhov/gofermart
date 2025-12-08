package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func handleRegister(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func handleLogin(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func handleOrderUpload(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func handleGetOrders(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func handleGetBalance(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func handleBalanceWithdraw(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func handleWithdrawList(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func handleIndex(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func MakeHandler() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(5))

	r.Get("/", handleIndex)

	r.Post("/api/user/register", handleRegister)
	r.Post("/api/user/login", handleLogin)
	r.Post("/api/user/orders", handleOrderUpload)
	r.Get("/api/user/orders", handleGetOrders)
	r.Get("/api/user/balance", handleGetBalance)
	r.Post("/api/user/balance/withdraw", handleBalanceWithdraw)
	r.Get("/api/user/withdrawals", handleWithdrawList)

	return r
}
