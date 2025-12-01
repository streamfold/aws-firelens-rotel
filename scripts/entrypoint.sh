#!/bin/bash
#

LAUNCHER_PATH="/usr/local/bin/launcher"
ROTEL_PATH="/usr/local/bin/rotel"
ROTEL_VENV="/rotel-venv"

FLUENTBIT_CONFIG=${FLUENTBIT_CONFIG:-"/fluent-bit/etc/fluent-bit.conf"}

if [ ! -f "$FLUENTBIT_CONFIG" ]; then
    echo "Error: Can not find configuration at ${FLUENTBIT_CONFIG}: Did you set {\"firelensConfiguration\": {\"type\": \"fluentbit\"}} ?"
    exit 1
fi

source ${ROTEL_VENV}/bin/activate

# TODO: More initial config setup here?

exec $LAUNCHER_PATH --fluent-bit-config ${FLUENTBIT_CONFIG} --rotel-path ${ROTEL_PATH}
