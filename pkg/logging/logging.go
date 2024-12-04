package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// SetupLogging initializes the logger with the appropriate settings.
func SetupLogging() *logrus.Logger {
	// Get the debug from the environment variable
	debug := false
	if os.Getenv("DEBUG") == "true" {
		debug = true
	}

	if logger == nil {
		logger = logrus.New()
		logger.SetOutput(os.Stdout)
		logger.SetReportCaller(false) // Disable filename and line number

		// Initialize a custom log formatter
		customFormatter := new(logrus.TextFormatter)
		customFormatter.DisableTimestamp = true // Disable timestamp since k8s handles it
		customFormatter.FullTimestamp = false   // Avoid full timestamps
		customFormatter.DisableQuote = true     // Avoid quoting strings in output
		logger.SetFormatter(customFormatter)

		// Set the logging level based on the debug environment variable
		if debug {
			logger.SetLevel(logrus.DebugLevel)
		} else {
			logger.SetLevel(logrus.InfoLevel)
		}
	}

	return logger
}
