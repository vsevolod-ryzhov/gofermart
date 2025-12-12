package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type User struct {
	ID       int
	Login    string
	Password string
}

type PostgresRepository struct {
	db *sql.DB
}

func applyMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}

func NewPostgresRepository(connectionString string) (*PostgresRepository, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	migrationDB, err := sql.Open("pgx", connectionString)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to open migration database: %w", err)
	}
	defer migrationDB.Close()

	if err := applyMigrations(migrationDB); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) Close() error {
	err := r.db.Close()
	if err != nil {
		return err
	}

	return nil
}

func (r *PostgresRepository) UserExists(ctx context.Context, login string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE login = $1)`

	err := r.db.QueryRowContext(ctx, query, login).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *PostgresRepository) CreateUser(ctx context.Context, login, password string) (int, error) {
	var userID int
	query := `INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id`

	err := r.db.QueryRowContext(ctx, query, login, password).Scan(&userID)

	return userID, err
}

func (r *PostgresRepository) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	var user User
	query := `SELECT id, login, password 
              FROM users WHERE login = $1`

	err := r.db.QueryRowContext(ctx, query, login).Scan(
		&user.ID,
		&user.Login,
		&user.Password,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &user, nil
}
