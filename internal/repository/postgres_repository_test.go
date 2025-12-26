package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ory/dockertest/v3"
	appErrors "github.com/vsevolod-ryzhov/gofermart/internal/errors"
)

var db *sql.DB

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	resource, err := pool.Run("postgres", "latest", []string{
		"POSTGRES_PASSWORD=secret",
		"POSTGRES_USER=user",
		"POSTGRES_DB=testdb",
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostPort := resource.GetPort("5432/tcp")
	connectionString := fmt.Sprintf("host=localhost port=%s user=user password=secret dbname=testdb sslmode=disable", hostPort)

	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", connectionString)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	projectRoot, err := getProjectRoot()
	if err != nil {
		log.Fatalf("Failed to find project root: %s", err)
	}

	migrationsPath := filepath.Join(projectRoot, "migrations")
	log.Printf("Applying migrations from: %s", migrationsPath)

	migrationDB, err := sql.Open("pgx", connectionString)
	if err != nil {
		log.Fatalf("Failed to apply migrations: %s", err)
	}
	defer migrationDB.Close()

	if err := applyMigrations(migrationDB, migrationsPath); err != nil {
		log.Fatalf("Failed to apply migrations: %s", err)
	}

	code := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func getProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentDir, nil
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}
		currentDir = parent
	}

	return "", fmt.Errorf("go.mod not found in any parent directory")
}

func cleanupTestData(t *testing.T) {
	_, err := db.Exec("DELETE FROM withdrawals")
	if err != nil {
		t.Logf("Warning: failed to clean withdrawals: %v", err)
	}

	_, err = db.Exec("DELETE FROM orders")
	if err != nil {
		t.Logf("Warning: failed to clean orders: %v", err)
	}

	_, err = db.Exec("DELETE FROM users")
	if err != nil {
		t.Logf("Warning: failed to clean users: %v", err)
	}

	_, err = db.Exec("ALTER SEQUENCE users_id_seq RESTART WITH 1")
	if err != nil {
		t.Logf("Warning: failed to reset sequence: %v", err)
	}
}

