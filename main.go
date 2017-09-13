package main

import (
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Financial-Times/kafka-client-go/kafka"
	slc "github.com/Financial-Times/smartlogic-concordance-transformer/smartlogic"
	log "github.com/sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	_ "github.com/joho/godotenv/autoload"
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
	logLevel := app.String(cli.StringOpt{
		Name:   "logLevel",
		Value:  "INFO",
		Desc:   "Log level",
		EnvVar: "LOG_LEVEL",
	})
	brokerConnectionString := app.String(cli.StringOpt{
		Name:   "brokerConnectionString",
		Desc:   "Zookeeper connection string in the form host1:2181,host2:2181/chroot",
		EnvVar: "BROKER_CONNECTION_STRING",
	})
	topic := app.String(cli.StringOpt{
		Name:   "topic",
		Value:  "SmartLogicConcepts",
		Desc:   "Kafka topic subscribed to",
		EnvVar: "KAFKA_TOPIC",
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

	lvl, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("Cannot parse log level: %s", *logLevel)
	}
	log.SetLevel(lvl)
	log.SetFormatter(&log.JSONFormatter{})

	log.WithFields(log.Fields{
		"WRITER_ADDRESS":           *writerAddress,
		"KAFKA_TOPIC":              *topic,
		"GROUP_NAME":               *groupName,
		"BROKER_CONNECTION_STRING": *brokerConnectionString,
	}).Infof("[Startup] smartlogic-concordance-transformer is starting ")

	app.Action = func() {
		log.Infof("System code: %s, App Name: %s, Port: %s", *appSystemCode, *appName, *port)

		consumerConfig := kafka.DefaultConsumerConfig()
		consumer, err := kafka.NewConsumer(*brokerConnectionString, *groupName, []string{*topic}, consumerConfig)
		if err != nil {
			log.WithError(err).Fatal("Cannot create Kafka client")
		}

		router := mux.NewRouter()
		transformer := slc.NewTransformerService(*topic, *writerAddress, &httpClient)
		handler := slc.NewHandler(transformer, consumer)
		handler.RegisterHandlers(router)
		handler.RegisterAdminHandlers(router)

		go func() {
			if err := http.ListenAndServe(":"+*port, nil); err != nil {
				log.WithError(err).Fatal("Unable to start server")
			}
		}()

		consumer.StartListening(handler.ProcessKafkaMessage)

		waitForSignal()
		log.Info("Shutting down Kafka consumer")
		consumer.Shutdown()
		log.Info("Stopping application")
	}

	runErr := app.Run(os.Args)
	if runErr != nil {
		log.Errorf("App could not start, error=[%s]\n", runErr)
		return
	}
}

func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
