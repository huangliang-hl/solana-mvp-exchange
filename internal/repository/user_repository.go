package repository

import (
	"context"
	"fmt"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*types.User, error)
	GetByUsername(ctx context.Context, username string) (*types.User, error)
	Create(ctx context.Context, user *types.User) error
	CreateWithTx(ctx context.Context, tx pgx.Tx, user *types.User) error
}

// userRepository 用户仓储实现
type userRepository struct {
	db *DB
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *DB) UserRepository {
	return &userRepository{db: db}
}

// GetByID 根据 ID 获取用户（排除已删除的用户）
func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.User, error) {
	query := `
		SELECT id, username, created_at 
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL
	`

	var user types.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("用户不存在: %s", id)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return &user, nil
}

// GetByUsername 根据用户名获取用户（排除已删除的用户）
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*types.User, error) {
	query := `
		SELECT id, username, created_at 
		FROM users 
		WHERE username = $1 AND deleted_at IS NULL
	`

	var user types.User
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("用户不存在: %s", username)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return &user, nil
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *types.User) error {
	query := `
		INSERT INTO users (id, username, created_at)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		user.ID,
		user.Username,
		user.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}

	return nil
}

// CreateWithTx 在事务中创建用户
func (r *userRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, user *types.User) error {
	query := `
		INSERT INTO users (id, username, created_at)
		VALUES ($1, $2, $3)
	`

	_, err := tx.Exec(ctx, query,
		user.ID,
		user.Username,
		user.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}

	return nil
}
