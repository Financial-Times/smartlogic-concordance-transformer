package smartlogic

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	log "github.com/Sirupsen/logrus"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/rcrowley/go-metrics"
	"github.com/Financial-Times/go-fthealth"
	"net/http"
	"io/ioutil"
	"github.com/Financial-Times/transactionid-utils-go"
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/pkg/errors"
	"strconv"
	"encoding/json"
)

type SmartlogicConcordanceTransformerHandler struct {
	transformer TransformerService
	Consumer queueConsumer.MessageConsumer
}

func NewHandler(transformer TransformerService) SmartlogicConcordanceTransformerHandler {
	return SmartlogicConcordanceTransformerHandler{
		transformer: transformer,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) ProcessKafkaMessage(msg queueConsumer.Message) {
	h.transformer.handleConcordanceEvent(msg.Body, msg.Headers["X-Request-Id"])
}

func (h *SmartlogicConcordanceTransformerHandler) TransformHandler(rw http.ResponseWriter, req *http.Request) {
	tid := transactionidutils.GetTransactionIDFromRequest(req)
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("X-Request-Id", tid)

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("Error whilst processing request body: %s", err)
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("{\"message\":\"Error whilst processing request body: " + err.Error() + "\"}"))
		return
	}

	conceptUuid, uppConcordance, err := convertToUppConcordance(string(body))
	log.Infof("Processing concordance transformation for concept with uuid: %s and trans id: %s", conceptUuid, tid)
	if err != nil {
		log.Errorf("Error whilst mapping to upp concordance model: %s", err)
		rw.WriteHeader(http.StatusUnprocessableEntity)
		rw.Write([]byte("{\"message\":\"Error whilst converting to concorded json: " + err.Error() + "\"}"))
		return
	}
	concordedJson, err := json.Marshal(uppConcordance)
	if err != nil {
		log.Errorf("Error whilst marshalling upp concordance model to json: %s", err)
		rw.WriteHeader(http.StatusUnprocessableEntity)
		rw.Write([]byte("{\"message\":\"Error whilst converting to concorded json: " + err.Error() + "\"}"))
		return
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write(concordedJson)
	return
}

func (h *SmartlogicConcordanceTransformerHandler) SendHandler(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	tid := transactionidutils.GetTransactionIDFromRequest(req)

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("Error %v whilst processing json body", err)
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("{\"message\":\"Error whilst processing request body:" + err.Error() + "\"}"))
		return
	}

	err = h.transformer.handleConcordanceEvent(string(body), tid)
	if err != nil {
		log.Errorf("Error whilst converting to concorded json: %s", err)
		rw.WriteHeader(http.StatusUnprocessableEntity)
		rw.Write([]byte("{\"message\":\"Error whilst processing request body:" + err.Error() + "\"}"))
		return
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("Concordance successfully written to db"))
}

func (h *SmartlogicConcordanceTransformerHandler) gtgCheck(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
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
	router.Handle("/transform/send", transformAndWrite)
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
		TechnicalSummary: `Check that kafka, zookeeper, kafka-proxy are healthy in this cluster; if so restart this service`,
		Checker:          h.checkKafkaConnectivity,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) concordanceRwDynamoDbHealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Editorial updates of concpet concordances will not be written into UPP",
		Name:             "Check connectivity to concordance reader/writer",
		PanicGuide:       "https://dewey.ft.com/smartlogic-concordance-transform.html",
		Severity:         3,
		TechnicalSummary: `Check health of concordance-rw-dynamodb`,
		Checker:          h.checkConcordanceRwConnectivity,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) checkConcordanceRwConnectivity() error {
	urlToCheck := h.transformer.writerAddress + "__gtg"
	request, err := http.NewRequest("GET", urlToCheck, nil)
	if err != nil {
		return errors.New("Failed to create request: " + urlToCheck)
	}
	resp, err := h.transformer.httpClient.Do(request)
	if err != nil {
		return errors.New("Error " + err.Error() + " calling writer at " + urlToCheck)
	}
	resp.Body.Close()
	if resp != nil && resp.StatusCode != http.StatusOK {
		return errors.New("Writer " + urlToCheck + " returned status " + strconv.Itoa(resp.StatusCode))
	}
	return nil
}

func (h *SmartlogicConcordanceTransformerHandler) checkKafkaConnectivity() error {
	_, err := h.Consumer.ConnectivityCheck()
	if err != nil {
		return err
	} else {
		return nil
	}
}