package smartlogic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"fmt"
	"github.com/Financial-Times/go-fthealth"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/kafka-client-go/kafka"
	serviceStatus "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rcrowley/go-metrics"
)

type SmartlogicConcordanceTransformerHandler struct {
	transformer TransformerService
	consumer    kafka.Consumer
}

func NewHandler(transformer TransformerService, consumer kafka.Consumer) SmartlogicConcordanceTransformerHandler {
	return SmartlogicConcordanceTransformerHandler{
		transformer: transformer,
		consumer:    consumer,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) ProcessKafkaMessage(msg kafka.FTMessage) error {
	var tid string
	if msg.Headers["X-Request-Id"] == "" {
		tid = transactionidutils.NewTransactionID()
	} else {
		tid = msg.Headers["X-Request-Id"]
	}
	return h.transformer.handleConcordanceEvent(msg.Body, tid)
}

func (h *SmartlogicConcordanceTransformerHandler) TransformHandler(rw http.ResponseWriter, req *http.Request) {
	tid := transactionidutils.GetTransactionIDFromRequest(req)
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("X-Request-Id", tid)

	var smartLogicConcept = SmartlogicConcept{}
	err := json.NewDecoder(req.Body).Decode(&smartLogicConcept)

	if err != nil {
		log.WithError(err).WithField("transaction_id", tid).Error("Error whilst processing request body")
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("{\"message\":\"Error whilst processing request body: " + err.Error() + "\"}"))
		return
	}

	log.WithField("transaction_id", tid).Debug("Processing concordance transformation")
	updateStatus, conceptUuid, uppConcordance, err := convertToUppConcordance(smartLogicConcept, tid)

	if err != nil {
		writeResponse(rw, updateStatus, err)
		return
	}
	defer req.Body.Close()

	json.NewEncoder(rw).Encode(uppConcordance)
	log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid, "status": http.StatusOK}).Info("Smartlogic payload successfully transformed")
	return
}

func (h *SmartlogicConcordanceTransformerHandler) SendHandler(rw http.ResponseWriter, req *http.Request) {
	tid := transactionidutils.GetTransactionIDFromRequest(req)
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("X-Request-Id", tid)

	var smartLogicConcept = SmartlogicConcept{}
	err := json.NewDecoder(req.Body).Decode(&smartLogicConcept)

	if err != nil {
		log.WithError(err).WithField("transaction_id", tid).Error("Error whilst processing request body")
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("{\"message\":\"Error whilst processing request body:" + err.Error() + "\"}"))
		return
	}

	log.WithField("transaction_id", tid).Debug("Processing concordance transformation")
	updateStatus, conceptUuid, uppConcordance, err := convertToUppConcordance(smartLogicConcept, tid)

	if err != nil {
		writeResponse(rw, updateStatus, err)
	}

	updateStatus, err = h.transformer.makeRelevantRequest(conceptUuid, uppConcordance, tid)

	if err != nil {
		writeResponse(rw, updateStatus, err)
		return
	}
	defer req.Body.Close()

	var logMsg string
	if updateStatus == VALID_CONCEPT {
		logMsg = "Concordance record forwarded to writer"
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("{\"message\":\"" + logMsg + "\"}"))
		return
	} else if updateStatus == NO_CONTENT {
		logMsg = "Concordance record successfuly deleted"
		rw.Write([]byte("{\"message\":\"" + logMsg + "\"}"))
	} else if updateStatus == NOT_FOUND {
		logMsg = "Concordance record not found"
		rw.Write([]byte("{\"message\":\"" + logMsg + "\"}"))
	}
	log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid, "status": http.StatusOK}).Info(logMsg)

	return
}

func writeResponse(rw http.ResponseWriter, updateStatus status, err error) {
	switch updateStatus {
	case SYNTACTICALLY_INCORRECT:
		writeJSONError(rw, err.Error(), http.StatusBadRequest)
		return
	case SEMANTICALLY_INCORRECT:
		writeJSONError(rw, err.Error(), http.StatusUnprocessableEntity)
		return
	case SERVICE_UNAVAILABLE:
		writeJSONError(rw, err.Error(), http.StatusServiceUnavailable)
		return
	case INTERNAL_ERROR:
		writeJSONError(rw, err.Error(), http.StatusInternalServerError)
		return
	default:
		writeJSONError(rw, "Unknown error", http.StatusInternalServerError)
		return
	}
}

func writeJSONError(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.WriteHeader(statusCode)
	fmt.Fprintln(w, fmt.Sprintf("{\"message\": \"%s\"}", errorMsg))
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
	http.HandleFunc("/__health", fthealth.Handler("Smartlogic Concordance Transformer Healthchecks", "Checks for accessing writer and kafka", checks...))
	http.HandleFunc("/__gtg", h.gtgCheck)
	http.HandleFunc("/__build-info", serviceStatus.BuildInfoHandler)
	http.Handle("/", monitoringRouter)
}

func (h *SmartlogicConcordanceTransformerHandler) kafkaHealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Editorial updates of concept concordances will not be written into UPP",
		Name:             "Check connectivity to Kafka",
		PanicGuide:       "https://dewey.ft.com/smartlogic-concordance-transform.html",
		Severity:         3,
		TechnicalSummary: `Check that kafka and zookeeper are healthy in this cluster; if so restart this service`,
		Checker:          h.checkKafkaConnectivity,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) concordanceRwDynamoDbHealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Editorial updates of concept concordances will not be written into UPP",
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
	err := h.consumer.ConnectivityCheck()
	if err != nil {
		return err
	} else {
		return nil
	}
}
