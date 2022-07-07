package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v3"
	slc "github.com/Financial-Times/smartlogic-concordance-transformer/smartlogic"
	"github.com/gorilla/mux"
	cli "github.com/jawher/mow.cli"
)

const appDescription = "Service which listens to kafka for concordance updates, transforms smartlogic concordance json and sends updates to concordances-rw-neo4j"

var httpClient = http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 128,
	},
}

func main() {
	app := cli.App("smartlogic-concordance-transformer", appDescription)

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "smartlogic-concordance-transform",
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
	topic := app.String(cli.StringOpt{
		Name:   "topic",
		Value:  "SmartlogicConcept",
		Desc:   "Kafka topic subscribed to",
		EnvVar: "KAFKA_TOPIC",
	})
	groupName := app.String(cli.StringOpt{
		Name:   "groupName",
		Value:  "SmartlogicConcordanceTransformer",
		Desc:   "Group name of connection to the Kafka topic",
		EnvVar: "GROUP_NAME",
	})
	writerAddress := app.String(cli.StringOpt{
		Name:   "writerAddress",
		Desc:   "Concordance rw address for routing requests",
		EnvVar: "WRITER_ADDRESS",
	})
	kafkaAddress := app.String(cli.StringOpt{
		Name:   "kafkaAddress",
		Value:  "kafka:9092",
		Desc:   "Address used to connect to Kafka",
		EnvVar: "KAFKA_ADDR",
	})
	consumerLagTolerance := app.Int(cli.IntOpt{
		Name:   "consumerLagTolerance",
		Value:  120,
		Desc:   "Kafka lag tolerance",
		EnvVar: "KAFKA_LAG_TOLERANCE",
	})

	log := logger.NewUPPLogger(*appName, *logLevel)

	app.Action = func() {
		log.WithFields(map[string]interface{}{
			"KAFKA_ADDRESS": *kafkaAddress,
			"KAFKA_TOPIC":   *topic,
			"GROUP_NAME":    *groupName,
		}).Infof("[Startup] %s is starting", *appName)

		log.Infof("System code: %s, App Name: %s, Port: %s", *appSystemCode, *appName, *port)

		consumerConfig := kafka.ConsumerConfig{
			BrokersConnectionString: *kafkaAddress,
			ConsumerGroup:           *groupName,
			ConnectionRetryInterval: time.Minute,
		}
		topics := []*kafka.Topic{
			kafka.NewTopic(*topic, kafka.WithLagTolerance(int64(*consumerLagTolerance))),
		}
		consumer := kafka.NewConsumer(consumerConfig, topics, log)

		transformer := slc.NewTransformerService(*topic, *writerAddress, &httpClient, log)
		handler := slc.NewHandler(transformer, consumer, log)

		router := mux.NewRouter()
		handler.RegisterHandlers(router)
		handler.RegisterAdminHandlers(router, *appSystemCode, *appName, appDescription)

		go func() {
			if err := http.ListenAndServe(":"+*port, nil); err != nil {
				log.WithError(err).Fatal("Unable to start server")
			}
		}()

		go consumer.Start(handler.ProcessKafkaMessage)
		defer func(consumer *kafka.Consumer) {
			log.Info("Shutting down Kafka consumer")
			err := consumer.Close()
			if err != nil {
				log.WithError(err).Error("Could not close kafka consumer")
			}
		}(consumer)

		waitForSignal()
		log.Info("Stopping application")
	}

	if runErr := app.Run(os.Args); runErr != nil {
		log.Errorf("App could not start, error=[%s]\n", runErr)
		return
	}
}

func waitForSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
