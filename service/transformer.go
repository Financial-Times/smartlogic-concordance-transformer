package service

import (
	"github.com/golang/go/src/pkg/fmt"
	"github.com/Shopify/sarama"
	"github.com/Financial-Times/smartlogic-concordance-transformer/utils"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"github.com/coreos/fleet/log"
	"strings"
)

const (
	writerRoute = "__concordance-rw-dynamodb/concordance/"
)

type TransformerService struct {
	consumer sarama.Consumer
	topic	string
	vulcanAddress string
	httpClient 	httpClient
}

type httpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

func NewTransformerService(consumer sarama.Consumer, topic string, vulcanAddress string, httpClient httpClient) TransformerService {
	return TransformerService{
		consumer:	consumer,
		topic: 		topic,
		vulcanAddress: vulcanAddress,
		httpClient:    httpClient,
	}
}

func (ts *TransformerService) handleConcordanceEvent(msgBody string, tid string) {
	conceptUuid, concordanceFound, uppConcordanceJson, err := convertToUppConcordance(msgBody)
	if err != nil {
		log.Errorf("Request resulted in error: %v\n", err)
		return
	}
	ts.makeRelevantRequest(conceptUuid, concordanceFound, uppConcordanceJson, tid)
}

func convertToUppConcordance(msgBody string) (string, bool, []byte, error) {
	smartLogicConcept := utils.SmartlogicConcept{}
	bodyAsBytes := []byte(msgBody)
	if err := json.Unmarshal(bodyAsBytes, &smartLogicConcept); err != nil {
		return "", false, nil, err
	}

	conceptUuid := smartLogicConcept.Concepts[0].Ids[0].Value
	if conceptUuid == "" {
		return "", false, nil, errors.New("Payload: " + msgBody + "; is missing concept uuid")
	}

	var concordanceIds []string
	for _, id := range smartLogicConcept.Concepts[0].Identifiers {
		concordanceIds = append(concordanceIds, id.Value)
	}

	if len(concordanceIds) > 0 {
		uppConcordance := utils.UppConcordance{}
		uppConcordance.ConceptUuid = conceptUuid
		uppConcordance.ConcordedIds = concordanceIds

		concordedJson, err := json.Marshal(uppConcordance)
		if err != nil {
			return "", false, nil, err
		}
		return conceptUuid, true, concordedJson, nil
	}
	return conceptUuid, false, nil, nil
}

func (ts *TransformerService) makeRelevantRequest(uuid string, concordanceFound bool, uppConcordanceJson []byte, tid string) {
	var err error
	if concordanceFound {
		log.Infof("Concordance found for %s; forwarding request to writer", uuid)
		err = ts.makeWriteRequest(uuid, uppConcordanceJson, tid)
	} else {
		log.Infof("No Concordance found for %s; making delete request", uuid)
		err = ts.makeDeleteRequest(uuid, tid)
	}

	if err != nil {
		log.Errorf("Write request resulted in error: %v\n", err)
		return
	}
}

func (ts *TransformerService) makeWriteRequest(uuid string, concordedJson []byte, tid string) error {
	reqURL := ts.vulcanAddress + writerRoute + uuid
	request, err := http.NewRequest("PUT", reqURL, strings.NewReader(string(concordedJson)))
	if err != nil {
		return fmt.Errorf("Failed to create request to %s with body %s", reqURL, concordedJson)
	}
	request.ContentLength = -1
	request.Header.Set("X-Request-Id", tid)

	resp, reqErr := ts.httpClient.Do(request)

	if reqErr != nil || resp.StatusCode != 200 {
		return fmt.Errorf("Request to %s returned status: %d", reqURL, strconv.Itoa(resp.StatusCode))
	}

	defer resp.Body.Close()
	return nil
}

func (ts *TransformerService) makeDeleteRequest(uuid string, tid string) error {
	reqURL := ts.vulcanAddress + writerRoute + uuid
	request, err := http.NewRequest("DELETE", reqURL, nil)
	if err != nil {
		return fmt.Errorf("Failed to create request to %s with body %s", reqURL, nil)
	}
	request.ContentLength = -1
	request.Header.Set("X-Request-Id", tid)

	resp, reqErr := ts.httpClient.Do(request)

	if reqErr != nil || resp.StatusCode != 204 || resp.StatusCode != 404 {
		return errors.New("Request to " + reqURL + " returned status: " + strconv.Itoa(resp.StatusCode) + "; skipping " + uuid)
	}
	defer resp.Body.Close()

	return nil
}