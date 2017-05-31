package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/jawher/mow.cli"
	"net/http"
	"os"
	"github.com/gorilla/mux"
	"net"
	"time"
	slc "github.com/Financial-Times/smartlogic-concordance-transformer/smartlogic"
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
	kafkaHost := app.String(cli.StringOpt{
		Name:   "kafkaHost",
		Value:  "localhost",
		Desc:   "Kafka broker host",
		EnvVar: "KAFKA_HOST",
	})
	kafkaPort := app.String(cli.StringOpt{
		Name:   "kafkaPort",
		Value:  "9092",
		Desc:   "Kafka broker port",
		EnvVar: "KAFKA_PORT",
	})
	topic := app.String(cli.StringOpt{
		Name:   "topic",
		Value:  "SmartLogicChangeEvents",
		Desc:   "Kafka topic subscribed to",
		EnvVar: "TOPIC",
	})
	groupName := app.String(cli.StringOpt{
		Name:   "groupName",
		Value:  "SmartlogicConcordanceSemantic",
		Desc:   "Group name of connection to SmartLogicChangeEvents Topic",
		EnvVar: "GROUP_NAME",
	})
	writerAddress := app.String(cli.StringOpt{
		Name:   "writerAddress",
		Value:  "http://localhost:8080/__concordance-rw-dynamodb/",
		Desc:   "Concordance rw address for routing requests",
		EnvVar: "WRITER_ADDRESS",
	})

	log.SetLevel(log.InfoLevel)
	log.Infof("[Startup] smartlogic-concordance-transformer is starting ")

	app.Action = func() {
		log.Infof("System code: %s, App Name: %s, Port: %s", *appSystemCode, *appName, *port)

		router := mux.NewRouter()
		transformer := slc.NewTransformerService(*consumer, *topic, *writerAddress, &httpClient)
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