package processors

import (
	"database/sql"
	"encoding/json"
	"time"

	"cstdev.co.uk/soil-moisture-exporter/pkg/soil"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type Processor interface {
	HandleMessage(client mqtt.Client, msg mqtt.Message)
}

type SoilChecker struct {
	Database        *sql.DB
	Metrics         *prometheus.GaugeVec
	BreachingMetric *prometheus.GaugeVec
	ReadingCounter  *prometheus.CounterVec
}

func (c *SoilChecker) HandleMessage(client mqtt.Client, msg mqtt.Message) {
	var reading soil.Reading
	err := json.Unmarshal(msg.Payload(), &reading)
	if err != nil {
		log.WithField("error", err).Error("failed to unmarshal message")
		return
	}
	log.WithField("payload", reading).Info("received message")

	log.WithFields(log.Fields{
		"value":  reading.Value,
		"sensor": reading.SensorName,
	}).Info("got reading")
	now := time.Now()

	c.Metrics.With(prometheus.Labels{"sensor": reading.SensorName}).Set(reading.Value)
	c.ReadingCounter.With(prometheus.Labels{"sensor": reading.SensorName}).Inc()

	err = c.storeReading(now.Unix(), reading.SensorName, reading.Value)
	if err != nil {
		log.WithField("error", err).Error("failed to store reading")
	}

	threshold, err := c.getSensorThreshold(reading.SensorName)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"sensor": reading.SensorName,
		}).Error("failed to get sensor threshold")
		return
	}

	log.WithFields(log.Fields{
		"sensor":    reading.SensorName,
		"value":     reading.Value,
		"threshold": threshold,
	}).Info("reading and threshold")
	if reading.Value < float64(threshold) {
		c.BreachingMetric.With(prometheus.Labels{"sensor": reading.SensorName}).Set(1)
	} else {
		c.BreachingMetric.With(prometheus.Labels{"sensor": reading.SensorName}).Set(0)
	}
}

func (c *SoilChecker) storeReading(timestamp int64, sensorName string, value float64) error {
	_, err := c.Database.Exec("INSERT INTO soilmoisture VALUES(?,?,?);", timestamp, sensorName, value)
	return err
}

func (c *SoilChecker) getSensorThreshold(sensorName string) (int, error) {
	log.Info(sensorName)
	row := c.Database.QueryRow("SELECT threshold FROM sensors WHERE id = ?;", sensorName)
	var threshold int
	var err error
	if err = row.Scan(&threshold); err != nil {
		if err == sql.ErrNoRows {
			log.WithField("sensor name", sensorName).Warn("sensor not found, won't send notification")
			return 0, nil
		}
		return 0, err
	}
	return threshold, err
}
