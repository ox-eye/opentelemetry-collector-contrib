// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package groupbytraceprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbytraceprocessor"

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/bsm/redislock"
	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"strconv"
	"time"

	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

const (
	// typeStr is the value of "type" for this processor in the configuration.
	typeStr config.Type = "groupbytrace"
	// The stability level of the processor.
	stability = component.StabilityLevelBeta

	defaultWaitDuration         = time.Second
	defaultNumTraces            = 1_000_000
	defaultDeDuplicationTimeout = time.Minute
	defaultNumWorkers           = 1
	defaultDiscardOrphans       = false
	defaultStoreOnDisk          = false
	defaultStoreCacheOnRedis    = false
	defaultRedisHost            = "127.0.0.1"
	defaultRedisPort            = 6379
	defaultRedisAuth            = ""
	defaultRedisTLS             = false
	defaultRedisCluster         = false
)

var (
	errDiskStorageNotSupported    = fmt.Errorf("option 'disk storage' not supported in this release")
	errDiscardOrphansNotSupported = fmt.Errorf("option 'discard orphans' not supported in this release")
)

// NewFactory returns a new factory for the Filter processor.
func NewFactory() component.ProcessorFactory {
	// TODO: find a more appropriate way to get this done, as we are swallowing the error here
	_ = view.Register(MetricViews()...)

	return component.NewProcessorFactory(
		typeStr,
		createDefaultConfig,
		component.WithTracesProcessorAndStabilityLevel(createTracesProcessor, stability))
}

// createDefaultConfig creates the default configuration for the processor.
func createDefaultConfig() config.Processor {
	return &Config{
		ProcessorSettings:    config.NewProcessorSettings(config.NewComponentID(typeStr)),
		NumTraces:            defaultNumTraces,
		DeduplicationTimeout: defaultDeDuplicationTimeout,
		NumWorkers:           defaultNumWorkers,
		WaitDuration:         defaultWaitDuration,

		StoreCacheOnRedis: defaultStoreCacheOnRedis,
		RedisHost:         defaultRedisHost,
		RedisPort:         defaultRedisPort,
		RedisAuth:         defaultRedisAuth,
		RedisTLS:          defaultRedisTLS,
		RedisCluster:      defaultRedisCluster,

		// not supported for now
		DiscardOrphans: defaultDiscardOrphans,
		StoreOnDisk:    defaultStoreOnDisk,
	}
}

func configureRedisClientOptions(cfg config.Processor) *redis.Options {
	oCfg := cfg.(*Config)
	redisOptions := redis.Options{
		Addr:     oCfg.RedisHost + ":" + strconv.Itoa(oCfg.RedisPort),
		Password: oCfg.RedisAuth,
	}

	if oCfg.RedisTLS {
		redisOptions.TLSConfig = &tls.Config{}
	}

	return &redisOptions
}

func configureRedisClusterOptions(cfg config.Processor) *redis.ClusterOptions {
	oCfg := cfg.(*Config)
	redisOptions := redis.ClusterOptions{
		Addrs:          []string{oCfg.RedisHost + ":" + strconv.Itoa(oCfg.RedisPort)},
		Password:       oCfg.RedisAuth,
		ReadOnly:       false,
		RouteRandomly:  false,
		RouteByLatency: false,
	}

	if oCfg.RedisTLS {
		redisOptions.TLSConfig = &tls.Config{}
	}

	return &redisOptions
}

func checkRedisConnection(redisClient *redis.Client, logger *zap.Logger) bool {
	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		logger.Error("Could not connect to redis", zap.Error(err))
		return false
	} else {
		logger.Info("Connected to redis")
		return true
	}
}

func checkRedisClusterConnection(redisClusterClient *redis.ClusterClient, logger *zap.Logger) bool {
	_, err := redisClusterClient.Ping(context.Background()).Result()
	if err != nil {
		logger.Error("Could not connect to redis cluster", zap.Error(err))
		return false
	} else {
		logger.Info("Connected to redis cluster")
		return true
	}
}

func connectRedisClient(cfg config.Processor, logger *zap.Logger) *redis.Client {
	oCfg := cfg.(*Config)
	if !oCfg.StoreCacheOnRedis {
		return nil
	}
	redisClientOptions := configureRedisClientOptions(cfg)
	redisClient := redis.NewClient(redisClientOptions)

	if !checkRedisConnection(redisClient, logger) {
		return nil
	}
	return redisClient
}

func connectRedisClusterClient(cfg config.Processor, logger *zap.Logger) *redis.ClusterClient {
	oCfg := cfg.(*Config)
	if !oCfg.StoreCacheOnRedis {
		return nil
	}
	redisClusterOptions := configureRedisClusterOptions(cfg)
	redisClusterClient := redis.NewClusterClient(redisClusterOptions)

	if !checkRedisClusterConnection(redisClusterClient, logger) {
		return nil
	}
	return redisClusterClient
}

func configureRedisCacheAndLock(cfg config.Processor, logger *zap.Logger) (*cache.Cache, *redislock.Client) {
	var redisClient *redis.Client
	var redisClusterClient *redis.ClusterClient
	oCfg := cfg.(*Config)

	localCache := cache.New(&cache.Options{
		LocalCache: cache.NewTinyLFU(bufferSize, oCfg.DeduplicationTimeout),
	})

	if !oCfg.StoreCacheOnRedis {
		logger.Info("Creating local cache")
		return localCache, nil
	}

	logger.Info("Creating redis cache")
	if oCfg.RedisCluster {
		redisClusterClient = connectRedisClusterClient(cfg, logger)
		if redisClusterClient == nil {
			return localCache, nil
		}
		redisCache := cache.New(&cache.Options{
			Redis: redisClusterClient,
		})
		redisLock := redislock.New(redisClusterClient)
		return redisCache, redisLock
	} else {
		redisClient = connectRedisClient(cfg, logger)
		if redisClient == nil {
			return localCache, nil
		}
		redisCache := cache.New(&cache.Options{
			Redis: redisClient,
		})
		redisLock := redislock.New(redisClient)
		return redisCache, redisLock
	}
}

// createTracesProcessor creates a trace processor based on this config.
func createTracesProcessor(
	_ context.Context,
	params component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Traces) (component.TracesProcessor, error) {
	oCfg := cfg.(*Config)

	var st storage
	var redisCache *cache.Cache
	var redisLock *redislock.Client

	if oCfg.StoreOnDisk {
		return nil, errDiskStorageNotSupported
	}
	if oCfg.DiscardOrphans {
		return nil, errDiscardOrphansNotSupported
	}

	if oCfg.StoreCacheOnRedis {
		redisCache, redisLock = configureRedisCacheAndLock(cfg, params.Logger)
	}
	st = newMemoryStorage()

	return newGroupByTraceProcessor(params.Logger, st, redisCache, redisLock, nextConsumer, *oCfg), nil
}
