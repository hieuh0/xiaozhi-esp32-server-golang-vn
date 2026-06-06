package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
	"xiaozhi-esp32-server-golang/internal/app/server/auth"
	redisdb "xiaozhi-esp32-server-golang/internal/db/redis"
	user_config "xiaozhi-esp32-server-golang/internal/domain/config"

	log "xiaozhi-esp32-server-golang/logger"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/mitchellh/hashstructure/v2"
	logrus "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

// Controls periodic configuration updates.
var (
	configUpdateTicker *time.Ticker
	configUpdateStop   chan struct{}
	configUpdateWg     sync.WaitGroup
)

func Init(configFile string) error {
	//init config
	err := initConfig(configFile)
	if err != nil {
		fmt.Printf("initConfig err: %+v", err)
		os.Exit(1)
		return err
	}

	//init log
	initLog()

	// Initialize the configuration system, including the WebSocket connection.
	// Apply pushed settings in main only after reading and comparing the current configuration.
	ctx := context.Background()
	if err := user_config.InitConfigSystem(ctx); err != nil {
		fmt.Printf("failed to initialize configuration system: %v\n", err)
	}

	// Fetch and update configuration from the API.
	if err := updateConfigFromAPI(); err != nil {
		fmt.Printf("failed to fetch configuration from API; using local configuration: %v\n", err)
	}

	// Start periodic configuration updates.
	startPeriodicConfigUpdate()

	//init vad
	initVad()

	//init redis
	initRedis()

	// The memory module is initialized lazily on first use.

	//init auth
	err = initAuthManager()
	if err != nil {
		fmt.Printf("initAuthManager err: %+v", err)
		os.Exit(1)
		return err
	}

	return nil
}

// startPeriodicConfigUpdate starts periodic configuration updates.
func startPeriodicConfigUpdate() {
	// Read the update interval, defaulting to five minutes.
	updateInterval := viper.GetDuration("config_provider.update_interval")
	if updateInterval <= 0 {
		updateInterval = 30 * time.Second
	}

	// Check whether periodic updates are enabled.
	if !viper.GetBool("config_provider.enable_periodic_update") {
		log.Info("periodic configuration updates are disabled")
		return
	}

	configUpdateStop = make(chan struct{})
	configUpdateTicker = time.NewTicker(updateInterval)

	configUpdateWg.Add(1)
	go func() {
		defer configUpdateWg.Done()
		defer configUpdateTicker.Stop()

		for {
			select {
			case <-configUpdateTicker.C:
				if err := updateConfigFromAPI(); err != nil {
					log.Warnf("periodic configuration update failed: %v", err)
				} else {
					// log.Debug("periodic configuration update succeeded")
				}
			case <-configUpdateStop:
				log.Info("periodic configuration updates stopped")
				return
			}
		}
	}()

	log.Infof("periodic configuration updates started, interval: %v", updateInterval)
}

// StopPeriodicConfigUpdate stops periodic configuration updates.
func StopPeriodicConfigUpdate() {
	if configUpdateStop != nil {
		close(configUpdateStop)
		configUpdateWg.Wait()
		logrus.Info("periodic configuration updates stopped")
	}
}

func initConfig(configFile string) error {
	viper.SetConfigFile(configFile)

	// Read the configuration file.
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return nil
}

// ApplySystemConfigToViper merges a pushed system_config update into viper.
func ApplySystemConfigToViper(data map[string]interface{}) {
	if err := viper.MergeConfigMap(data); err != nil {
		log.Warnf("failed to merge pushed configuration into viper: %v", err)
		return
	}
	log.Info("merged system configuration pushed over WebSocket into viper")
}

// SystemConfigEqual compares system configurations semantically using order-independent fingerprints.
func SystemConfigEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		log.Debugf("[SystemConfigEqual] result: true (both nil)")
		return true
	}
	if a == nil || b == nil {
		log.Debugf("[SystemConfigEqual] result: false (one side is nil)")
		return false
	}
	ha, err1 := hashstructure.Hash(a, hashstructure.FormatV2, nil)
	hb, err2 := hashstructure.Hash(b, hashstructure.FormatV2, nil)
	if err1 != nil || err2 != nil {
		log.Debugf("[SystemConfigEqual] result: false (hash failed err1=%v err2=%v)", err1, err2)
		return false
	}
	equal := ha == hb
	log.Debugf("[SystemConfigEqual] result: %t (ha=%d hb=%d), a: %+v, b: %+v", equal, ha, hb, a, b)
	return equal
}

