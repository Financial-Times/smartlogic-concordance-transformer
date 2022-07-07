package smartlogic

import (
	"encoding/json"
	"net/http"

	"fmt"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v3"
	transactionidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type messageConsumer interface {
	ConnectivityCheck() error
	MonitorCheck() error
}

type ConcordanceTransformerHandler struct {
	transformer TransformerService
	consumer    messageConsumer
	log         *logger.UPPLogger
}

func NewHandler(transformer TransformerService, consumer messageConsumer, log *logger.UPPLogger) ConcordanceTransformerHandler {
	return ConcordanceTransformerHandler{
		transformer: transformer,
		consumer:    consumer,
		log:         log,
	}
}

func (h *ConcordanceTransformerHandler) ProcessKafkaMessage(msg kafka.FTMessage) {
	var tid string
	if msg.Headers["X-Request-Id"] == "" {
		tid = transactionidutils.NewTransactionID()
	} else {
		tid = msg.Headers["X-Request-Id"]
	}

	_ = h.transformer.handleConcordanceEvent(msg.Body, tid)
}

func (h *ConcordanceTransformerHandler) RegisterHandlers(router *mux.Router) {
	h.log.Info("Registering handlers")
	transformAndWrite := handlers.MethodHandler{
		"POST": http.HandlerFunc(h.SendHandler),
	}
	router.Handle("/transform/send", transformAndWrite)
	transformAndReturn := handlers.MethodHandler{
		"POST": http.HandlerFunc(h.TransformHandler),
	}
	router.Handle("/transform", transformAndReturn)
}

func (h *ConcordanceTransformerHandler) TransformHandler(rw http.ResponseWriter, req *http.Request) {
	tid := transactionidutils.GetTransactionIDFromRequest(req)
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("X-Request-Id", tid)

	var smartLogicConcept = ConceptData{}
	err := json.NewDecoder(req.Body).Decode(&smartLogicConcept)

	if err != nil {
		h.log.WithError(err).WithField("transaction_id", tid).Error("Error whilst processing request body")
		rw.WriteHeader(http.StatusBadRequest)
		_, err := rw.Write([]byte("{\"message\":\"Error whilst processing request body: " + err.Error() + "\"}"))
		if err != nil {
			h.log.WithError(err).Info("Failed to send response from TransformHandler")
		}
		return
	}

	h.log.WithField("transaction_id", tid).Debug("Processing concordance transformation")
	updateStatus, conceptUUID, uppConcordance, err := convertToUppConcordance(smartLogicConcept, tid, h.log)

	if err != nil {
		writeResponse(rw, updateStatus, err)
		return
	}

	if err := json.NewEncoder(rw).Encode(uppConcordance); err != nil {
		h.log.WithError(err).Error("Could not encode transformed concordance response")
		return
	}
	h.log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID, "status": http.StatusOK}).Info("Smartlogic payload successfully transformed")
}

func (h *ConcordanceTransformerHandler) SendHandler(rw http.ResponseWriter, req *http.Request) {
	tid := transactionidutils.GetTransactionIDFromRequest(req)
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("X-Request-Id", tid)

	var smartLogicConcept = ConceptData{}
	err := json.NewDecoder(req.Body).Decode(&smartLogicConcept)

	if err != nil {
		h.log.WithError(err).WithField("transaction_id", tid).Error("Error whilst processing request body")
		rw.WriteHeader(http.StatusBadRequest)
		_, err := rw.Write([]byte("{\"message\":\"Error whilst processing request body:" + err.Error() + "\"}"))
		if err != nil {
			h.log.WithError(err).Errorf("Failed to send response from SendHandler")
		}
		return
	}

	h.log.WithField("transaction_id", tid).Debug("Processing concordance transformation")
	updateStatus, conceptUUID, uppConcordance, err := convertToUppConcordance(smartLogicConcept, tid, h.log)

	if err != nil {
		writeResponse(rw, updateStatus, err)
	}

	updateStatus, err = h.transformer.makeRelevantRequest(conceptUUID, uppConcordance, tid)

	if err != nil {
		writeResponse(rw, updateStatus, err)
		return
	}

	var message string
	switch updateStatus {
	case ValidConcept:
		message = "Concordance record forwarded to writer"
	case NoContent:
		message = "Concordance record successfully deleted"
	case NotFound:
		message = "Concordance record not found"
	}

	if _, err = rw.Write([]byte("{\"message\":\"" + message + "\"}")); err != nil {
		h.log.
			WithError(err).
			WithField("response_message", message).
			Info("Failed to send response")
		return
	}

	h.log.
		WithTransactionID(tid).
		WithUUID(conceptUUID).
		WithField("status", http.StatusOK).
		Info(message)
}

func writeResponse(rw http.ResponseWriter, updateStatus status, err error) {
	switch updateStatus {
	case SyntacticallyIncorrect:
		writeJSONError(rw, err.Error(), http.StatusBadRequest)
		return
	case SemanticallyIncorrect:
		writeJSONError(rw, err.Error(), http.StatusUnprocessableEntity)
		return
	case ServiceUnavailable:
		writeJSONError(rw, err.Error(), http.StatusServiceUnavailable)
		return
	case InternalError:
		writeJSONError(rw, err.Error(), http.StatusInternalServerError)
		return
	default:
		writeJSONError(rw, "Unknown error", http.StatusInternalServerError)
		return
	}
}

func writeJSONError(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(fmt.Sprintf("{\"message\": \"%s\"}", errorMsg)))
}
