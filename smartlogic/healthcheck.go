package smartlogic

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/service-status-go/gtg"
	serviceStatus "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

const (
	panicGuideURL  = "https://runbooks.ftops.tech/smartlogic-concordance-transform"
	businessImpact = "Editorial updates of concordance records in smartlogic will not be ingested into UPP"
)

func (h *SmartlogicConcordanceTransformerHandler) RegisterAdminHandlers(router *mux.Router, appSystemCode string, appName string, appDescription string) {
	log.Info("Registering admin handlers")

	var monitoringRouter http.Handler = router
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	var checks = []fthealth.Check{h.concordanceRwNeo4jHealthCheck(), h.kafkaHealthCheck()}

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
	http.HandleFunc(serviceStatus.GTGPath, serviceStatus.NewGoodToGoHandler(gtg.StatusChecker(h.gtg)))
	http.HandleFunc(serviceStatus.BuildInfoPath, serviceStatus.BuildInfoHandler)

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
		PanicGuide:       panicGuideURL,
		Severity:         3,
		TechnicalSummary: `Check that kafka and zookeeper are healthy in this cluster; if so restart this service`,
		Checker:          h.checkKafkaConnectivity,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) concordanceRwNeo4jHealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   businessImpact,
		Name:             "Check connectivity to concordance reader/writer ",
		PanicGuide:       panicGuideURL,
		Severity:         3,
		TechnicalSummary: `Check health of concordances-rw-neo4j`,
		Checker:          h.checkConcordanceRwConnectivity,
	}
}

func (h *SmartlogicConcordanceTransformerHandler) checkConcordanceRwConnectivity() (string, error) {
	urlToCheck := h.transformer.writerAddress + "__gtg"
	request, err := http.NewRequest("GET", urlToCheck, nil)
	if err != nil {
		clientError := fmt.Sprintf("Error creating request to writer %s : %v", urlToCheck, err)
		log.WithError(err).Error(clientError)
		return clientError, errors.New("Unable to verify availibility of concordances-rw-neo4j")
	}
	resp, err := h.transformer.httpClient.Do(request)
	if err != nil {
		clientError := fmt.Sprintf("Error calling writer at %s : %v", urlToCheck, err)
		log.WithError(err).Error(clientError)
		return clientError, errors.New("Unable to verify availibility of concordances-rw-neo4j")
	}
	defer resp.Body.Close()
	if resp != nil && resp.StatusCode != http.StatusOK {
		clientError := fmt.Sprintf("Writer %v returned status %d", urlToCheck, resp.StatusCode)
		log.WithError(err).Error(clientError)
		return clientError, errors.New("Unable to verify availibility of concordances-rw-neo4j")
	}
	return "Successfully connected to Concordances Rw Neo4j", nil
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
