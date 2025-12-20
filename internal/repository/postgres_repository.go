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
	appErrors "github.com/vsevolod-ryzhov/gofermart/internal/errors"
	"github.com/vsevolod-ryzhov/gofermart/internal/model"
)

type User struct {
	ID       int
	Login    string
	Password string
}

type PostgresRepository struct {
	db *sql.DB
}

const newOrderStatus = "NEW"
const processingOrderStatus = "PROCESSING"

func getPendingStatuses() [2]string {
	var ret = [...]string{newOrderStatus, processingOrderStatus}
	return ret
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

func (r *PostgresRepository) GetUserOrders(ctx context.Context, userID int) (*model.UserDisplayOrders, error) {
	query := `SELECT number, user_id, status, updated_at, COALESCE(accrual, -1) as accrual FROM orders WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders model.UserDisplayOrders
	for rows.Next() {
		var record model.UserDisplayOrder
		var balance int
		if err := rows.Scan(&record.Number, &record.UserID, &record.Status, &record.UpdatedAt, &balance); err != nil {
			return nil, err
		}
		if balance != -1 {
			value := convertStoredMoneyToFloat(balance)
			record.Accrual = &value
		}
		orders = append(orders, record)
	}

	return &orders, nil
}

func (r *PostgresRepository) GetOrder(ctx context.Context, orderID int) (*model.UserOrder, error) {
	var record model.UserOrder

	query := `SELECT number, user_id, status, updated_at, accrual FROM orders WHERE number = $1`

	err := r.db.QueryRowContext(ctx, query, orderID).Scan(&record.Number, &record.UserID, &record.Status, &record.UpdatedAt, &record.Accrual)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &record, nil
}

func (r *PostgresRepository) AddOrder(ctx context.Context, orderID, userID int) error {
	query := `INSERT INTO orders (number, status, user_id) VALUES ($1, $2, $3)`

	_, err := r.db.ExecContext(ctx, query, orderID, newOrderStatus, userID)
	if err != nil {
		return err
	}

	return nil
}

func (r *PostgresRepository) GetUserBalanceInfo(ctx context.Context, userID int) *model.User {
	var user model.User
	var balance int
	user.Withdrawn = 0
	query := `SELECT balance FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&balance)
	if err != nil {
		return nil
	}
	user.Balance = convertStoredMoneyToFloat(balance)

	var withdrawn int
	query = `SELECT SUM(sum) FROM withdrawals WHERE user_id = $1`
	_ = r.db.QueryRowContext(ctx, query, userID).Scan(&withdrawn)
	user.Withdrawn = convertStoredMoneyToFloat(withdrawn)

	return &user
}

func (r *PostgresRepository) GetPendingOrders(ctx context.Context) (*model.UserOrders, error) {
	query := `SELECT number, user_id, status, updated_at, accrual FROM orders WHERE status IN ($1, $2)`
	rows, err := r.db.QueryContext(ctx, query, newOrderStatus, processingOrderStatus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders model.UserOrders
	for rows.Next() {
		var record model.UserOrder

		if err := rows.Scan(&record.Number, &record.UserID, &record.Status, &record.UpdatedAt, &record.Accrual); err != nil {
			return nil, err
		}

		orders = append(orders, record)
	}

	return &orders, nil
}

func (r *PostgresRepository) UpdateOrderStatus(ctx context.Context, userID, orderID int, status string, accrual float64) error {
	query := `UPDATE orders SET status = $1, accrual = $2 WHERE number = $3 and user_id = $4`

	intAccrual := convertFloatToStoredMoney(accrual)

	txn, txnErr := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if txnErr != nil {
		return txnErr
	}
	defer txn.Rollback()

	_, err := r.db.ExecContext(ctx, query, status, intAccrual, orderID, userID)
	if err != nil {
		return err
	}

	query = `UPDATE users SET balance = balance + $1 WHERE id = $2`
	_, err = r.db.ExecContext(ctx, query, intAccrual, userID)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (r *PostgresRepository) CreateWithdrawal(ctx context.Context, userID, orderNumber int, sumFloat float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	sum := convertFloatToStoredMoney(sumFloat)

	var currentBalance int
	queryCheck := `SELECT balance FROM users WHERE id = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, queryCheck, userID).Scan(&currentBalance)
	if err != nil {
		return err
	}

	if currentBalance < sum {
		return appErrors.ErrNotEnoughMoney
	}

	queryInsert := `INSERT INTO withdrawals (user_id, order_number, sum) VALUES ($1, $2, $3)`
	_, err = tx.ExecContext(ctx, queryInsert, userID, orderNumber, sum)
	if err != nil {
		return err
	}

	queryUpdate := `UPDATE users SET balance = balance - $1 WHERE id = $2`
	_, err = tx.ExecContext(ctx, queryUpdate, sum, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (r *PostgresRepository) GetUserWithdrawals(ctx context.Context, userID int) (*model.Withdrawals, error) {
	query := `SELECT user_id, order_number, sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdraws model.Withdrawals
	for rows.Next() {
		var record model.Withdrawal
		var sumInt int
		if err := rows.Scan(&record.UserID, &record.OrderNumber, &sumInt, &record.ProcessedAt); err != nil {
			return nil, err
		}
		record.Sum = convertStoredMoneyToFloat(sumInt)
		withdraws = append(withdraws, record)
	}

	return &withdraws, nil
}

func convertStoredMoneyToFloat(moneyFromDatabase int) float64 {
	return float64(moneyFromDatabase) / 100
}

func convertFloatToStoredMoney(moneyFromDatabase float64) int {
	return int(moneyFromDatabase * 100)
}
