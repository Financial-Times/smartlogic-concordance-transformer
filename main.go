package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/jawher/mow.cli"
	"os"
	"os/signal"
	"syscall"
)

const appDescription = "Service to transform the JSON-LD from Smart Logic to an UPP source system representation of a " +
	"concordance and send it to the concordances-rw-s3"

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
	//kafkaAddress := app.String(cli.StringOpt{
	//	Name:   "kafka-address",
	//	Value:  "http://localhost:9092",
	//	Desc:   "Kafka broker address",
	//	EnvVar: "KAFKA_ADDR",
	//})
	//kafkaTopic := app.String(cli.StringOpt{
	//	Name:   "kafka-topic",
	//	Value:  "SmartLogicChangeEvents",
	//	Desc:   "Kafka topic subscribed to",
	//	EnvVar: "TOPIC",
	//})
	//kafkaGroup := app.String(cli.StringOpt{
	//	Name:   "kafka-group",
	//	Desc:   "Kafka topic subscribed to",
	//	EnvVar: "TOPIC",
	//})
	//writerEndpoint := app.String(cli.StringOpt{
	//	Name:   "writerEndpoint",
	//	Value:  "http://localhost:8080",
	//	Desc:   "Endpoint for the concordance RW app.",
	//	EnvVar: "WRITER_ENDPOINT",
	//})

	logLevel := app.String(cli.StringOpt{
		Name:   "logLevel",
		Value:  "INFO",
		Desc:   "Log level",
		EnvVar: "LOG_LEVEL",
	})

	lvl, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("Cannot parse log level: %s", *logLevel)
	}
	log.SetLevel(lvl)

	log.Infof("[Startup] smartlogic-concordance-transformer is starting ")

	app.Action = func() {
		log.Infof("System code: %s, App Name: %s, Port: %s", *appSystemCode, *appName, *port)

		// DO stuff
		waitForSignal()
	}
	errs := app.Run(os.Args)
	if errs != nil {
		log.Errorf("App: smartlogic-concordance-transformer could not start, error=[%s]\n", err)
		return
	}
}


func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}

