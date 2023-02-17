package main

import (
	"crypto/tls"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/flagship-io/decision-api/pkg/config"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/logger"
)

func getAssignmentsManager(cfg *config.Config) (assignmentsManager connectors.AssignmentsManager, err error) {
	switch cfg.GetDefaultString("cache.type", "") {
	case "memory":
		assignmentsManager = assignments_managers.InitMemoryManager()
	case "local":
		assignmentsManager, err = assignments_managers.InitLocalCacheManager(assignments_managers.LocalOptions{
			DbPath: cfg.GetDefaultString("cache.options.dbpath", "cache_data"),
		})
	case "redis":
		var tlsConfig *tls.Config
		if cfg.GetBool("cache.options.redisTls") {
			tlsConfig = &tls.Config{}
		}
		assignmentsManager, err = assignments_managers.InitRedisManager(assignments_managers.RedisOptions{
			Host:      cfg.GetDefaultString("cache.options.redisHost", "localhost:6379"),
			Username:  cfg.GetDefaultString("cache.options.redisUsername", ""),
			Password:  cfg.GetDefaultString("cache.options.redisPassword", ""),
			Db:        cfg.GetDefaultInt("cache.options.redisDb", 0),
			TTL:       cfg.GetDefaultDuration("cache.options.redisTtl", 3*30*24*time.Hour),
			LogLevel:  cfg.GetDefaultString("log.level", config.LoggerLevel),
			LogFormat: logger.LogFormat(cfg.GetDefaultString("log.format", config.LoggerFormat)),
			TLSConfig: tlsConfig,
		})
	case "dynamo":
		session, _ := session.NewSession()
		client := dynamodb.New(session)
		assignmentsManager = assignments_managers.InitDynamoManager(assignments_managers.DynamoManagerOptions{
			Client:              client,
			TableName:           cfg.GetDefaultString("cache.options.dynamoTableName", "visitor-assignments"),
			PrimaryKeySeparator: cfg.GetDefaultString("cache.options.dynamoPKSeparator", "."),
			PrimaryKeyField:     cfg.GetDefaultString("cache.options.dynamoPKField", "id"),
			GetItemTimeout:      cfg.GetDefaultDuration("cache.options.dynamoGetTimeout", 1*time.Second),
			LogLevel:            cfg.GetDefaultString("log.level", config.LoggerLevel),
			LogFormat:           logger.LogFormat(cfg.GetDefaultString("log.format", config.LoggerFormat)),
		})
	default:
		assignmentsManager = &assignments_managers.EmptyManager{}
	}

	return assignmentsManager, err
}
