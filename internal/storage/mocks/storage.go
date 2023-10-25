package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

var _ metrics.Storager = (*MockStorage)(nil)

type MockStorage struct {
	mock.Mock
}

func NewMockStorage() *MockStorage {
	return &MockStorage{}
}

func (m *MockStorage) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	err := args.Error(0)
	return err
}

func (m *MockStorage) Close() error {
	args := m.Called()
	err := args.Error(0)
	return err
}

func (m *MockStorage) SaveMany(ctx context.Context, values []metrics.Metric) error {
	args := m.Called(ctx, values)
	err := args.Error(0)
	return err
}

func (m *MockStorage) Set(ctx context.Context, metric metrics.Metric) (metrics.Metric, error) {
	args := m.Called(ctx, metric)
	value := args.Get(0).(metrics.Metric)
	err := args.Error(1)
	return value, err
}

func (m *MockStorage) Add(ctx context.Context, metric metrics.Metric) (metrics.Metric, error) {
	args := m.Called(ctx, metric)
	value := args.Get(0).(metrics.Metric)
	err := args.Error(1)
	return value, err
}

func (m *MockStorage) Get(ctx context.Context, name string) (metrics.Metric, error) {
	args := m.Called(ctx, name)
	value := args.Get(0).(metrics.Metric)
	err := args.Error(1)
	return value, err
}

func (m *MockStorage) GetAll(ctx context.Context) ([]metrics.Metric, error) {
	args := m.Called(ctx)
	values := args.Get(0).([]metrics.Metric)
	err := args.Error(1)
	return values, err
}
