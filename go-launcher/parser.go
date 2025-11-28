package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// FluentBitConfig holds the parsed configuration values
type FluentBitConfig struct {
	ReceiverEndpoint   string // Listen:Port from forward input
	ReceiverSocket     string // unix_path from forward input
	ResourceAttributes string // key=value pairs from record_modifier filter
}

// ParseFluentBitConfig parses a Fluent Bit configuration file
func ParseFluentBitConfig(filename string) (*FluentBitConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	config := &FluentBitConfig{}
	scanner := bufio.NewScanner(file)

	var currentSection string
	var sectionType string // INPUT, FILTER, OUTPUT
	var sectionName string // forward, record_modifier, etc.
	var listen string
	var port string
	var resourceAttrs []string

	onEndSection := func() {
		if sectionType == "INPUT" && sectionName == "forward" {
			if listen != "" && port != "" {
				config.ReceiverEndpoint = fmt.Sprintf("%s:%s", listen, port)
			}
		}
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			onEndSection()

			// Reset for new section
			currentSection = strings.TrimSpace(line[1 : len(line)-1])
			sectionType = strings.ToUpper(currentSection)
			sectionName = ""
			listen = ""
			port = ""
			continue
		}

		// Parse key-value pairs within sections
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		// Section names are the same
		if key == "name" {
			sectionName = value
			continue
		}

		// Handle based on section type
		switch sectionType {
		case "INPUT":
			switch key {
			case "listen":
				if sectionName == "forward" {
					listen = value
				}
			case "port":
				if sectionName == "forward" {
					port = value
				}
			case "unix_path":
				if sectionName == "forward" {
					config.ReceiverSocket = value
				}
			}

		case "FILTER":
			switch key {
			case "record":
				if sectionName == "record_modifier" {
					// Split the record value into key=value format
					recordParts := strings.SplitN(value, " ", 2)
					if len(recordParts) == 2 {
						recordKey := strings.TrimSpace(recordParts[0])
						recordValue := strings.TrimSpace(recordParts[1])
						resourceAttrs = append(resourceAttrs, fmt.Sprintf("%s=%s", recordKey, recordValue))
					}
				}
			}
		}
	}

	onEndSection()

	// Combine resource attributes
	if len(resourceAttrs) > 0 {
		config.ResourceAttributes = strings.Join(resourceAttrs, ",")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	return config, nil
}
