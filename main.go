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
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/rcrowley/go-metrics"
	standardLog "log"
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
	routingAddress := app.String(cli.StringOpt{
		Name:   "vulcan_addr",
		Value:  "https://vulcan-address",
		Desc:   "Address used for routing requests",
		EnvVar: "ROUTING_ADDRESS",
	})
	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "APP_PORT",
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
	consumerOffset := app.String(cli.StringOpt{
		Name:   "consumer_offset",
		Value:  "",
		Desc:   "Kafka read offset.",
		EnvVar: "OFFSET",
	})
	consumerAutoCommitEnable := app.Bool(cli.BoolOpt{
		Name:   "consumer_autocommit_enable",
		Value:  true,
		Desc:   "Enable autocommit for small messages.",
		EnvVar: "COMMIT_ENABLE",
	})
	consumerQueue := app.String(cli.StringOpt{
		Name:   "consumer_queue_id",
		Value:  "",
		Desc:   "The kafka queue id",
		EnvVar: "QUEUE_ID",
	})
	graphiteTCPAddress := app.String(cli.StringOpt{
		Name:   "graphite-tcp-address",
		Value:  "",
		Desc:   "Graphite TCP address, e.g. graphite.ft.com:2003. Leave as default if you do NOT want to output to graphite (e.g. if running locally)",
		EnvVar: "GRAPHITE_TCP_ADDRESS",
	})
	graphitePrefix := app.String(cli.StringOpt{
		Name:   "graphite-prefix",
		Value:  "",
		Desc:   "Prefix to use. Should start with content, include the environment, and the host name. e.g. coco.pre-prod.special-reports-rw-neo4j.1",
		EnvVar: "GRAPHITE_PREFIX",
	})

	log.SetLevel(log.InfoLevel)
	log.Infof("[Startup] smartlogic-concordance-transformer is starting ")

	app.Action = func() {
		log.Infof("System code: %s, App Name: %s, Port: %s", *appSystemCode, *appName, *port)
		consumerConfig := queueConsumer.QueueConfig{
			Addrs:                []string{*routingAddress},
			Group:                *groupName,
			Queue:                *consumerQueue,
			Topic:                *topic,
			Offset:               *consumerOffset,
			AutoCommitEnable:     *consumerAutoCommitEnable,
			ConcurrentProcessing: true,
		}
		outputMetricsIfRequired(*graphiteTCPAddress, *graphitePrefix, true)

		router := mux.NewRouter()
		transformer := slc.NewTransformerService(*topic, *writerAddress, &httpClient)
		queueService := slc.NewQueueService(consumerConfig, &httpClient)
		handler := slc.NewHandler(transformer, queueService)
		handler.RegisterHandlers(router)
		handler.RegisterAdminHandlers(router)

		go handler.SubscribeToQueue(httpClient)

		if err := http.ListenAndServe(":" + *port, nil); err != nil {
			log.Fatalf("Unable to start server: %v\n", err)
		}
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("App could not start, error=[%s]\n", err)
		return
	}
}

func outputMetricsIfRequired(graphiteTCPAddress string, graphitePrefix string, logMetrics bool) {
	if graphiteTCPAddress != "" {
		addr, _ := net.ResolveTCPAddr("tcp", graphiteTCPAddress)
		go graphite.Graphite(metrics.DefaultRegistry, 5*time.Second, graphitePrefix, addr)
	}
	if logMetrics { //useful locally
		//messy use of the 'standard' log package here as this method takes the log struct, not an interface, so can't use logrus.Logger
		go metrics.Log(metrics.DefaultRegistry, 60*time.Second, standardLog.New(os.Stdout, "metrics", standardLog.Lmicroseconds))
	}
}