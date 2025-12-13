package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	myMiddleware "github.com/vsevolod-ryzhov/gofermart/internal/middleware"
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

	if req.Header.Get("Content-Type") != "application/json" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var requestModel model.UserRegisterRequest
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&requestModel); err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	response, registerError := h.auth.Register(req.Context(), requestModel.Login, requestModel.Password)
	if registerError != nil {
		switch {
		case errors.Is(registerError, service.ErrUserAlreadyExists):
			res.WriteHeader(http.StatusConflict)
		case errors.Is(registerError, service.ErrValidationFailed):
			res.WriteHeader(http.StatusBadRequest)
		default:
			res.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	http.SetCookie(res, &http.Cookie{
		Name:     "auth_token",
		Value:    response.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(map[string]interface{}{
		"status":  "success",
		"user_id": response.UserID,
		"token":   response.Token,
	})
}

func (h *Handler) handleLogin(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	if req.Header.Get("Content-Type") != "application/json" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var requestModel model.UserLoginRequest
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&requestModel); err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	response, loginError := h.auth.Login(req.Context(), requestModel.Login, requestModel.Password)
	if loginError != nil {
		switch loginError {
		case service.ErrInvalidCredentials:
			res.WriteHeader(http.StatusUnauthorized)
		default:
			res.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	http.SetCookie(res, &http.Cookie{
		Name:     "auth_token",
		Value:    response.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(map[string]interface{}{
		"token": response.Token,
	})
}

func (h *Handler) handleOrderUpload(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleGetOrders(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleGetBalance(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleBalanceWithdraw(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleWithdrawList(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	//TODO: to be implemented
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) MakeHandler() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(5))

	r.Group(func(r chi.Router) {
		r.Post("/api/user/register", h.handleRegister)
		r.Post("/api/user/login", h.handleLogin)
	})

	r.Group(func(r chi.Router) {
		r.Use(myMiddleware.Auth(h.auth))
		r.Post("/api/user/orders", h.handleOrderUpload)
		r.Get("/api/user/orders", h.handleGetOrders)
		r.Get("/api/user/balance", h.handleGetBalance)
		r.Post("/api/user/balance/withdraw", h.handleBalanceWithdraw)
		r.Get("/api/user/withdrawals", h.handleWithdrawList)
	})

	return r
}
