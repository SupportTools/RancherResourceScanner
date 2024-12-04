package main

import (
	"bytes"
	"fmt"

	"github.com/supporttools/RancherResourceScanner/pkg/config"
	"github.com/supporttools/RancherResourceScanner/pkg/k8s"
	"github.com/supporttools/RancherResourceScanner/pkg/logging"
	"gopkg.in/gomail.v2"
)

var logger = logging.SetupLogging()

func main() {
	// Load configuration
	config.LoadConfiguration()

	// Log application start
	logger.Infof("Starting Rancher Resource Scanner for Cluster: %s", config.CFG.ClusterName)

	// Connect to the Kubernetes cluster
	clientset, dynamicClient, err := k8s.ConnectToCluster(config.CFG.Kubeconfig)
	if err != nil {
		logger.Fatalf("Error connecting to Kubernetes cluster: %v", err)
	}

	// Verify access to the cluster
	if err := k8s.VerifyAccessToCluster(clientset); err != nil {
		logger.Fatalf("Cluster access verification failed: %v", err)
	}

	// Scan resources for issues
	logger.Info("Scanning resources...")
	results, err := k8s.ScanNamespaceResources(clientset, dynamicClient)
	if err != nil {
		logger.Fatalf("Error scanning resources: %v", err)
	}

	if len(results) > 0 {
		logger.Info("Issues found, generating report...")
		report := generateReport(results)
		if config.CFG.EmailReport {
			sendEmailReport(report)
		} else {
			logger.Info(report)
		}
	} else {
		logger.Info("No issues found in resources.")
	}
}

func generateReport(results []k8s.ResourceCheckResult) string {
	var buffer bytes.Buffer
	buffer.WriteString("Daily Kubernetes Resource Report\n\n")
	buffer.WriteString("Detected issues in resources:\n\n")

	for _, result := range results {
		buffer.WriteString(fmt.Sprintf("Namespace: %s\n", result.Namespace))
		buffer.WriteString(fmt.Sprintf("Resource: %s\n", result.Resource))
		buffer.WriteString(fmt.Sprintf("Name: %s\n", result.Name))
		buffer.WriteString(fmt.Sprintf("Issue: %s\n", result.Issue))
		buffer.WriteString(fmt.Sprintf("Additional Info: %s\n", result.AdditionalInfo))
		buffer.WriteString("\n")
	}

	return buffer.String()
}

func sendEmailReport(report string) {
	mailer := gomail.NewMessage()
	mailer.SetHeader("From", config.CFG.SmtpFrom)
	mailer.SetHeader("To", config.CFG.SmtpTo)
	mailer.SetHeader("Subject", "Daily Kubernetes Resource Report")
	mailer.SetBody("text/plain", report)

	dialer := gomail.NewDialer(config.CFG.SmtpHost, config.CFG.SmtpPort, config.CFG.SmtpUser, config.CFG.SmtpPassword)

	if err := dialer.DialAndSend(mailer); err != nil {
		logger.Errorf("Failed to send email report: %v", err)
	} else {
		logger.Info("Email report sent successfully.")
	}
}
