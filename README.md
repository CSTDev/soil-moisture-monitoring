# Soil Moisture Monitoring

This repository contains the code that I use to monitor the soil of my house plants at home and alert me when they need watering.

It is made up of two parts:

1. An Arduino Sketch for an ESP8266 with a connected moisture sensor
2. A service that reads off MQTT and exposes prometheus metrics. 

## 2. Soil Moisture Exporter

A service written in Go that reads from MQTT and publishes prometheus metrics. It uses the threshold defined for a sensor in an SQLite database to provide a binary ok/not ok value to monitor and alert off of.

The metrics it exposes are:

| Metric | Type | Labels | Description |
|---|---|---|---|
| `moisture_percentage` | Gauge | `sensor` | Current moisture level as a percentage for the given sensor |
| `moisture_below_threshold` | Gauge | `sensor` | Whether the moisture level is below the configured threshold — `1` if breaching, `0` if ok |
| `reading_count` | Counter | `sensor` | Total number of readings received from the given sensor |

Metrics are served on port `2112` at the `/metrics` endpoint.

### SQLite Database Setup

The service requires an SQLite database to store readings and sensor thresholds. If you don't have one yet, create an empty database file using the `sqlite3` CLI:

```sh
sqlite3 /data/soilConsumer/config/iot.db
```

This opens an interactive shell. You can then run the following statements to initialise the schema:

```sql
CREATE TABLE soilmoisture(timestamp INTEGER, sensorName TEXT, value REAL, PRIMARY KEY(timestamp, sensorName));
CREATE TABLE sensors (id TEXT PRIMARY KEY, threshold int);
```

To register a sensor and set its moisture threshold (the minimum value before an alert is triggered), insert a row into the `sensors` table:

```sql
INSERT INTO sensors (id, threshold) VALUES ('my-sensor-name', 50);
```

The `id` must match the `name` field in the MQTT payload published by the ESP8266.

### Running with Docker Compose

The service is configured via a YAML file (see [sample-config.yml](soil-moisture-exporter/sample-config.yml)). Mount the config and database into the container:

```yaml
soilconsumer:
  image: soil-consumer:0.3.0-arm32v7
  command: -config /app/config/poller.yaml
  volumes:
    - /data/soilConsumer/config:/app/config      # contains poller.yaml
    - /data/soilConsumer/config:/app/data/iot.db # sqlite database file

otelcollector:
  image: otel/opentelemetry-collector-contrib@sha256:0076e6c250bef16968e29e8f94177b87b5a851c79ce14b270e657f5f655b9e04
  volumes:
    - /data/otel/config/config.yaml:/etc/otelcol-contrib/config.yaml
```

The OpenTelemetry Collector can be used to scrape the `/metrics` endpoint and forward data to a backend of your choice.
