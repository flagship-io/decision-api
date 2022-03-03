package environment_loaders

import (
	common "github.com/flagship-io/flagship-common"
)

type MockLoader struct {
	MockedEnvironment *common.Environment
}

func (loader *MockLoader) Init(envID string, APIKey string) error {
	return nil
}

func (l *MockLoader) LoadEnvironment(envID string, APIKey string) (*common.Environment, error) {
	return l.MockedEnvironment, nil
}
