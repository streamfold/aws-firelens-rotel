import itertools
import json

from rotel_sdk.open_telemetry.common.v1 import AnyValue, KeyValue
from rotel_sdk.open_telemetry.logs.v1 import ResourceLogs

def process_logs(resource_logs: ResourceLogs):
    """
    Parses JSON log bodies and sets fields based on the properties of the JSON log.
    The remaining top-level keys are set as attributes on the log record.
    """

    for log_record in itertools.chain.from_iterable(
        scope_log.log_records for scope_log in resource_logs.scope_logs
    ):
        try:
            if not log_record.body:
                continue
            
            if not isinstance(log_record.body.value, str):
                continue
                
            if not log_record.body.value.startswith("{"):
                continue

            log = json.loads(log_record.body.value)

            # Body may be in message, msg, or log
            for key in ["message", "msg", "log"]:
                if key in log and log[key]:
                    log_record.body = AnyValue(log[key])
                    del log[key]
                    break
            # Extract trace and span ID's if available
            if "trace_id" in log and log["trace_id"]:
                log_record.trace_id = bytes.fromhex(log["trace_id"])
                del log["trace_id"]
            if "span_id" in log and log["span_id"]:
                log_record.span_id = bytes.fromhex(log["span_id"])
                del log["span_id"]

            # Extract log level if available
            # TODO: Use a contstant for these
            if "level" in log and log["level"]:
                level = log["level"].lower()
                if level == "info":
                    log_record.severity_number = 9
                elif level == "warn":
                    log_record.severity_number = 13
                elif level == "error":
                    log_record.severity_number = 17
                elif level == "debug":
                    log_record.severity_number = 5
                elif level == "fatal":
                    log_record.severity_number = 21
                elif level == "trace":
                    log_record.severity_number = 1

                log_record.severity_text = log["level"]
                del log["level"]

            # Parse and remove timestamps
            for key in ["timestamp", "ts"]:
                if key in log and log[key]:
                    # Parse timestamp - could be RFC3339 or integer seconds
                    timestamp_value = log[key]
                    if isinstance(timestamp_value, str):
                        # Try parsing as RFC3339 format
                        import datetime

                        try:
                            dt = datetime.datetime.fromisoformat(
                                timestamp_value.replace("Z", "+00:00")
                            )
                            timestamp_ns = int(dt.timestamp() * 1_000_000_000)
                        except ValueError:
                            # If RFC3339 parsing fails, try as string representation of number
                            try:
                                timestamp_ns = int(
                                    float(timestamp_value) * 1_000_000_000
                                )
                            except ValueError:
                                continue
                    else:
                        # Assume it's a numeric value in seconds
                        timestamp_ns = int(float(timestamp_value) * 1_000_000_000)

                    log_record.time_unix_nano = timestamp_ns
                    del log[key]
                    break

            # Add remaining log fields as attributes
            for key, value in log.items():
                log_record.attributes.append(KeyValue(key, AnyValue(value)))

        except Exception as _e:
            pass
