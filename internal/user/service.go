package user

import (
	"context"
	"fmt"
	"time"

	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Service 用户服务接口
type Service interface {
	// CreateUser 创建用户
	CreateUser(ctx context.Context, req *types.CreateUserRequest) (*types.CreateUserResponse, error)
	// GetUser 获取用户信息
	GetUser(ctx context.Context, userID string) (*types.GetUserResponse, error)
	// GetUserByUsername 根据用户名获取用户信息
	GetUserByUsername(ctx context.Context, username string) (*types.GetUserResponse, error)
	// UpdateUser 更新用户信息
	UpdateUser(ctx context.Context, userID string, req *types.UpdateUserRequest) (*types.UpdateUserResponse, error)
	// DeleteUser 删除用户（软删除）
	DeleteUser(ctx context.Context, userID string) error
	// UserExists 检查用户是否存在
	UserExists(ctx context.Context, userID string) (bool, error)
}

// service 用户服务实现
type service struct {
	db       DBInterface
	userRepo repository.UserRepository
}

// NewService 创建用户服务
func NewService(db *repository.DB, userRepo repository.UserRepository) Service {
	return &service{
		db:       db,
		userRepo: userRepo,
	}
}

// CreateUser 创建用户
func (s *service) CreateUser(ctx context.Context, req *types.CreateUserRequest) (*types.CreateUserResponse, error) {
	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("用户名已存在: %s", req.Username)
	}

	// 创建新用户
	user := &types.User{
		ID:        uuid.New(),
		Username:  req.Username,
		CreatedAt: time.Now().UTC(),
	}

	// 使用事务执行
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// 创建用户
		if err := s.userRepo.CreateWithTx(ctx, tx, user); err != nil {
			return fmt.Errorf("创建用户失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.CreateUserResponse{
		UserID:    user.ID.String(),
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}, nil
}

// GetUser 获取用户信息
func (s *service) GetUser(ctx context.Context, userID string) (*types.GetUserResponse, error) {
	// 解析用户 ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("无效的用户 ID: %s", userID)
	}

	// 查询用户
	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	return &types.GetUserResponse{
		UserID:    user.ID.String(),
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}, nil
}

// GetUserByUsername 根据用户名获取用户信息
func (s *service) GetUserByUsername(ctx context.Context, username string) (*types.GetUserResponse, error) {
	// 查询用户
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	return &types.GetUserResponse{
		UserID:    user.ID.String(),
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}, nil
}

// UpdateUser 更新用户信息
func (s *service) UpdateUser(ctx context.Context, userID string, req *types.UpdateUserRequest) (*types.UpdateUserResponse, error) {
	// 解析用户 ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("无效的用户 ID: %s", userID)
	}

	// 检查用户是否存在
	existingUser, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 检查新用户名是否已被其他用户使用
	if req.Username != existingUser.Username {
		userWithSameName, err := s.userRepo.GetByUsername(ctx, req.Username)
		if err == nil && userWithSameName != nil && userWithSameName.ID != uid {
			return nil, fmt.Errorf("用户名已存在: %s", req.Username)
		}
	}

	// 更新用户信息
	updatedUser := &types.User{
		ID:        uid,
		Username:  req.Username,
		CreatedAt: existingUser.CreatedAt,
	}

	// 使用事务执行更新
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// 执行更新
		query := `
			UPDATE users 
			SET username = $2, updated_at = NOW()
			WHERE id = $1
		`
		_, err := tx.Exec(ctx, query, uid, req.Username)
		if err != nil {
			return fmt.Errorf("更新用户失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.UpdateUserResponse{
		UserID:    updatedUser.ID.String(),
		Username:  updatedUser.Username,
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// DeleteUser 删除用户（软删除）
func (s *service) DeleteUser(ctx context.Context, userID string) error {
	// 解析用户 ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("无效的用户 ID: %s", userID)
	}

	// 检查用户是否存在
	_, err = s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	// 使用事务执行删除
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// 软删除用户（添加 deleted_at 字段标记）
		query := `
			UPDATE users 
			SET deleted_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`
		result, err := tx.Exec(ctx, query, uid)
		if err != nil {
			return fmt.Errorf("删除用户失败: %w", err)
		}

		if result.RowsAffected() == 0 {
			return fmt.Errorf("用户不存在或已被删除: %s", userID)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// UserExists 检查用户是否存在
func (s *service) UserExists(ctx context.Context, userID string) (bool, error) {
	// 解析用户 ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("无效的用户 ID: %s", userID)
	}

	// 查询用户
	_, err = s.userRepo.GetByID(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("检查用户存在性失败: %w", err)
	}

	return true, nil
}
