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
	"github.com/wvanbergen/kafka/consumergroup"
	"github.com/Shopify/sarama"
	"github.com/wvanbergen/kazoo-go"
	"fmt"
	"strings"
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

		config := consumergroup.NewConfig()
		config.Offsets.Initial = sarama.OffsetOldest
		config.Offsets.ProcessingTimeout = 10 * time.Second
		fmt.Printf("Group name is %s\n", *groupName)
		fmt.Printf("Topic is %s\n", *topic)
		fmt.Printf("Kafka address is %s\n", *kafkaHost + ":" + *kafkaPort)

		zookeeperNodes, chroot := kazoo.ParseConnectionString(*kafkaHost + ":" + *kafkaPort)
		config.Zookeeper.Chroot = chroot
		config.Zookeeper.Timeout = 10 * time.Second
		kafkaTopics := strings.Split(*topic, ",")
		fmt.Printf("Topic is %s\n", kafkaTopics)
		fmt.Printf("Kafka address is %s\n", zookeeperNodes)

		//config := cluster.NewConfig()
		//config.Consumer.Return.Errors = true
		//config.Group.Return.Notifications = true
		//
		//// init consumer
		//brokers := []string{*kafkaHost + ":" + *kafkaPort}
		//topics := []string{*topic}
		//consumer, err := cluster.NewConsumer(brokers, *groupName, topics, config)

		consumer, err := consumergroup.JoinConsumerGroup(*groupName, kafkaTopics, zookeeperNodes, config)
		if err != nil {
			log.Error("Error creating kafka consumer: %v", err)
			return
		}

		router := mux.NewRouter()
		transformer := slc.NewTransformerService(*consumer, *topic, *writerAddress, &httpClient)
		handler := slc.NewHandler(transformer)
		handler.RegisterHandlers(router)
		handler.RegisterAdminHandlers(router)

		go handler.Run()

		defer func() {
			if cErr := consumer.Close(); cErr != nil {
				log.Fatal(cErr)
				}
			}()
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("App could not start, error=[%s]\n", err)
		return
	}
}