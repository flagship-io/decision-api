package environment_loaders

import (
	"github.com/flagship-io/decision-api/pkg/models"
)

type MockLoader struct {
	MockedEnvironment *models.Environment
	ErrorReturned     error
}

func (loader *MockLoader) Init(envID string, APIKey string) error {
	return nil
}

func (l *MockLoader) LoadEnvironment(envID string, APIKey string) (*models.Environment, error) {
	if l.ErrorReturned != nil {
		return nil, l.ErrorReturned
	}
	return l.MockedEnvironment, nil
}
