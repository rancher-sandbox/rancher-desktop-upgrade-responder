#!/bin/sh
REPO_ROOT=$(git rev-parse --show-toplevel)
$REPO_ROOT/bin/upgrade-responder --debug start \
      --upgrade-response-config $REPO_ROOT/rancherdesktop/sample-config.json \
      --application-name postman \
      --influxdb-url http://localhost:8086 \
      --influxdb-user admin \
      --influxdb-pass password \
      --geodb $REPO_ROOT/package/GeoLite2-City.mmdb