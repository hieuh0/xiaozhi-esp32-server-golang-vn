package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	mqtt_server "xiaozhi-esp32-server-golang/internal/app/mqtt_server"
	log "xiaozhi-esp32-server-golang/logger"
)

// init configures logging.
func Init(configFile string) error {
	err := initConfig(configFile)
	if err != nil {
		return err
	}

	err = initLog()
	if err != nil {
		return err
	}

	return nil
}

func initLog() error {
	// Always write logs to a file.
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
	logrus.SetOutput(writer)
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000", // Include milliseconds.
		ForceColors:     false,                     // Disable colors for file output.
	})

	// Disable default caller reporting; use the custom caller field.
	logrus.SetReportCaller(false)
	logLevel, _ := logrus.ParseLevel(viper.GetString("log.level"))
	logrus.SetLevel(logLevel)

	return nil

}

func initConfig(configFile string) error {
	basePath, file := filepath.Split(configFile)

	// Get the file name and extension.
	fileName, fileExt := func(file string) (string, string) {
		if pos := strings.LastIndex(file, "."); pos != -1 {
			return file[:pos], strings.ToLower(file[pos+1:])
		}
		return file, ""
	}(file)

	// Set the config file name without its extension.
	viper.SetConfigName(fileName)
	viper.AddConfigPath(basePath)

	// Set the config type from the file extension.
	switch fileExt {
	case "json":
		viper.SetConfigType("json")
	case "yaml", "yml":
		viper.SetConfigType("yaml")
	default:
		return fmt.Errorf("unsupported config file type: %s", fileExt)
	}

	return viper.ReadInConfig()
}

func main() {
	// Parse command-line arguments.
	configFile := flag.String("c", "config/mqtt_config.json", "configuration file path")
	flag.Parse()

	if *configFile == "" {
		fmt.Println("configuration file path cannot be empty")
		return
	}

	// Initialize configuration and logging.
	err := Init(*configFile)
	if err != nil {
		fmt.Printf("initialization failed: %v\n", err)
		return
	}

	// Start the MQTT server.
	err = mqtt_server.StartMqttServer()
	if err != nil {
		log.Errorf("failed to start MQTT server: %v", err)
		return
	}

	fmt.Println("MQTT server started")

	// Wait for an exit signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Info("MQTT server started; press Ctrl+C to exit")
	<-quit

	log.Info("shutting down MQTT server...")
	log.Info("MQTT server stopped")
}
