package main

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/flagship-io/decision-api/pkg/utils/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestCreateLogger(t *testing.T) {
	cfg := &config.Config{
		Viper: viper.New(),
	}
	cfg.Set("log.level", "debug")
	log := createLogger(cfg)

	assert.Equal(t, logrus.DebugLevel, log.Logger.Level)
}

func TestCreateServer(t *testing.T) {
	cfg, _ := config.NewFromFilename("test")
	log := createLogger(cfg)
	_, err := createServer(cfg, log)
	assert.NotNil(t, err)

	cfg.Set("api_key", "test_api_key")
	cfg.Set("env_id", "env_id")

	_, err = createServer(cfg, log)
	assert.Nil(t, err)
}

func TestMain(t *testing.T) {
	os.Setenv("API_KEY", "api_key")
	os.Setenv("ENV_ID", "env_id")
	go func() {
		time.Sleep(2 * time.Second)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		assert.Nil(t, err)
	}()

	main()
}
