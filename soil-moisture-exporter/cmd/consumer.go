package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"cstdev.co.uk/soil-moisture-exporter/pkg/config"
	"cstdev.co.uk/soil-moisture-exporter/pkg/processors"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/cstdev/go-helpers/pkg/initialise"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.WithFields(log.Fields{
		"message": msg.Payload(),
		"topic":   msg.Topic(),
	}).Info("Received message")
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Info("Connected")
	subscribe(client)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Error("Connection lost", err)
}

var (
	moisture = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "moisture_percentage",
		Help: "Current moisture level",
	}, []string{"sensor"})

	thresholdGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "moisture_below_threshold",
		Help: "Is the moisture level below the threshold",
	}, []string{"sensor"})

	readingCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "reading_count",
		Help: "Number of readings",
	}, []string{"sensor"})
)

var cfg config.ConfigData

func main() {
	initialise.SetupLogging()
	var configFilePath string
	flag.StringVar(&configFilePath, "config", "poller.yaml", "path of the yaml config file")
	flag.Parse()
	c, err := config.ReadConfig(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	cfg = c

	clientId := uuid.New()
	broker := cfg.MQTT.Broker
	port := cfg.MQTT.Port
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", c.MQTT.Protocol, broker, port))
	opts.SetClientID(clientId.String())
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.SetUsername(cfg.MQTT.Username)
	opts.SetPassword(cfg.MQTT.Password)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Error(token.Error())
		return
	}

	// Metrics setup
	prometheus.MustRegister(moisture)
	prometheus.MustRegister(thresholdGauge)
	prometheus.MustRegister(readingCounter)
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		log.Fatal(http.ListenAndServe(":2112", nil))
	}()

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	<-termChan
	log.Info("Shutting down")

	client.Disconnect(250)
	os.Exit(0)
}

func subscribe(client mqtt.Client) {
	db, err := sql.Open("sqlite3", cfg.DatabaseFile)
	if err != nil {
		log.WithField("error", err).Fatal("unable to connect to db")
		return
	}

	processor := processors.SoilChecker{
		Database:        db,
		Metrics:         moisture,
		BreachingMetric: thresholdGauge,
		ReadingCounter:  readingCounter,
	}

	topic := "soilMoisture"
	log.Info("Subscribing")
	token := client.Subscribe(topic, cfg.MQTT.QOS, processor.HandleMessage)
	log.Debug("Waiting on token")
	token.Wait()
	log.Debug("Token Wait returned")
	if token.Error() != nil {
		log.Error("Failed to subscribe to topic")
		log.Fatal(token.Error())
	}

	log.WithField("topic", topic).Info("Subscribed")
}
