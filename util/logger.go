package util

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger
var once sync.Once

// initLogger initializes the logger with a default level of DebugLevel.
func initLogger() {
	once.Do(func() {
		logger = logrus.New()
		logger.SetOutput(os.Stdout)
		logger.SetLevel(logrus.DebugLevel) // Default level is Debug
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339, // Format with date and time
		})
	})
}

// LogError logs an error message
func LogError(message string, args ...interface{}) {
	initLogger() // Ensure logger is initialized
	logger.Errorf(fmt.Sprintf(message, args...))
}

// LogWarn logs a warning message
func LogWarn(message string, args ...interface{}) {
	initLogger() // Ensure logger is initialized
	logger.Warnf(fmt.Sprintf(message, args...))
}

// LogInfo logs an info message
func LogInfo(message string, args ...interface{}) {
	initLogger() // Ensure logger is initialized
	logger.Infof(fmt.Sprintf(message, args...))
}

// LogDebug logs a debug message
func LogDebug(message string, args ...interface{}) {
	initLogger() // Ensure logger is initialized
	logger.Debugf(fmt.Sprintf(message, args...))
}

// setLogLevel maps a string log level to the custom LogLevel and sets it using SetLoggerLevel.
func SetLogLevel(level string) error {
	initLogger() // Ensure logger is initialized

	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logger.SetLevel(logrus.FatalLevel)
	default:
		logger.Warn("Unknown log level provided, defaulting to Debug")
		logger.SetLevel(logrus.DebugLevel)
	}
	return nil
}

func GetLogLevel() logrus.Level {
	initLogger() // Ensure logger is initialized
	return logger.GetLevel()
}