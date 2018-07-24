package smartlogic

import (
	"encoding/json"
	"net/http"

	"fmt"

	"github.com/Financial-Times/kafka-client-go/kafka"
	"github.com/Financial-Times/transactionid-utils-go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
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
