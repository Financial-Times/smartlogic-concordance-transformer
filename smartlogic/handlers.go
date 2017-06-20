package smartlogic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/Financial-Times/go-fthealth"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/kafka-client-go/kafka"
	serviceStatus "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/Financial-Times/transactionid-utils-go"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rcrowley/go-metrics"
	"fmt"
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
	return h.transformer.handleConcordanceEvent(msg.Body, msg.Headers["X-Request-Id"])
}

func (h *SmartlogicConcordanceTransformerHandler) TransformHandler(rw http.ResponseWriter, req *http.Request) {
	tid := transactionidutils.GetTransactionIDFromRequest(req)
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("X-Request-Id", tid)

	var smartLogicConcept = SmartlogicConcept{}
	err := json.NewDecoder(req.Body).Decode(&smartLogicConcept)

	if err != nil {
		log.Errorf("Error whilst processing request body: %s", err)
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("{\"message\":\"Error whilst processing request body: " + err.Error() + "\"}"))
		return
	}

	updateStatus, conceptUuid, uppConcordance, err := convertToUppConcordance(smartLogicConcept)
	log.WithFields(log.Fields{"Transaction Id": tid, "Uuid": conceptUuid}).Info("Processing concordance transformation")

	defer req.Body.Close()

	writeResponse(rw, updateStatus, err, uppConcordance)
	//concordedJson, err := json.Marshal(uppConcordance)
	//if err != nil {
	//	log.Errorf("Error whilst marshalling upp concordance model to json: %s", err)
	//	rw.WriteHeader(http.StatusUnprocessableEntity)
	//	rw.Write([]byte("{\"message\":\"Error whilst converting to concorded json: " + err.Error() + "\"}"))
	//	return
	//}
	//rw.WriteHeader(http.StatusOK)
	//rw.Write(concordedJson)
	return
}

func (h *SmartlogicConcordanceTransformerHandler) SendHandler(rw http.ResponseWriter, req *http.Request) {
	tid := transactionidutils.GetTransactionIDFromRequest(req)
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("X-Request-Id", tid)

	var smartLogicConcept = SmartlogicConcept{}
	err := json.NewDecoder(req.Body).Decode(&smartLogicConcept)

	if err != nil {
		fmt.Printf("We got here?\n")
		log.Errorf("Error %v whilst processing json body", err)
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("{\"message\":\"Error whilst processing request body:" + err.Error() + "\"}"))
		return
	}

	updateStatus, conceptUuid, uppConcordance, err := convertToUppConcordance(smartLogicConcept)
	if err != nil {
		writeResponse(rw, updateStatus, err, uppConcordance)
	}

	updateStatus, err = h.transformer.makeRelevantRequest(conceptUuid, uppConcordance, tid)
	if err != nil {
		writeResponse(rw, updateStatus, err, uppConcordance)
		return
	}


	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("Concordance successfully written to db"))

	defer req.Body.Close()
	//if err != nil {
	//	log.Errorf("Error whilst converting to concorded json: %s", err)
	//	rw.WriteHeader(http.StatusUnprocessableEntity)
	//	rw.Write([]byte("{\"message\":\"Error whilst processing request body:" + err.Error() + "\"}"))
	//	return
	//}
	//rw.WriteHeader(http.StatusOK)
	//rw.Write([]byte("Concordance successfully written to db"))
	return
}

func writeResponse(rw http.ResponseWriter, updateStatus status, err error, concordance UppConcordance) {
	enc := json.NewEncoder(rw)
	if err == nil {
		fmt.Printf("Status is %s\n", updateStatus)
		rw.WriteHeader(http.StatusOK)
		bytes, _:= json.Marshal(concordance)
		rw.Write(bytes)
		return
	}
	switch updateStatus {
	case VALID_CONCEPT:
		if err := enc.Encode(concordance); err != nil {
			writeJSONError(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	case SYNTACTICALLY_INCORRECT:
		fmt.Printf("Bad request\n")
		writeJSONError(rw, err.Error(), http.StatusBadRequest)
		return
	case SEMANTICALLY_INCORRECT:
		writeJSONError(rw, err.Error(), http.StatusUnprocessableEntity)
		return
	case DELETED_CONCEPT:
		writeJSONError(rw, err.Error(), http.StatusNoContent)
		return
	default:
		writeJSONError(rw, err.Error(), http.StatusInternalServerError)
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
	http.HandleFunc("/__health", fthealth.Handler("ConceptIngester Healthchecks", "Checks for accessing writer", checks...))
	http.HandleFunc("/__gtg", h.gtgCheck)
	http.HandleFunc("/__ping", serviceStatus.PingHandler)
	http.HandleFunc("/__build-info", serviceStatus.BuildInfoHandler)
	http.Handle("/", monitoringRouter)
}

func (h *SmartlogicConcordanceTransformerHandler) kafkaHealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Editorial updates of concpet concordances will not be written into UPP",
		Name:             "Check connectivity to Kafka",
		PanicGuide:       "https://dewey.ft.com/smartlogic-concordance-transform.html",
		Severity:         3,
		TechnicalSummary: `Check that kafka and zookeeper are healthy in this cluster; if so restart this service`,
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
	err := h.consumer.ConnectivityCheck()
	if err != nil {
		return err
	} else {
		return nil
	}
}
