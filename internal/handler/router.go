package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vsevolod-ryzhov/gofermart/internal/model"
	"github.com/vsevolod-ryzhov/gofermart/internal/service"
)

type Handler struct {
	auth *service.AuthService
}

func NewHandler(auth *service.AuthService) *Handler {
	return &Handler{
		auth: auth,
	}
}

func (h *Handler) handleRegister(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	if req.Header.Get("Content-Type") == "application/json" {
		var requestModel model.UserRegisterRequest
		dec := json.NewDecoder(req.Body)
		if err := dec.Decode(&requestModel); err != nil {
			fmt.Println(err)
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		registerError := h.auth.Register(req.Context(), requestModel.Login, requestModel.Password)
		if registerError != nil {
			switch {
			case errors.Is(registerError, service.ErrUserAlreadyExists):
				res.WriteHeader(http.StatusConflict)
			default:
				res.WriteHeader(http.StatusInternalServerError)
			}
		}
		res.WriteHeader(http.StatusOK)
	}
	res.WriteHeader(http.StatusBadRequest)
}

func (h *Handler) handleLogin(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleOrderUpload(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleGetOrders(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleGetBalance(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleBalanceWithdraw(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleWithdrawList(res http.ResponseWriter, req *http.Request) {
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleIndex(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) MakeHandler() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(5))

	r.Get("/", h.handleIndex)

	r.Post("/api/user/register", h.handleRegister)
	r.Post("/api/user/login", h.handleLogin)
	r.Post("/api/user/orders", h.handleOrderUpload)
	r.Get("/api/user/orders", h.handleGetOrders)
	r.Get("/api/user/balance", h.handleGetBalance)
	r.Post("/api/user/balance/withdraw", h.handleBalanceWithdraw)
	r.Get("/api/user/withdrawals", h.handleWithdrawList)

	return r
}