func TestUserOperations(t *testing.T) {
	if db == nil {
		t.Fatal("Database not initialized")
	}

	repo := &PostgresRepository{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cleanupTestData(t)

	t.Run("CreateUser", func(t *testing.T) {
		userID, err := repo.CreateUser(ctx, "testuser1", "password123")
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		if userID <= 0 {
			t.Errorf("Expected positive user ID, got %d", userID)
		}

		exists, err := repo.UserExists(ctx, "testuser1")
		if err != nil {
			t.Fatalf("Failed to check user existence: %v", err)
		}

		if !exists {
			t.Error("User should exist after creation")
		}
	})

	t.Run("CreateDuplicateUser", func(t *testing.T) {
		_, err := repo.CreateUser(ctx, "duplicateuser", "pass1")
		if err != nil {
			t.Fatalf("Failed to create first user: %v", err)
		}

		_, err = repo.CreateUser(ctx, "duplicateuser", "pass2")
		if err == nil {
			t.Error("Expected error when creating duplicate user")
		}
	})

	t.Run("GetNonExistentUser", func(t *testing.T) {
		user, err := repo.GetUserByLogin(ctx, "nonexistent_user_12345")
		if err != sql.ErrNoRows {
			t.Errorf("Expected sql.ErrNoRows, got %v", err)
		}

		if user != nil {
			t.Error("Expected nil user for non-existent login")
		}
	})
}

func TestOrderOperations(t *testing.T) {
	if db == nil {
		t.Fatal("Database not initialized")
	}

	repo := &PostgresRepository{db: db}
	ctx := context.Background()

	cleanupTestData(t)

	userID, err := repo.CreateUser(ctx, "order_user", "password")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("NoPendingOrders", func(t *testing.T) {
		orders, err := repo.GetPendingOrders(ctx)
		if err != nil {
			t.Fatalf("Failed to get pending orders: %v", err)
		}

		if orders == nil {
			t.Fatal("Expected non-nil orders slice")
		}

		if len(*orders) != 0 {
			t.Errorf("Expected 0 orders when no orders exist, got %d", len(*orders))
		}
	})

	t.Run("AddOrder", func(t *testing.T) {
		orderNumber := 123456789

		err := repo.AddOrder(ctx, orderNumber, userID)
		if err != nil {
			t.Fatalf("Failed to add order: %v", err)
		}

		order, err := repo.GetOrder(ctx, orderNumber)
		if err != nil {
			t.Fatalf("Failed to get order: %v", err)
		}

		if order.Number != orderNumber {
			t.Errorf("Expected order number %d, got %d", orderNumber, order.Number)
		}

		if order.UserID != userID {
			t.Errorf("Expected user ID %d, got %d", userID, order.UserID)
		}

		if order.Status != newOrderStatus {
			t.Errorf("Expected status %s, got %s", newOrderStatus, order.Status)
		}
	})

	t.Run("GetUserOrders", func(t *testing.T) {
		orderNumbers := []int{111111, 222222, 333333}
		for _, orderNum := range orderNumbers {
			err := repo.AddOrder(ctx, orderNum, userID)
			if err != nil {
				t.Fatalf("Failed to add order %d: %v", orderNum, err)
			}
		}

		orders, err := repo.GetUserOrders(ctx, userID)
		if err != nil {
			t.Fatalf("Failed to get user orders: %v", err)
		}

		if orders == nil {
			t.Fatal("Expected orders slice, got nil")
		}

		if len(*orders) != 4 {
			t.Errorf("Expected 4 orders, got %d", len(*orders))
		}

		if len(*orders) > 0 {
			lastOrder := (*orders)[0]
			if lastOrder.Number != 333333 {
				t.Logf("Note: orders might not be sorted by created_at DESC")
			}
		}
	})

	t.Run("PendingOrders", func(t *testing.T) {
		orders, err := repo.GetPendingOrders(ctx)
		if err != nil {
			t.Fatalf("Failed to get pending orders: %v", err)
		}

		if orders == nil {
			t.Fatal("Expected non-nil orders slice")
		}

		if len(*orders) != 4 {
			t.Errorf("Expected 4 pending orders, got %d", len(*orders))
		}
	})

	t.Run("AddDuplicateOrder", func(t *testing.T) {
		duplicateOrderNumber := 999999

		err := repo.AddOrder(ctx, duplicateOrderNumber, userID)
		if err != nil {
			t.Fatalf("Failed to add order first time: %v", err)
		}

		err = repo.AddOrder(ctx, duplicateOrderNumber, userID)
		if err == nil {
			t.Error("Expected error when adding duplicate order")
		}
	})
}

func TestBalanceOperations(t *testing.T) {
	if db == nil {
		t.Fatal("Database not initialized")
	}

	repo := &PostgresRepository{db: db}
	ctx := context.Background()

	cleanupTestData(t)

	userID, err := repo.CreateUser(ctx, "balance_user", "password")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("GetEmptyBalance", func(t *testing.T) {
		user := repo.GetUserBalanceInfo(ctx, userID)
		if user == nil {
			t.Fatal("Failed to get user balance info")
		}

		if user.Balance != 0 {
			t.Errorf("Expected balance 0 for new user, got %.2f", user.Balance)
		}

		if user.Withdrawn != 0 {
			t.Errorf("Expected withdrawn 0 for new user, got %.2f", user.Withdrawn)
		}
	})

	t.Run("UpdateOrderStatusWithAccrual", func(t *testing.T) {
		orderNumber := 555555
		err := repo.AddOrder(ctx, orderNumber, userID)
		if err != nil {
			t.Fatalf("Failed to add order: %v", err)
		}

		accrual := 150.75
		err = repo.UpdateOrderStatus(ctx, userID, orderNumber, "PROCESSED", accrual)
		if err != nil {
			t.Fatalf("Failed to update order status: %v", err)
		}

		user := repo.GetUserBalanceInfo(ctx, userID)
		if user == nil {
			t.Fatal("Failed to get user balance info after accrual")
		}

		expectedBalance := 150.75
		if user.Balance != expectedBalance {
			t.Errorf("Expected balance %.2f after accrual, got %.2f", expectedBalance, user.Balance)
		}

		order, err := repo.GetOrder(ctx, orderNumber)
		if err != nil {
			t.Fatalf("Failed to get updated order: %v", err)
		}

		if order.Status != "PROCESSED" {
			t.Errorf("Expected order status PROCESSED, got %s", order.Status)
		}

		if convertStoredMoneyToFloat(*order.Accrual) != accrual {
			t.Errorf("Expected order accrual %.2f, got %.2f", accrual, convertStoredMoneyToFloat(*order.Accrual))
		}
	})

	t.Run("CreateWithdrawal", func(t *testing.T) {
		orderNumber := 666666
		err := repo.AddOrder(ctx, orderNumber, userID)
		if err != nil {
			t.Fatalf("Failed to add order: %v", err)
		}

		accrual := 100.25
		err = repo.UpdateOrderStatus(ctx, userID, orderNumber, "PROCESSED", accrual)
		if err != nil {
			t.Fatalf("Failed to update order status: %v", err)
		}

		withdrawalOrderNumber := 777777
		withdrawalAmount := 50.50
		err = repo.CreateWithdrawal(ctx, userID, withdrawalOrderNumber, withdrawalAmount)
		if err != nil {
			t.Fatalf("Failed to create withdrawal: %v", err)
		}

		user := repo.GetUserBalanceInfo(ctx, userID)
		if user == nil {
			t.Fatal("Failed to get user balance info after withdrawal")
		}

		expectedBalance := 200.50
		if math.Abs(user.Balance-expectedBalance) > 0.01 {
			t.Errorf("Expected balance %.2f after withdrawal, got %.2f", expectedBalance, user.Balance)
		}

		expectedWithdrawn := 50.50
		if math.Abs(user.Withdrawn-expectedWithdrawn) > 0.01 {
			t.Errorf("Expected withdrawn %.2f, got %.2f", expectedWithdrawn, user.Withdrawn)
		}

		withdrawals, err := repo.GetUserWithdrawals(ctx, userID)
		if err != nil {
			t.Fatalf("Failed to get user withdrawals: %v", err)
		}

		if withdrawals == nil {
			t.Fatal("Expected withdrawals slice, got nil")
		}

		if len(*withdrawals) != 1 {
			t.Errorf("Expected 1 withdrawal, got %d", len(*withdrawals))
		}

		if len(*withdrawals) > 0 {
			withdrawal := (*withdrawals)[0]
			if withdrawal.Sum != withdrawalAmount {
				t.Errorf("Expected withdrawal amount %.2f, got %.2f", withdrawalAmount, withdrawal.Sum)
			}

			if withdrawal.OrderNumber != strconv.Itoa(withdrawalOrderNumber) {
				t.Errorf("Expected withdrawal order number %d, got %s", withdrawalOrderNumber, withdrawal.OrderNumber)
			}
		}
	})

	t.Run("InsufficientBalance", func(t *testing.T) {
		err := repo.CreateWithdrawal(ctx, userID, 888888, 50000.00)
		if err == nil {
			t.Error("Expected error when withdrawing more than balance")
		}

		if !errors.Is(err, appErrors.ErrNotEnoughMoney) {
			t.Errorf("Expected ErrNotEnoughMoney, got %v", err)
		}
	})
}
