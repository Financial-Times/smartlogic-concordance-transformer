package smartlogic

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	log "github.com/Sirupsen/logrus"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/rcrowley/go-metrics"
	"fmt"
	"github.com/Financial-Times/go-fthealth"
	"net/http"
	"os"
	"os/signal"
	"io/ioutil"
	"github.com/Financial-Times/transactionid-utils-go"
	"github.com/Shopify/sarama"
)

type SmartlogicConcordanceTransformerHandler struct {
	service TransformerService
}

func NewHandler(service TransformerService) SmartlogicConcordanceTransformerHandler {
	return SmartlogicConcordanceTransformerHandler{
		service: service,
	}
}

//func (h *SmartlogicConcordanceTransformerHandler) Run() {
//	fmt.Println("Step 0\n")
//	defer func() {
//		if err := h.service.consumer.Close(); err != nil {
//			log.Fatal(err)
//		}
//	}()
//	fmt.Println("Step 1\n")
//	h.service.consumer.ConsumePartition()
//
//	partitionConsumer, err := h.service.consumer.ConsumePartition(h.service.topic, 1, 1)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//
//	fmt.Println("Step 2\n")
//	defer func() {
//		if err := partitionConsumer.Close(); err != nil {
//			log.Fatal(err)
//		}
//	}()
//
//	fmt.Println("Step 3\n")
//	signals := make(chan os.Signal, 1)
//	signal.Notify(signals, os.Interrupt)
//
//
//	fmt.Println("Step 4\n")
//	ConsumerLoop:
//	for {
//		select {
//		case msg := <-partitionConsumer.Messages():
//			fmt.Println("Step 5\n")
//			go h.processKafkaMessage(*msg)
//		case <- signals:
//			break ConsumerLoop
//		}
//
//	}
//}

func (h *SmartlogicConcordanceTransformerHandler) Run() {
	fmt.Printf("We got here!\n")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	fmt.Printf("Step 1!")
	go func() {
		<-c
		if err := h.service.consumer.Close(); err != nil {
			log.Error("Error closing the consumer: %v", err)
		}
	}()
	fmt.Printf("Step 2!")

	go func() {
		for err := range h.service.consumer.Errors() {
			log.Println(err)
		}
	}()
	fmt.Printf("Step 3!")
	offsets := make(map[string]map[int32]int64)
	for message := range h.service.consumer.Messages() {
		fmt.Printf("Step 4!")
		if offsets[message.Topic] == nil {
			offsets[message.Topic] = make(map[int32]int64)
		}
		go h.processKafkaMessage(*message)

		offsets[message.Topic][message.Partition] = message.Offset
		h.service.consumer.CommitUpto(message)
	}
}

//func (h *SmartlogicConcordanceTransformerHandler) Run() {
//	fmt.Printf("We got here!\n")
//	var list []sarama.ConsumerMessage
//	signals := make(chan os.Signal, 1)
//	signal.Notify(signals, os.Interrupt)
//	fmt.Printf("Step 1!")
//	for {
//		select {
//		case msg, more := <-h.service.consumer.Messages():
//			if msg != nil {
//				fmt.Printf("Adding message to list!\n")
//				list = append(list, *msg)
//			}
//			if more {
//				fmt.Fprintf(os.Stdout, "%s/%d/%d\t%s\t%s\n", msg.Topic, msg.Partition, msg.Offset, msg.Key, msg.Value)
//				h.service.consumer.MarkOffset(msg, "") // mark message as processed
//			}
//		case err, more := <-h.service.consumer.Errors():
//			if more {
//				log.Printf("Error: %s\n", err.Error())
//			}
//		case ntf, more := <-h.service.consumer.Notifications():
//			if more {
//				log.Printf("Rebalanced: %+v\n", ntf)
//				fmt.Printf("Step 4!\n")
//			}
//		case <-signals:
//			fmt.Printf("Step 5!\n")
//			return
//		}
//		if len(list) > 0 {
//			for _, messsage := range list {
//				fmt.Print("Here")
//				h.processKafkaMessage(messsage)
//			}
//		}
//	}
//}

func (h *SmartlogicConcordanceTransformerHandler) processKafkaMessage(msg sarama.ConsumerMessage) {
	fmt.Printf("Message processed %s\n", msg)
	//Extract body and tid from message

	var tid string = ""
	var msgBody string = ""

	h.service.handleConcordanceEvent(msgBody, tid)
}

