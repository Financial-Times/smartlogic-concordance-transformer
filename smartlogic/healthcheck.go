package smartlogic

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/kafka-client-go/v3"
	"github.com/Financial-Times/service-status-go/gtg"
	serviceStatus "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
	"github.com/rcrowley/go-metrics"
)

const (
	panicGuideURL  = "https://runbooks.ftops.tech/smartlogic-concordance-transform"
	businessImpact = "Editorial updates of concordance records in smartlogic will not be ingested into UPP"
)

func (h *ConcordanceTransformerHandler) RegisterAdminHandlers(router *mux.Router, appSystemCode string, appName string, appDescription string) {
	h.log.Info("Registering admin handlers")

	var monitoringRouter http.Handler = router
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(h.log.Logger, monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	var checks = []fthealth.Check{h.concordanceRwNeo4jHealthCheck(), h.kafkaHealthCheck(), h.kafkaMonitorCheck()}

	timedHC := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  appSystemCode,
			Description: appDescription,
			Name:        appName,
			Checks:      checks,
		},
		Timeout: 10 * time.Second,
	}

	http.HandleFunc("/__health", fthealth.Handler(&timedHC))
	http.HandleFunc(serviceStatus.GTGPath, serviceStatus.NewGoodToGoHandler(h.gtg))
	http.HandleFunc(serviceStatus.BuildInfoPath, serviceStatus.BuildInfoHandler)

	http.Handle("/", monitoringRouter)
}

func (h *ConcordanceTransformerHandler) gtg() gtg.Status {
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

func (h *ConcordanceTransformerHandler) kafkaHealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   businessImpact,
		Name:             "Check connectivity to Kafka",
		PanicGuide:       panicGuideURL,
		Severity:         3,
		TechnicalSummary: `Check that kafka and zookeeper are healthy in this cluster; if so restart this service`,
		Checker:          h.checkKafkaConnectivity,
	}
}

func (h *ConcordanceTransformerHandler) kafkaMonitorCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Consumer is lagging behind when reading messages. Editorial updates of concordance records in SmartLogic will be delayed.",
		Name:             "Check consumer status",
		Severity:         3,
		TechnicalSummary: kafka.LagTechnicalSummary,
		Checker:          h.monitorKafkaConnectivity,
	}
}

func (h *ConcordanceTransformerHandler) concordanceRwNeo4jHealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   businessImpact,
		Name:             "Check connectivity to concordance reader/writer ",
		PanicGuide:       panicGuideURL,
		Severity:         3,
		TechnicalSummary: `Check health of concordances-rw-neo4j`,
		Checker:          h.checkConcordanceRwConnectivity,
	}
}

func (h *ConcordanceTransformerHandler) checkConcordanceRwConnectivity() (string, error) {
	urlToCheck := h.transformer.writerAddress + "__gtg"
	request, err := http.NewRequest("GET", urlToCheck, nil)
	if err != nil {
		clientError := fmt.Sprintf("Error creating request to writer %s : %v", urlToCheck, err)
		h.log.WithError(err).Error(clientError)
		return clientError, errors.New("unable to verify availibility of concordances-rw-neo4j")
	}
	resp, err := h.transformer.httpClient.Do(request)
	if err != nil {
		clientError := fmt.Sprintf("Error calling writer at %s : %v", urlToCheck, err)
		h.log.WithError(err).Error(clientError)
		return clientError, errors.New("unable to verify availibility of concordances-rw-neo4j")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			h.log.WithError(err).Info("Could not close body")
		}
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		clientError := fmt.Sprintf("Writer %v returned status %d", urlToCheck, resp.StatusCode)
		h.log.WithError(err).Error(clientError)
		return clientError, errors.New("unable to verify availibility of concordances-rw-neo4j")
	}
	return "Successfully connected to Concordances Rw Neo4j", nil
}

func (h *ConcordanceTransformerHandler) checkKafkaConnectivity() (string, error) {
	err := h.consumer.ConnectivityCheck()
	if err != nil {
		h.log.WithError(err).Error("error verifying open connection to Kafka")
		return "Error connecting with Kafka", err
	}
	return "Successfully connected to Kafka", nil
}

func (h *ConcordanceTransformerHandler) monitorKafkaConnectivity() (string, error) {
	if err := h.consumer.MonitorCheck(); err != nil {
		h.log.WithError(err).Error("kafka consumer is lagging")
		return "Error communicating with Kafka", err
	}
	return "Kafka connection is ok", nil
}
