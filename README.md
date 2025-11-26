# AWS FireLens Rotel

A lightweight, high-performance integration that combines [AWS FireLens](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/using_firelens.html) with [Rotel](https://rotel.dev), a high-performance and resource-efficient OpenTelemetry collection data plane written in Rust. This project enables seamless collection and forwarding of container logs and metrics from Amazon ECS tasks to OpenTelemetry-compatible backends.

This replaces the use of Fluent Bit or Fluentd as the log_router container.

**NOTE**: This is an early release built for experimenting. Please take that in mind and share use cases!

## Benefits

* Native OpenTelmetry support, no need for additional containers/collectors
* High performance (benchmarks coming)
* Much smaller container size (less than half the AWS Fluent Bit container size)

## Overview

AWS FireLens provides native integration with Fluent Bit for log routing in ECS tasks. This project provides an alternative
based on the Rotel project.

* Uses a small Go launcher to read the Fluent Bit configuration file to parse out the listen address and Unix socket path
* Go launcher sets appropriate environment variables for Rotel before launching it
* Just switch the container name and set Rotel exporter configuration

Rotel automatically starts a native OpenTelemetry reciever on localhost ports 4317 (gRPC) and 4318 (HTTP). ECS logs are
automatically converted to OTLP log format.

## Quick Start

### Using with AWS ECS

1. Add the FireLens Rotel container to your ECS task definition.
2. Set Rotel environment variable [configuration](https://rotel.dev/docs/category/configuration) variables on the _log_router_ container.

For example, to export logs, metrics and traces to ClickHouse Cloud:

```json
{
  "family": "my-task",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "my-app:latest",
      "logConfiguration": {
        "logDriver": "awsfirelens",
        "options": {}
      }
    },
    {
      "name": "log-router",
      "image": "streamfold/aws-firelens-rotel:latest",
      "firelensConfiguration": {
        "type": "fluentbit"
      },
      "environment": [
        {
          "name": "ROTEL_CLICKHOUSE_EXPORTER_ENDPOINT",
          "value": "https://xxxx.us-east-1.aws.clickhouse.cloud:8443"
        },
        {
          "name": "ROTEL_CLICKHOUSE_EXPORTER_USER",
          "value": "default"
        },
        {
          "name": "ROTEL_CLICKHOUSE_EXPORTER_PASSWORD",
          "value": "my-password"
        },
        {
          "name": "ROTEL_EXPORTER",
          "value": "clickhouse"
        }
      ]
    }
  ]
}
```

## Configuration

### Environment Variables Set by Go launcher

The launcher automatically sets these environment variables based on the Fluent Bit configuration:

| Variable | Description | Example |
|----------|-------------|---------|
| `ROTEL_FIRELENS_RECEIVER_ENDPOINT` | TCP endpoint for Fluent Bit forward protocol | `127.0.0.1:24224` |
| `ROTEL_FIRELENS_RECEIVER_SOCKET` | Unix socket path for Fluent Bit forward protocol | `/var/run/fluent.sock` |
| `ROTEL_OTEL_RESOURCE_ATTRIBUTES` | Resource attributes from ECS attributes | `ecs_cluster=prod,ecs_task_arn=arn:aws:...` |

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

---

Made with ❤️ by [Streamfold](https://streamfold.com)
