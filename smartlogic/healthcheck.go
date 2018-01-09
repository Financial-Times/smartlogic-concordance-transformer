package smartlogic

import (
	"github.com/gorilla/mux"
	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	log "github.com/sirupsen/logrus"
	"net/http"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/rcrowley/go-metrics"
	serviceStatus "github.com/Financial-Times/service-status-go/httphandlers"
	"errors"
	"time"
	"github.com/gorilla/handlers"
	"github.com/Financial-Times/service-status-go/gtg"
	"fmt"
)

const (
	deweyURL = "https://dewey.ft.com/smartlogic-concordance-transform.html"
	businessImpact = "Editorial updates of concordance records in smartlogic will not be ingested into UPP"
)

func (h *SmartlogicConcordanceTransformerHandler) RegisterAdminHandlers(router *mux.Router, appSystemCode string, appName string, appDescription string) {
	log.Info("Registering admin handlers")

	var monitoringRouter http.Handler = router
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	var checks = []fthealth.Check{h.concordanceRwDynamoDbHealthCheck(), h.kafkaHealthCheck()}

	timedHC := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode: appSystemCode,
			Description: appDescription,
			Name: appName,
			Checks: checks,
		},
		Timeout: 10 * time.Second,
	}

	router.Path("/__health").Handler(handlers.MethodHandler{"GET": http.HandlerFunc(fthealth.Handler(&timedHC))})
	gtgHandler := serviceStatus.NewGoodToGoHandler(gtg.StatusChecker(h.gtg))
	router.Path("/__gtg").Handler(handlers.MethodHandler{"GET": http.HandlerFunc(gtgHandler)})

	http.HandleFunc("/__build-info", serviceStatus.BuildInfoHandler)
	http.Handle("/", monitoringRouter)
}

func (h *SmartlogicConcordanceTransformerHandler) gtg() gtg.Status {
	kafkaQueueCheck := func() gtg.Status {
		return gtgCheck(h.checkKafkaConnectivity)
	}

	conceptsRwS3Check := func() gtg.Status {
		return gtgCheck(h.checkConcordanceRwConnectivity)
	}

	return gtg.FailFastParallelCheck([]gtg.StatusChecker{
		kafkaQueueCheck,
		conceptsRwS3Check,
	})()
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}

func (h *SmartlogicConcordanceTransformerHandler) kafkaHealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   businessImpact,
		Name:             "Check connectivity to Kafka",
		PanicGuide:       deweyURL,
		Severity:         3,
		TechnicalSummary: `Check that kafka and zookeeper are healthy in this cluster; if so restart this service`,
		Checker:          h.checkKafkaConnectivity,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) concordanceRwDynamoDbHealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   businessImpact,
		Name:             "Check connectivity to concordance reader/writer ",
		PanicGuide:       deweyURL,
		Severity:         3,
		TechnicalSummary: `Check health of concordances-rw-dynamodb`,
		Checker:          h.checkConcordanceRwConnectivity,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) checkConcordanceRwConnectivity() (string, error) {
	urlToCheck := h.transformer.writerAddress + "__gtg"
	request, err := http.NewRequest("GET", urlToCheck, nil)
	if err != nil {
		clientError := fmt.Sprintf("Error creating request to writer %s : %v", urlToCheck, err)
		log.WithError(err).Error(clientError)
		return clientError, errors.New("Unable to verify availibility of concordances-rw-dynamodb")
	}
	resp, err := h.transformer.httpClient.Do(request)
	if err != nil {
		clientError := fmt.Sprintf("Error calling writer at %s : %v", urlToCheck, err)
		log.WithError(err).Error(clientError)
		return clientError, errors.New("Unable to verify availibility of concordances-rw-dynamodb")
	}
	defer resp.Body.Close()
	if resp != nil && resp.StatusCode != http.StatusOK {
		clientError := fmt.Sprintf("Writer %v returned status %d", urlToCheck, resp.StatusCode)
		log.WithError(err).Error(clientError)
		return clientError, errors.New("Unable to verify availibility of concordances-rw-dynamodb")
	}
	return "Successfully connected to Concordances Rw DynamoDB", nil
}

func (h *SmartlogicConcordanceTransformerHandler) checkKafkaConnectivity() (string, error) {
	err := h.consumer.ConnectivityCheck()
	if err != nil {
		clientError := fmt.Sprint("Error verifying open connection to Kafka")
		log.WithError(err).Error(clientError)
		return "Error connecting with Kafka", errors.New(clientError)
	} else {
		return "Successfully connected to Kafka", nil
	}
}
