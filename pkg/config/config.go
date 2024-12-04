package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// AppConfig structure for environment-based configurations.
type AppConfig struct {
	Debug        bool   `json:"debug"`
	LogLevel     string `json:"log_level"`
	MetricsPort  int    `json:"metricsPort"`
	Kubeconfig   string `json:"kubeconfig"`
	ClusterName  string `json:"cluster_name"`
	CronSchedule string `json:"cron_schedule"`
	RunOnce      bool   `json:"run_once"`
	EmailReport  bool   `json:"email_report"`
	SmtpHost     string `json:"smtp_host"`
	SmtpPort     int    `json:"smtp_port"`
	SmtpUser     string `json:"smtp_user"`
	SmtpPassword string `json:"smtp_password"`
	SmtpFrom     string `json:"smtp_from"`
	SmtpTo       string `json:"smtp_to"`
}

// CFG is the global configuration object.
var CFG AppConfig

// LoadConfiguration loads configuration from environment variables.
func LoadConfiguration() {
	CFG.Debug = parseEnvBool("DEBUG", false)
	CFG.LogLevel = getEnvOrDefault("LOG_LEVEL", "info")
	CFG.Kubeconfig = getEnvOrDefault("KUBECONFIG", "~/.kube/config")
	CFG.MetricsPort = parseEnvInt("METRICS_PORT", 9999)
	CFG.ClusterName = getEnvOrDefault("CLUSTER_NAME", "k8s-cluster")
	CFG.CronSchedule = getEnvOrDefault("CRON_SCHEDULE", "0 0 * * *")
	CFG.RunOnce = parseEnvBool("RUN_ONCE", false)
	CFG.EmailReport = parseEnvBool("EMAIL_REPORT", false)
	CFG.SmtpHost = getEnvOrDefault("SMTP_HOST", "")
	CFG.SmtpPort = parseEnvInt("SMTP_PORT", 25)
	CFG.SmtpUser = getEnvOrDefault("SMTP_USER", "")
	CFG.SmtpPassword = getEnvOrDefault("SMTP_PASSWORD", "")
	CFG.SmtpFrom = getEnvOrDefault("SMTP_FROM", "")
	CFG.SmtpTo = getEnvOrDefault("SMTP_TO", "")
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func parseEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var intValue int
	_, err := fmt.Sscanf(value, "%d", &intValue)
	if err != nil {
		log.Printf("Failed to parse environment variable %s: %v. Using default value: %d", key, err, defaultValue)
		return defaultValue
	}
	return intValue
}

func parseEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	value = strings.ToLower(value)

	// Handle additional truthy and falsy values
	switch value {
	case "1", "t", "true", "yes", "on", "enabled":
		return true
	case "0", "f", "false", "no", "off", "disabled":
		return false
	default:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Error parsing %s as bool: %v. Using default value: %t", key, err, defaultValue)
			return defaultValue
		}
		return boolValue
	}
}
