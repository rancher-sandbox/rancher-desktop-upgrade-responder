#!/bin/sh
docker run -d -p 8086:8086 \
      -e INFLUXDB_ADMIN_USER=admin \
      -e INFLUXDB_ADMIN_PASSWORD=password \
      influxdb:1.8