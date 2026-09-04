package main

import (
	"crypto/tls"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/decision-api/pkg/utils/config"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
)

func getAssignmentsManager(cfg *config.Config) (assignmentsManager connectors.AssignmentsManager, err error) {
	switch cfg.GetStringDefault("cache.type", "") {
	case "memory":
		assignmentsManager = assignments_managers.InitMemoryManagerWithOptions(
			cfg.GetIntDefault("cache.options.memoryMaxEntries", assignments_managers.DefaultMemoryMaxEntries),
			cfg.GetDurationDefault("cache.options.memoryTtl", assignments_managers.DefaultMemoryTTL))
	case "local":
		assignmentsManager, err = assignments_managers.InitLocalCacheManager(assignments_managers.LocalOptions{
			DbPath: cfg.GetStringDefault("cache.options.dbpath", "cache_data"),
		})
	case "redis":
		var tlsConfig *tls.Config
		if cfg.GetBool("cache.options.redisTls") {
			tlsConfig = &tls.Config{}
		}
		assignmentsManager, err = assignments_managers.InitRedisManager(assignments_managers.RedisOptions{
			Host:      cfg.GetStringDefault("cache.options.redisHost", "localhost:6379"),
			Username:  cfg.GetStringDefault("cache.options.redisUsername", ""),
			Password:  cfg.GetStringDefault("cache.options.redisPassword", ""),
			Db:        cfg.GetIntDefault("cache.options.redisDb", 0),
			TTL:       cfg.GetDurationDefault("cache.options.redisTtl", 3*30*24*time.Hour),
			LogLevel:  cfg.GetStringDefault("log.level", config.LoggerLevel),
			LogFormat: logger.LogFormat(cfg.GetStringDefault("log.format", config.LoggerFormat)),
			TLSConfig: tlsConfig,
		})
	case "dynamo":
		session, _ := session.NewSession()
		client := dynamodb.New(session)
		assignmentsManager = assignments_managers.InitDynamoManager(assignments_managers.DynamoManagerOptions{
			Client:              client,
			TableName:           cfg.GetStringDefault("cache.options.dynamoTableName", "visitor-assignments"),
			PrimaryKeySeparator: cfg.GetStringDefault("cache.options.dynamoPKSeparator", "."),
			PrimaryKeyField:     cfg.GetStringDefault("cache.options.dynamoPKField", "id"),
			GetItemTimeout:      cfg.GetDurationDefault("cache.options.dynamoGetTimeout", 1*time.Second),
			LogLevel:            cfg.GetStringDefault("log.level", config.LoggerLevel),
			LogFormat:           logger.LogFormat(cfg.GetStringDefault("log.format", config.LoggerFormat)),
		})
	default:
		assignmentsManager = &assignments_managers.EmptyManager{}
	}

	return assignmentsManager, err
}

func getHitsProcessor(cfg *config.Config, logLvl string, logFmt logger.LogFormat) connectors.HitsProcessor {
	var hitsProcessor connectors.HitsProcessor = hits_processors.NewDataCollectProcessor(
		hits_processors.WithLogger(logLvl, logFmt),
		// Unset keys read as zero, which every option below replaces with its default.
		hits_processors.WithBatchOptions(
			cfg.GetInt("hits.batch_size"),
			cfg.GetDuration("hits.batching_window")),
		hits_processors.WithSendOptions(cfg.GetInt("hits.max_batch_bytes")),
	)

	// On unless turned off. Every distinct context still reaches the collector at least once per
	// TTL, so what a visitor was segmented on is unchanged - only the number of times it is repeated.
	if cfg.GetBoolDefault("hits.deduplicate_context", true) {
		hitsProcessor = hits_processors.NewContextDeduplicator(
			hitsProcessor,
			cfg.GetIntDefault("hits.context_max_entries", hits_processors.DefaultContextMaxEntries),
			cfg.GetDurationDefault("hits.context_ttl", hits_processors.DefaultContextTTL),
			logLvl, logFmt)
	}

	return hitsProcessor
}
