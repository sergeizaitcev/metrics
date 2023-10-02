package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

type MockFileStorage struct {
	mock.Mock
}

func NewMockFileStorage() *MockFileStorage {
	return &MockFileStorage{}
}

func (m *MockFileStorage) Append(metric metrics.Metric) error {
	args := m.Called(metric)
	return args.Error(0)
}
