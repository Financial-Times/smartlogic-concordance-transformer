package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/jawher/mow.cli"
	"net/http"
	"os"
	"github.com/Shopify/sarama"
	"github.com/gorilla/mux"
	"net"
	"time"
	slc "github.com/Financial-Times/smartlogic-concordance-transformer/smartlogicconcordance"
)

const appDescription = "Service which listens to kafka for concordance updates, transforms smartlogic concordance json and sends updates to concordance-rw-dynamodb"

var httpClient = http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 128,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
	},
}
func main() {
	app := cli.App("smartlogic-concordance-transformer", appDescription)

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "smartlogic-concordance-transformer",
		Desc:   "System Code of the application",
		EnvVar: "APP_SYSTEM_CODE",
	})
	appName := app.String(cli.StringOpt{
		Name:   "app-name",
		Value:  "Smartlogic Concordance Transformer",
		Desc:   "Application name",
		EnvVar: "APP_NAME",
	})
	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "APP_PORT",
	})
	kafkaAddress := app.String(cli.StringOpt{
		Name:   "kafka_addr",
		Value:  "http://localhost:9092",
		Desc:   "Kafka broker address",
		EnvVar: "KAFKA_ADDR",
	})
	topic := app.String(cli.StringOpt{
		Name:   "topic",
		Value:  "SmartLogicChangeEvents",
		Desc:   "Kafka topic subscribed to",
		EnvVar: "TOPIC",
	})
	vulcanAddress := app.String(cli.StringOpt{
		Name:   "vulcanAddress",
		Value:  "http://localhost:8080/",
		Desc:   "Vulcan address for routing requests",
		EnvVar: "VULCAN_ADDR",
	})

	log.SetLevel(log.InfoLevel)
	log.Infof("[Startup] smartlogic-concordance-transformer is starting ")

	app.Action = func() {
		log.Infof("System code: %s, App Name: %s, Port: %s", *appSystemCode, *appName, *port)

		consumer, err := sarama.NewConsumer([]string{*kafkaAddress}, sarama.NewConfig())
		if err != nil {
			log.Fatal(err)
		}

		router := mux.NewRouter()
		transformer := slc.NewTransformerService(consumer, *topic, *vulcanAddress, &httpClient)
		handler := slc.NewHandler(transformer)
		handler.RegisterHandlers(router)
		handler.RegisterAdminHandlers(router)

		go handler.Run()
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("App could not start, error=[%s]\n", err)
		return
	}
}