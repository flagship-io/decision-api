package assignments_managers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	common "github.com/flagship-io/flagship-common"
	"github.com/go-redis/redis/v8"
)

// RedisManager represents a redis db manager object
type RedisManager struct {
	client *redis.Client
	logger *logger.Logger
}

// RedisOptions are the options necessary to make redis cache manager work
type RedisOptions struct {
	Host      string
	Username  string
	Password  string
	TLSConfig *tls.Config
	Db        int
	LogLevel  string
}

var rdb *redis.Client
var ctx = context.Background()

func InitRedisManager(options RedisOptions) (*RedisManager, error) {
	logger := logger.New(options.LogLevel, "redis")
	logger.Info("Connecting to server...")
	rdb = redis.NewClient(&redis.Options{
		Addr:      options.Host,
		Username:  options.Username,
		TLSConfig: options.TLSConfig,
		Password:  options.Password,
		DB:        options.Db,
	})
	_, err := rdb.Ping(ctx).Result()

	if err != nil {
		logger.Errorf("Error when connecting to redis server: %v", err)
		return nil, err
	}

	logger.Info("Succesfully connected to redis server")

	return &RedisManager{
		client: rdb,
		logger: logger,
	}, nil
}

// SaveAssignments saves the assignments in cache for this visitor
func (m *RedisManager) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time, context connectors.SaveAssignmentsContext) error {
	if m.client == nil {
		return errors.New("redis cache manager not initialized")
	}

	data, err := json.Marshal(&common.VisitorAssignments{
		Assignments: vgIDAssignments,
		Timestamp:   date.UnixMilli(),
	})
	if err != nil {
		return err
	}

	m.logger.Infof("Setting visitor cache for ID %s", visitorID)
	cmd := m.client.Set(ctx, visitorID, string(data), 0)
	_, err = cmd.Result()

	return err
}

// Get returns the campaigns in cache for this visitor
func (m *RedisManager) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	if m.client == nil {
		return nil, errors.New("redis cache manager not initialized")
	}

	m.logger.Infof("Getting visitor cache for ID %s", visitorID)
	cmd := m.client.Get(ctx, visitorID)
	data, err := cmd.Bytes()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	cache := &common.VisitorAssignments{}
	err = json.Unmarshal(data, &cache)

	return cache, err
}
