package user

import (
	"context"
	"testing"
	"time"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 简化的 Mock 实现用于基本测试

type SimpleUserRepo struct {
	mock.Mock
}

func (m *SimpleUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*types.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *SimpleUserRepo) GetByUsername(ctx context.Context, username string) (*types.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *SimpleUserRepo) Create(ctx context.Context, user *types.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *SimpleUserRepo) CreateWithTx(ctx context.Context, tx pgx.Tx, user *types.User) error {
	args := m.Called(ctx, tx, user)
	return args.Error(0)
}

type SimpleDB struct {
	mock.Mock
}

func (m *SimpleDB) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	// 简化实现：直接执行函数
	return fn(nil)
}

func TestUserService_Simple(t *testing.T) {
	t.Run("GetUser - 成功获取用户", func(t *testing.T) {
		// 创建 mock
		mockRepo := new(SimpleUserRepo)
		mockDB := new(SimpleDB)

		// 设置期望
		userID := uuid.New()
		expectedUser := &types.User{
			ID:        userID,
			Username:  "testuser",
			CreatedAt: time.Now(),
		}
		mockRepo.On("GetByID", mock.Anything, userID).Return(expectedUser, nil)

		// 创建服务
		service := &service{
			db:       mockDB,
			userRepo: mockRepo,
		}

		// 执行测试
		result, err := service.GetUser(context.Background(), userID.String())

		// 验证结果
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, userID.String(), result.UserID)
		assert.Equal(t, "testuser", result.Username)

		// 验证 mock 调用
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetUser - 用户不存在", func(t *testing.T) {
		// 创建 mock
		mockRepo := new(SimpleUserRepo)
		mockDB := new(SimpleDB)

		// 设置期望
		userID := uuid.New()
		mockRepo.On("GetByID", mock.Anything, userID).Return(nil, assert.AnError)

		// 创建服务
		service := &service{
			db:       mockDB,
			userRepo: mockRepo,
		}

		// 执行测试
		result, err := service.GetUser(context.Background(), userID.String())

		// 验证结果
		assert.Error(t, err)
		assert.Nil(t, result)

		// 验证 mock 调用
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetUserByUsername - 成功", func(t *testing.T) {
		// 创建 mock
		mockRepo := new(SimpleUserRepo)
		mockDB := new(SimpleDB)

		// 设置期望
		expectedUser := &types.User{
			ID:        uuid.New(),
			Username:  "testuser",
			CreatedAt: time.Now(),
		}
		mockRepo.On("GetByUsername", mock.Anything, "testuser").Return(expectedUser, nil)

		// 创建服务
		service := &service{
			db:       mockDB,
			userRepo: mockRepo,
		}

		// 执行测试
		result, err := service.GetUserByUsername(context.Background(), "testuser")

		// 验证结果
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "testuser", result.Username)

		// 验证 mock 调用
		mockRepo.AssertExpectations(t)
	})

	t.Run("UserExists - 用户存在", func(t *testing.T) {
		// 创建 mock
		mockRepo := new(SimpleUserRepo)
		mockDB := new(SimpleDB)

		// 设置期望
		userID := uuid.New()
		expectedUser := &types.User{
			ID:       userID,
			Username: "testuser",
		}
		mockRepo.On("GetByID", mock.Anything, userID).Return(expectedUser, nil)

		// 创建服务
		service := &service{
			db:       mockDB,
			userRepo: mockRepo,
		}

		// 执行测试
		exists, err := service.UserExists(context.Background(), userID.String())

		// 验证结果
		assert.NoError(t, err)
		assert.True(t, exists)

		// 验证 mock 调用
		mockRepo.AssertExpectations(t)
	})

	t.Run("UserExists - 用户不存在", func(t *testing.T) {
		// 创建 mock
		mockRepo := new(SimpleUserRepo)
		mockDB := new(SimpleDB)

		// 设置期望
		userID := uuid.New()
		mockRepo.On("GetByID", mock.Anything, userID).Return(nil, pgx.ErrNoRows)

		// 创建服务
		service := &service{
			db:       mockDB,
			userRepo: mockRepo,
		}

		// 执行测试
		exists, err := service.UserExists(context.Background(), userID.String())

		// 验证结果
		assert.NoError(t, err)
		assert.False(t, exists)

		// 验证 mock 调用
		mockRepo.AssertExpectations(t)
	})
}
