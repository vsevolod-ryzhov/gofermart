package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	authMiddleware "github.com/vsevolod-ryzhov/gofermart/internal/middleware"
	"github.com/vsevolod-ryzhov/gofermart/internal/model"
	"github.com/vsevolod-ryzhov/gofermart/internal/service"
)

type AuthService interface {
	Register(ctx context.Context, login, password string) (*service.RegisterResponse, error)
	Login(ctx context.Context, login, password string) (*service.LoginResponse, error)
	ValidateToken(tokenString string) (int, error)
}

type OrdersService interface {
	GetUserOrders(ctx context.Context, userID int) (*model.UserDisplayOrders, error)
	AddOrder(ctx context.Context, userID, orderID int) error
	GetUserBalanceInfo(ctx context.Context, userID int) (*model.User, error)
	ApplyWithdrawal(ctx context.Context, userID, orderID int, sum float64) error
	GetUserWithdrawals(ctx context.Context, userID int) (*model.Withdrawals, error)
	ValidateOrderNumber(orderNumber int) bool
}

type Handler struct {
	auth   AuthService
	orders OrdersService
}

func NewHandler(auth AuthService, orders OrdersService) *Handler {
	return &Handler{
		auth:   auth,
		orders: orders,
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

	userID, ok := authMiddleware.GetUserID(req)
	if !ok {
		authMiddleware.RespondWithJSONError(res, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var body []byte
	var err error

	body, err = io.ReadAll(req.Body)
	if err != nil {
		authMiddleware.RespondWithJSONError(res, http.StatusInternalServerError, err.Error())
		return
	}

	orderNumberStr := string(body)
	orderNumberStr = strings.TrimSpace(orderNumberStr)

	orderNumber, err := strconv.ParseInt(orderNumberStr, 10, 64)
	if err != nil {
		authMiddleware.RespondWithJSONError(res, http.StatusBadRequest, "Invalid order number format")
		return
	}

	err = h.orders.AddOrder(req.Context(), userID, int(orderNumber))
	if err != nil {
		switch err {
		case service.ErrNumberInvalid:
			authMiddleware.RespondWithJSONError(res, http.StatusUnprocessableEntity, err.Error())
		case service.ErrOrderAddedByAnotherUser:
			authMiddleware.RespondWithJSONError(res, http.StatusConflict, err.Error())
		case service.ErrOrderAlreadyExists:
			res.WriteHeader(http.StatusOK)
		default:
			authMiddleware.RespondWithJSONError(res, http.StatusInternalServerError, err.Error())
		}
		return
	}

	res.WriteHeader(http.StatusAccepted)
}

func (h *Handler) handleGetOrders(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	userID, ok := authMiddleware.GetUserID(req)
	if !ok {
		authMiddleware.RespondWithJSONError(res, http.StatusUnauthorized, "Unauthorized")
		return
	}

	orders, err := h.orders.GetUserOrders(req.Context(), userID)
	if err != nil {
		authMiddleware.RespondWithJSONError(res, http.StatusInternalServerError, err.Error())
		return
	}

	if len(*orders) == 0 {
		authMiddleware.RespondWithJSONError(res, http.StatusNoContent, "No content")
		return
	}

	if err := json.NewEncoder(res).Encode(orders); err != nil {
		authMiddleware.RespondWithJSONError(res, http.StatusInternalServerError, err.Error())
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleGetBalance(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	userID, ok := authMiddleware.GetUserID(req)
	if !ok {
		authMiddleware.RespondWithJSONError(res, http.StatusUnauthorized, "Unauthorized")
		return
	}

	balance, err := h.orders.GetUserBalanceInfo(req.Context(), userID)
	if err != nil {
		authMiddleware.RespondWithJSONError(res, http.StatusInternalServerError, err.Error())
		return
	}

	if err := json.NewEncoder(res).Encode(balance); err != nil {
		authMiddleware.RespondWithJSONError(res, http.StatusInternalServerError, err.Error())
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleBalanceWithdraw(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	userID, ok := authMiddleware.GetUserID(req)
	if !ok {
		authMiddleware.RespondWithJSONError(res, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var requestModel model.WithdrawRequest
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&requestModel); err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	orderNumber, err := strconv.ParseInt(requestModel.Order, 10, 64)
	if err != nil {
		authMiddleware.RespondWithJSONError(res, http.StatusBadRequest, "Invalid order number format")
		return
	}

	if !h.orders.ValidateOrderNumber(int(orderNumber)) {
		authMiddleware.RespondWithJSONError(res, http.StatusUnprocessableEntity, "Invalid order number format")
		return
	}

	balance, err := h.orders.GetUserBalanceInfo(req.Context(), userID)
	if err != nil {
		authMiddleware.RespondWithJSONError(res, http.StatusInternalServerError, err.Error())
		return
	}

	if balance.Balance < requestModel.Sum {
		authMiddleware.RespondWithJSONError(res, http.StatusPaymentRequired, "Insufficient balance")
		return
	}

	err = h.orders.ApplyWithdrawal(req.Context(), userID, int(orderNumber), requestModel.Sum)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNumberInvalid):
			authMiddleware.RespondWithJSONError(res, http.StatusPaymentRequired, "Insufficient balance")
		default:
			authMiddleware.RespondWithJSONError(res, http.StatusInternalServerError, err.Error())
		}
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (h *Handler) handleWithdrawList(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	userID, ok := authMiddleware.GetUserID(req)
	if !ok {
		authMiddleware.RespondWithJSONError(res, http.StatusUnauthorized, "Unauthorized")
		return
	}

	withdrawals, err := h.orders.GetUserWithdrawals(req.Context(), userID)
	if err != nil {
		authMiddleware.RespondWithJSONError(res, http.StatusInternalServerError, err.Error())
		return
	}

	if len(*withdrawals) == 0 {
		authMiddleware.RespondWithJSONError(res, http.StatusNoContent, "No content")
		return
	}

	if err := json.NewEncoder(res).Encode(withdrawals); err != nil {
		authMiddleware.RespondWithJSONError(res, http.StatusInternalServerError, err.Error())
		return
	}

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
		r.Use(authMiddleware.Auth(h.auth))
		r.Post("/api/user/orders", h.handleOrderUpload)
		r.Get("/api/user/orders", h.handleGetOrders)
		r.Get("/api/user/balance", h.handleGetBalance)
		r.Post("/api/user/balance/withdraw", h.handleBalanceWithdraw)
		r.Get("/api/user/withdrawals", h.handleWithdrawList)
	})

	return r
}
