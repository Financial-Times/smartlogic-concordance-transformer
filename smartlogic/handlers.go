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
	"io/ioutil"
	"github.com/Financial-Times/transactionid-utils-go"
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"sync"
	"os"
	"os/signal"
	"syscall"
)

type SmartlogicConcordanceTransformerHandler struct {
	transformer TransformerService
	queue  QueueService

}

func NewHandler(transformer TransformerService, queue QueueService) SmartlogicConcordanceTransformerHandler {
	return SmartlogicConcordanceTransformerHandler{
		transformer: transformer,
		queue: queue,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) SubscribeToQueue(client http.Client) {
	consumer := queueConsumer.NewConsumer(h.queue.consumerConfig, h.processKafkaMessage, &client)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		consumer.Start()
		wg.Done()
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	<-ch
	log.Println("Shutting down application...")

	consumer.Stop()
	wg.Wait()

	log.Println("Application closing")
}

func (h *SmartlogicConcordanceTransformerHandler) processKafkaMessage(msg queueConsumer.Message) {
	fmt.Printf("Message processed %s\n", msg)
	//Extract body and tid from message

	var tid string = ""
	var msgBody string = ""

	h.transformer.handleConcordanceEvent(msgBody, tid)
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

	h.transformer.handleConcordanceEvent(string(body), tid)
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
	urlToCheck := h.transformer.writerAddress + "__gtg"
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