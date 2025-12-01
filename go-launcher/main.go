package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	// Define command-line flags
	fluentBitConfig := flag.String("fluent-bit-config", "", "Path to Fluent Bit configuration file")
	rotelPath := flag.String("rotel-path", "", "Path to rotel executable")
	flag.Parse()

	// Validate required flags
	if *fluentBitConfig == "" {
		fmt.Fprintln(os.Stderr, "Error: --fluent-bit-config is required")
		flag.Usage()
		os.Exit(1)
	}

	if *rotelPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --rotel-path is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse the Fluent Bit configuration file
	config, err := ParseFluentBitConfig(*fluentBitConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing Fluent Bit config: %v\n", err)
		os.Exit(1)
	}

	// Set environment variables from parsed config
	if err := setEnvironmentVariables(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting environment variables: %v\n", err)
		os.Exit(1)
	}

	// Download S3 log processors if configured
	if err := downloadS3LogProcessors(); err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading S3 log processors: %v\n", err)
		os.Exit(1)
	}

	// Execute rotel
	if err := executeRotel(*rotelPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing rotel: %v\n", err)
		os.Exit(1)
	}
}

// setEnvironmentVariables sets the environment variables based on parsed config
func setEnvironmentVariables(config *FluentBitConfig) error {
	if config.ReceiverEndpoint != "" {
		if err := os.Setenv("ROTEL_FLUENT_RECEIVER_ENDPOINT", config.ReceiverEndpoint); err != nil {
			return fmt.Errorf("failed to set ROTEL_FLUENT_RECEIVER_ENDPOINT: %w", err)
		}
	}

	if config.ReceiverSocket != "" {
		if err := os.Setenv("ROTEL_FLUENT_RECEIVER_SOCKET", config.ReceiverSocket); err != nil {
			return fmt.Errorf("failed to set ROTEL_FLUENT_RECEIVER_SOCKET: %w", err)
		}
	}

	// Set ROTEL_OTEL_RESOURCE_ATTRIBUTES if resource attributes are available
	if config.ResourceAttributes != "" {
		existing := os.Getenv("ROTEL_OTEL_RESOURCE_ATTRIBUTES")

		value := config.ResourceAttributes
		if existing != "" {
			// Keep the existing and add to the end for precedence
			value = fmt.Sprintf("%s,%s", config.ResourceAttributes, existing)
		}

		if err := os.Setenv("ROTEL_OTEL_RESOURCE_ATTRIBUTES", value); err != nil {
			return fmt.Errorf("failed to set ROTEL_OTEL_RESOURCE_ATTRIBUTES: %w", err)
		}
	}

	// Enable fluent and otlp receivers
	if err := os.Setenv("ROTEL_RECEIVERS", "fluent,otlp"); err != nil {
		return fmt.Errorf("failed to set ROTEL_RECEIVERS: %w", err)
	}

	// Attempt to detect if no exporter config has been set, since this can
	// result in rotel immediately exiting.
	if os.Getenv("ROTEL_EXPORTER") == "" && os.Getenv("ROTEL_EXPORTERS") == "" {
		if os.Getenv("ROTEL_OTLP_EXPORTER_ENDPOINT") == "" {
			fmt.Printf("WARN: Rotel exporters have not been configured, defaulting to blackhole\n")
			if err := os.Setenv("ROTEL_EXPORTER", "blackhole"); err != nil {
				return fmt.Errorf("failed to set ROTEL_EXPORTER: %w", err)
			}
		}
	}

	return nil
}

// executeRotel executes the rotel binary with the current environment and any additional arguments
func executeRotel(rotelPath string) error {
	// Resolve the absolute path to rotel
	absPath, err := filepath.Abs(rotelPath)
	if err != nil {
		return fmt.Errorf("failed to resolve rotel path: %w", err)
	}

	// Check if rotel executable exists
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("rotel executable not found at %s: %w", absPath, err)
	}

	// Create the command
	cmd := exec.Command(absPath, "start")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ() // Pass through all environment variables

	// Execute the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rotel execution failed: %w", err)
	}

	return nil
}

// S3Object represents a parsed S3 path
type S3Object struct {
	Bucket string
	Key    string
}

// parseS3Path parses an S3 path and returns the bucket and key
func parseS3Path(s3Path string) (*S3Object, error) {
	// Path format: s3://bucket-name/path/to/file
	if !strings.HasPrefix(s3Path, "s3://") {
		return nil, fmt.Errorf("invalid S3 path (must start with s3://): %s", s3Path)
	}

	// Remove s3:// prefix
	path := strings.TrimPrefix(s3Path, "s3://")

	// Split bucket and key
	slashIdx := strings.Index(path, "/")
	if slashIdx == -1 {
		return nil, fmt.Errorf("S3 path missing key: %s", s3Path)
	}

	return &S3Object{
		Bucket: path[:slashIdx],
		Key:    path[slashIdx+1:],
	}, nil
}

// downloadS3LogProcessors downloads S3 objects specified in S3_OTLP_LOG_PROCESSORS
func downloadS3LogProcessors() error {
	s3ARNs := os.Getenv("S3_OTLP_LOG_PROCESSORS")
	if s3ARNs == "" {
		// No log processors configured, skip
		return nil
	}

	// Parse comma-separated list
	arns := strings.Split(s3ARNs, ",")
	if len(arns) == 0 {
		return nil
	}

	// Create directory for log processors
	processorDir := "/tmp/log_processors"
	if err := os.MkdirAll(processorDir, 0755); err != nil {
		return fmt.Errorf("failed to create log processor directory: %w", err)
	}

	// Initialize AWS S3 client
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	// Download each S3 object
	var downloadedPaths []string
	for i, arn := range arns {
		arn = strings.TrimSpace(arn)
		if arn == "" {
			continue
		}

		// Parse S3 path
		s3Obj, err := parseS3Path(arn)
		if err != nil {
			return fmt.Errorf("failed to parse S3 path: %w", err)
		}

		// Determine filename with prefix
		filename := filepath.Base(s3Obj.Key)
		prefixedFilename := fmt.Sprintf("%02d_%s", i+1, filename)
		destPath := filepath.Join(processorDir, prefixedFilename)

		// Download the object
		fmt.Printf("Downloading S3 object s3://%s/%s to %s\n", s3Obj.Bucket, s3Obj.Key, destPath)

		result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &s3Obj.Bucket,
			Key:    &s3Obj.Key,
		})
		if err != nil {
			return fmt.Errorf("failed to download s3://%s/%s: %w", s3Obj.Bucket, s3Obj.Key, err)
		}
		defer result.Body.Close()

		// Write to file
		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", destPath, err)
		}

		_, err = destFile.ReadFrom(result.Body)
		destFile.Close()
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}

		downloadedPaths = append(downloadedPaths, destPath)
	}

	// Set environment variable with downloaded file paths
	if len(downloadedPaths) > 0 {
		pathList := strings.Join(downloadedPaths, ",")
		if err := os.Setenv("ROTEL_OTLP_WITH_LOGS_PROCESSOR", pathList); err != nil {
			return fmt.Errorf("failed to set ROTEL_OTLP_WITH_LOGS_PROCESSOR: %w", err)
		}
		fmt.Printf("Set ROTEL_OTLP_WITH_LOGS_PROCESSOR=%s\n", pathList)
	}

	return nil
}
