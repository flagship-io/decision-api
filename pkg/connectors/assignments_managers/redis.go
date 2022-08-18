package assignments_managers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	TTL    time.Duration
}

// RedisOptions are the options necessary to make redis cache manager work
type RedisOptions struct {
	Host      string
	Username  string
	Password  string
	TLSConfig *tls.Config
	Db        int
	LogLevel  string
	LogFormat logger.LogFormat
	TTL       time.Duration
}

var rdb *redis.Client
var ctx = context.Background()

func InitRedisManager(options RedisOptions) (*RedisManager, error) {
	logger := logger.New(options.LogLevel, options.LogFormat, "redis")
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

	logger.Info("Successfully connected to redis server")

	return &RedisManager{
		client: rdb,
		logger: logger,
		TTL:    options.TTL,
	}, nil
}

// Get returns the campaigns in cache for this visitor
func (m *RedisManager) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	if m.client == nil {
		return nil, errors.New("redis cache manager not initialized")
	}

	m.logger.Infof("Getting visitor cache for ID %s", visitorID)
	cmd := m.client.HGetAll(ctx, visitorID)
	data, err := cmd.Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	cache := &common.VisitorAssignments{
		Assignments: make(map[string]*common.VisitorCache),
	}
	for k, v := range data {
		if k == "ts" {
			ts, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
			cache.Timestamp = ts
			continue
		}

		vCache := &common.VisitorCache{}
		err = json.Unmarshal([]byte(v), &vCache)
		if err != nil {
			return nil, err
		}
		cache.Assignments[k] = vCache
	}

	return cache, err
}

func (d *RedisManager) ShouldSaveAssignments(context connectors.SaveAssignmentsContext) bool {
	return true
}

// SaveAssignments saves the assignments in cache for this visitor
func (m *RedisManager) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time) error {
	if m.client == nil {
		return errors.New("redis cache manager not initialized")
	}

	m.logger.Infof("Setting visitor cache for ID %s", visitorID)
	values := map[string]interface{}{}
	for k, v := range vgIDAssignments {
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		values[k] = string(data)
	}
	values["ts"] = fmt.Sprintf("%d", date.UnixMilli())

	pipe := m.client.Pipeline()
	pipe.HSet(ctx, visitorID, values)
	pipe.Expire(ctx, visitorID, m.TTL)

	_, err := pipe.Exec(ctx)
	return err
}
