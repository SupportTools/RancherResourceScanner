# Rancher Resource Scanner

This project is a tool to scan resources in a Kubernetes cluster for potential issues. It connects to the Kubernetes cluster, scans the resources in namespaces, and generates a report of any detected issues. It can also send an email report if configured to do so.

## Usage

To use this tool, you can run the `main.go` file. The tool will load the configuration, connect to the Kubernetes cluster, scan the resources, and generate the report. If issues are found, it will print the report to the console or send an email report based on the configuration.

## Dependencies

This project uses the following external libraries:
- `github.com/supporttools/RancherResourceScanner` for internal packages
- `github.com/sirupsen/logrus` for logging
- `gomail.v2` for sending email reports
- `k8s.io` packages for interacting with Kubernetes

## Structure

- `main.go`: Entry point of the tool that starts the scanning process.
- `pkg/k8s/k8s.go`: Package for interacting with the Kubernetes cluster, including connecting, fetching resources, and scanning.
- `pkg/logging/logging.go`: Package for setting up logging with Logrus.
- `pkg/config/config.go`: Package for loading configurations from environment variables.

## Configuration

The tool reads configuration from environment variables to customize its behavior. Here are the available configuration options:
- `DEBUG`: Enable debug mode.
- `LOG_LEVEL`: Set the log level.
- `KUBECONFIG`: Path to the kubeconfig file.
- `CRON_SCHEDULE`: Cron schedule for periodic scanning.
- `EMAIL_REPORT`: Enable email reporting.
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TO`: SMTP settings for email reporting.

## Collaborate

Feel free to contribute to this project by forking it, making changes, and submitting pull requests. If you encounter any issues or have suggestions, please open an issue in this repository.