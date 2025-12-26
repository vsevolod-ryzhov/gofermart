package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ory/dockertest/v3"
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
}
