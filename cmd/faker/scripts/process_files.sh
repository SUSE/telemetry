#!/bin/bash

FILES_COUNT=${1:-30}
CUSTOMER_ID=${2:-123456789}
SERVER_URL="http://192.168.0.143:9999/telemetry"
CONFIG="$(pwd)/cmd/faker/config/client.yaml"
TELEMETRY_FILES_DIR="$(pwd)/cmd/faker/fake_telemetry_data"

sed -e "s|__SERVER_URL__|${SERVER_URL}|g" -e "s|__CUSTOMER_ID__|${CUSTOMER_ID}|g" $CONFIG_TEMPLATE > $CONFIG

# authenticate
cd cmd/authenticator
go run . --config=$CONFIG

# submit
cd ../generator

for i in $(seq 1 $FILES_COUNT); do
  FILE="$TELEMETRY_FILES_DIR/telemetry_data_${i}.json"
  if [ -f "$FILE" ]; then
    echo "Processing $FILE with config $CONFIG"
    go run . --config "$CONFIG" --telemetry=FAKER-Telemetry-Data --tag DEVTEST "$FILE"
  else
    echo "File $FILE does not exist. Skipping."
  fi
done