// updateConfigFromAPI fetches API configuration and updates viper.
// It retries until successful.
func updateConfigFromAPI() error {
	configProviderType := viper.GetString("config_provider.type")
	retryInterval := 10 * time.Second // Retry interval.
	retryCount := 0

	for {
		// Get the manager backend address from configuration.
		configProvider, err := user_config.GetProvider(configProviderType)
		if err != nil {
			retryCount++
			log.Warnf("failed to get configuration provider (attempt %d): %v; retrying in %v", retryCount, err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		// Create the request context.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		// Fetch the system configuration JSON.
		configJSON, err := configProvider.GetSystemConfig(ctx)
		cancel()

		if err != nil {
			retryCount++
			log.Warnf("failed to fetch system configuration (attempt %d): %v; retrying in %v", retryCount, err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		if configJSON == "" {
			// Treat an empty response as success.
			if retryCount > 0 {
				log.Infof("configuration fetched successfully after %d retries (empty configuration)", retryCount)
			}
			return nil
		}

		// Parse JSON into a map.
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
			retryCount++
			log.Warnf("failed to parse configuration JSON (attempt %d): %v; retrying in %v", retryCount, err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		//log.Debugf("Load config from API: %+v", configMap)

		// Merge the configuration into viper.
		if err := viper.MergeConfigMap(configMap); err != nil {
			retryCount++
			log.Warnf("failed to merge configuration into viper (attempt %d): %v; retrying in %v", retryCount, err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		// Success.
		if retryCount > 0 {
			log.Infof("configuration fetched successfully after %d retries", retryCount)
		} else {
			log.Debug("configuration fetched successfully")
		}
		return nil
	}
}

func initLog() error {
	// Write logs to a file.
	binPath, _ := os.Executable()
	baseDir := filepath.Dir(binPath)
	logPath := fmt.Sprintf("%s/%s%s", baseDir, viper.GetString("log.path"), viper.GetString("log.file"))
	/* Log rotation options:
	`WithLinkName` creates a symlink to the latest log.
	`WithRotationTime` sets the rotation interval.
	Only one of WithMaxAge and WithRotationCount may be set.
	`WithMaxAge` sets how long files are retained.
	`WithRotationCount` sets the maximum number of retained files.
	*/
	// Rotate every minute and retain the most recent three minutes of logs.
	writer, err := rotatelogs.New(
		logPath+".%Y%m%d",
		rotatelogs.WithLinkName(logPath),
		rotatelogs.WithRotationCount(uint(viper.GetInt("log.max_age"))),
		rotatelogs.WithRotationTime(time.Duration(86400)*time.Second),
	)
	if err != nil {
		fmt.Printf("init log error: %v\n", err)
		os.Exit(1)
		return err
	}

	// Select log outputs from configuration.
	if viper.GetBool("log.stdout") {
		// Write to both file and standard output.
		multiWriter := io.MultiWriter(writer, os.Stdout)
		logrus.SetOutput(multiWriter)
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000", // Include milliseconds.
			ForceColors:     true,                      // Enable colors on standard output.
		})
	} else {
		// Write only to file.
		logrus.SetOutput(writer)
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000", // Include milliseconds.
			ForceColors:     false,                     // Disable colors for file output.
		})
	}

	// Disable default caller reporting; use the custom caller field.
	logrus.SetReportCaller(false)
	logLevel, _ := logrus.ParseLevel(viper.GetString("log.level"))
	logrus.SetLevel(logLevel)

	return nil
}

func initVad() error {
	log.Infof("initializing VAD module...")
	vadProvider := viper.GetString("vad.provider")
	log.Infof("VAD provider: %s", vadProvider)

	// VAD is initialized lazily through the global resource pool.
	log.Infof("VAD module will initialize lazily on first use")
	return nil
}

func initRedis() error {
	// Initialize the shared Redis module.
	redisConfig := &redisdb.Config{
		Host:     viper.GetString("redis.host"),
		Port:     viper.GetInt("redis.port"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	}

	err := redisdb.Init(redisConfig)
	if err != nil {
		fmt.Printf("init redis error: %v\n", err)
		return err
	}

	return nil
}

func initAuthManager() error {
	return auth.Init()
}