func (h *SmartlogicConcordanceTransformerHandler) TransformHandler(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	tid := transactionidutils.GetTransactionIDFromRequest(req)

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("Error %v whilst processing json body", err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		rw.Write([]byte("{\"message\":\"Error whilst processing request body.\"}"))
		return
	}

	h.service.handleConcordanceEvent(string(body), tid)
}

func (h *SmartlogicConcordanceTransformerHandler) SendHandler(rw http.ResponseWriter, req *http.Request) {
	tid := transactionidutils.GetTransactionIDFromRequest(req)
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("X-Request-Id", tid)

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("Error %v whilst processing json body", err)
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("{\"message\":\"Error whilst processing request body.\"}"))
		return
	}

	_, _, uppConcordanceJson, err := convertToUppConcordance(string(body))
	if err != nil {
		log.Errorf("Error %v whilst processing json", err)
		rw.WriteHeader(http.StatusUnprocessableEntity)
		rw.Write([]byte("{\"message\":\"Error whilst processing json.\"}"))
		return
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write(uppConcordanceJson)
}

func (h *SmartlogicConcordanceTransformerHandler) gtgCheck(rw http.ResponseWriter, req *http.Request) {
	if err := h.checkKafkaConnectivity(); err != nil {
		log.Errorf("Kafka Healthcheck failed; %v", err.Error())
		rw.WriteHeader(http.StatusServiceUnavailable)
		rw.Write([]byte("Kafka healthcheck failed"))
		return
	}
	if err := h.checkConcordanceRwConnectivity(); err != nil {
		log.Errorf("Concordance Rw Dynamodb Healthcheck failed; %v", err.Error())
		rw.WriteHeader(http.StatusServiceUnavailable)
		rw.Write([]byte("Concordance Rw Dynamodb healthcheck failed"))
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func (h *SmartlogicConcordanceTransformerHandler) RegisterHandlers(router *mux.Router) {
	log.Info("Registering handlers")
	transformAndWrite := handlers.MethodHandler{
		"POST": http.HandlerFunc(h.SendHandler),
	}
	router.Handle("/transformer/send", transformAndWrite)
	transformAndReturn := handlers.MethodHandler{
		"POST": http.HandlerFunc(h.TransformHandler),
	}
	router.Handle("/transform", transformAndReturn)
}

func (h *SmartlogicConcordanceTransformerHandler) RegisterAdminHandlers(router *mux.Router) {
	log.Info("Registering admin handlers")
	var monitoringRouter http.Handler = router
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	var checks []fthealth.Check = []fthealth.Check{h.concordanceRwDynamoDbHealthCheck(), h.kafkaHealthCheck()}
	http.HandleFunc("/__health", fthealth.Handler("ConceptIngester Healthchecks", "Checks for accessing writer", checks...))
	http.HandleFunc("/__gtg", h.gtgCheck)
	http.HandleFunc("/__ping", status.PingHandler)
	http.HandleFunc("/__build-info", status.BuildInfoHandler)
	http.Handle("/", monitoringRouter)
}

func (h *SmartlogicConcordanceTransformerHandler) kafkaHealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Editorial updates of concpet concordances will not be written into UPP",
		Name:             "Check connectivity to Kafka",
		PanicGuide:       "https://dewey.ft.com/smartlogic-concordance-transform.html",
		Severity:         3,
		TechnicalSummary: `Cannot connect to Kafka. If false check that kafka is healthy in this cluster; if so restart service`,
		Checker:          h.checkKafkaConnectivity,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) concordanceRwDynamoDbHealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Editorial updates of concpet concordances will not be written into UPP",
		Name:             "Check connectivity to concordance reader/writer",
		PanicGuide:       "https://dewey.ft.com/smartlogic-concordance-transform.html",
		Severity:         3,
		TechnicalSummary: `Cannot connect to concordance rw. If false, check health of concordance-rw-dynamodb`,
		Checker:          h.checkConcordanceRwConnectivity,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) checkConcordanceRwConnectivity() error {
	urlToCheck := h.service.writerAddress + "__gtg"
	resp, err := http.Get(urlToCheck)
	if err != nil {
		return fmt.Errorf("Error calling writer at %s : %v", urlToCheck, err)
	}
	resp.Body.Close()
	if resp != nil && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Writer %v returned status %d", urlToCheck, resp.StatusCode)
	}
	return nil
}

func (h *SmartlogicConcordanceTransformerHandler) checkKafkaConnectivity() error {
	var err error
	if err != nil {
		return err
	} else {
		return nil
	}
}