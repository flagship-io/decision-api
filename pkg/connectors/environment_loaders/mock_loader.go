package environment_loaders

import (
	"github.com/flagship-io/decision-api/pkg/models"
)

type MockLoader struct {
	MockedEnvironment *models.Environment
}

func (loader *MockLoader) Init(envID string, APIKey string) error {
	return nil
}

func (l *MockLoader) LoadEnvironment(envID string, APIKey string) (*models.Environment, error) {
	return l.MockedEnvironment, nil
}
