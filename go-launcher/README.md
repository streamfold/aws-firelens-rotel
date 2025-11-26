# Go Launcher for Rotel

A Go CLI wrapper that parses Fluent Bit configuration files and executes the Rotel application with the appropriate environment variables.

## Features

- Parses Fluent Bit configuration files (`.conf` format)
- Extracts relevant configuration and sets environment variables
- Executes the Rotel binary with the configured environment
- Uses only Go standard library (no external dependencies)

## Building

### Using Make (Recommended)

```bash
# Download dependencies and build
make all

# Or run steps separately
make deps   # Download and verify dependencies
make build  # Build the binary
```

### Using Go directly

```bash
go build -o launcher
```

### Available Make Targets

Run `make help` to see all available targets:

```bash
make help
```

Common targets:
- `make all` - Download dependencies and build the project
- `make deps` - Download and verify dependencies
- `make build` - Build the launcher binary
- `make test` - Run unit tests with race detection
- `make test-integration` - Run integration tests
- `make check` - Run all checks (lint and test)
- `make clean` - Remove build artifacts
- `make install` - Install binary to /usr/local/bin
```

## Usage

```bash
./launcher --fluent-bit-config <path-to-config> --rotel-path <path-to-rotel> [additional args for rotel]
```

### Options

- `--fluent-bit-config`: Path to the Fluent Bit configuration file (required)
- `--rotel-path`: Path to the Rotel executable (required)

Any additional arguments after the flags will be passed through to the Rotel executable.

## Environment Variables Set

The launcher parses the Fluent Bit configuration and sets the following environment variables before executing Rotel:

### ROTEL_FIRELENS_RECEIVER_ENDPOINT

Extracted from the `[INPUT]` section with `Name forward` that has both `Listen` and `Port` fields.

Format: `<Listen>:<Port>`

Example: `127.0.0.1:24224`

### ROTEL_FIRELENS_RECEIVER_SOCKET

Extracted from the `[INPUT]` section with `Name forward` that has a `unix_path` field.

Format: Path to Unix socket

Example: `/var/run/fluent.sock`

### ROTEL_RESOURCE_ATTRIBUTES

Extracted from the `[FILTER]` section with `Name record_modifier`. All `Record` entries are collected as key=value pairs separated by commas.

Format: `key1=value1,key2=value2,...`

Example: `ecs_cluster=firelenstest,ecs_task_arn=arn:aws:ecs:us-east-2:279234357137:task/firelenstest/72aa3d1989dc4561b975631f36170c09,ecs_task_definition=firelens-test1:8`

## Example

Given a Fluent Bit configuration file `fluent-bit.conf`:

```
[INPUT]
    Name forward
    Listen 127.0.0.1
    Port 24224
[INPUT]
    Name forward
    unix_path /var/run/fluent.sock
[FILTER]
    Name record_modifier
    Match *
    Record ecs_cluster firelenstest
    Record ecs_task_arn arn:aws:ecs:us-east-2:279234357137:task/firelenstest/72aa3d1989dc4561b975631f36170c09
    Record ecs_task_definition firelens-test1:8
```

Run the launcher:

```bash
./launcher --fluent-bit-config fluent-bit.conf --rotel-path /usr/local/bin/rotel
```

This will set:
- `ROTEL_FIRELENS_RECEIVER_ENDPOINT=127.0.0.1:24224`
- `ROTEL_FIRELENS_RECEIVER_SOCKET=/var/run/fluent.sock`
- `ROTEL_RESOURCE_ATTRIBUTES=ecs_cluster=firelenstest,ecs_task_arn=arn:aws:ecs:us-east-2:279234357137:task/firelenstest/72aa3d1989dc4561b975631f36170c09,ecs_task_definition=firelens-test1:8`

And then execute the Rotel binary at `/usr/local/bin/rotel`.

## Testing

You can test the launcher with a dummy script to verify environment variables:

```bash
# Create a test script
cat > test-rotel.sh << 'EOF'
#!/bin/bash
echo "ROTEL_FIRELENS_RECEIVER_ENDPOINT=$ROTEL_FIRELENS_RECEIVER_ENDPOINT"
echo "ROTEL_FIRELENS_RECEIVER_SOCKET=$ROTEL_FIRELENS_RECEIVER_SOCKET"
echo "ROTEL_RESOURCE_ATTRIBUTES=$ROTEL_RESOURCE_ATTRIBUTES"
EOF

chmod +x test-rotel.sh

# Run the launcher with the test script
./launcher --fluent-bit-config example-fluent-bit.conf --rotel-path ./test-rotel.sh
```

## Project Structure

```
go-launcher/
├── main.go                    # Main CLI application and execution logic
├── parser.go                  # Fluent Bit configuration parser
├── example-fluent-bit.conf    # Example configuration file
├── go.mod                     # Go module definition
└── README.md                  # This file
```
