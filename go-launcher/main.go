package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
