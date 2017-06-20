package smartlogic

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/pborman/uuid"
	"bytes"
	log "github.com/Sirupsen/logrus"
	"fmt"
)

var uuidMatcher = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
type status int

const (
	THING_URI_PREFIX = "http://www.ft.com/thing/"
	DELETED_CONCEPT status = iota
	SYNTACTICALLY_INCORRECT
	SEMANTICALLY_INCORRECT
	VALID_CONCEPT
	INTERNAL_ERROR
)

type TransformerService struct {
	topic	string
	writerAddress string
	httpClient 	httpClient
}

type httpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

func NewTransformerService(topic string, writerAddress string, httpClient httpClient) TransformerService {
	return TransformerService{
		topic: 		topic,
		writerAddress: writerAddress,
		httpClient:    httpClient,
	}
}

func (ts *TransformerService) handleConcordanceEvent(msgBody string, tid string) error {
	var smartLogicConcept = SmartlogicConcept{}
	decoder := json.NewDecoder(bytes.NewBufferString(msgBody))
	err := decoder.Decode(&smartLogicConcept)
	if err != nil {
		log.WithError(err).Error("Failed to decode Kafka payload")
		return errors.New("")
	}
	_, conceptUuid, uppConcordance, err := convertToUppConcordance(smartLogicConcept)
	if err != nil {
		return err
	}
	_, err = ts.makeRelevantRequest(conceptUuid, uppConcordance, tid)
	if err != nil {
		return err
	}
	return nil
}

func convertToUppConcordance(smartlogicConcepts SmartlogicConcept) (status, string, UppConcordance, error) {
	if len(smartlogicConcepts.Concepts) == 0 {
		return SEMANTICALLY_INCORRECT, "", UppConcordance{}, logAndReturnTheError("Bad Request: Requst json: %s is missing @graph field", smartlogicConcepts)
	}
	if len(smartlogicConcepts.Concepts) > 1 {
		return SEMANTICALLY_INCORRECT, "", UppConcordance{}, logAndReturnTheError("Bad Request: More than 1 concept in smartlogic concept payload which is currently not supported", "")
	}

	smartlogicConcept := smartlogicConcepts.Concepts[0]

	conceptUuid := extractUuid(smartlogicConcept.Id)
	if conceptUuid == "" {
		return SYNTACTICALLY_INCORRECT, conceptUuid, UppConcordance{}, logAndReturnTheError("Bad Request: Requst json: %s has missing/invalid @id field", smartlogicConcepts)
	}

	concordanceIds := make([]string, 0)
	for _, id := range smartlogicConcept.TmeIdentifiers {
		uuidFromTmeId, err := validateIdAndConvertToUuid(id.Value)
		if err != nil {
			return SYNTACTICALLY_INCORRECT, conceptUuid, UppConcordance{}, logAndReturnTheError("Bad Request: Concordance id %s is not a valid TME Id", id.Value)
		}
		if len(concordanceIds) > 0 {
			for _, concordedId := range concordanceIds {
				if concordedId == uuidFromTmeId {
					return SYNTACTICALLY_INCORRECT, conceptUuid, UppConcordance{}, logAndReturnTheError("Payload from smartlogic: %s contains duplicate TME id values", smartlogicConcept)
				}
			}
			concordanceIds = append(concordanceIds, uuidFromTmeId)
		} else {
			concordanceIds = append(concordanceIds, uuidFromTmeId)
		}
	}
	uppConcordance := UppConcordance{}
	uppConcordance.ConceptUuid = conceptUuid
	uppConcordance.ConcordedIds = concordanceIds
	return VALID_CONCEPT, conceptUuid, uppConcordance, nil
}

func validateIdAndConvertToUuid(tmeId string) (string, error) {
	subStrings := strings.Split(tmeId, "-")
	if len(subStrings) != 2 || validateSubstrings(subStrings) == true {
		return "", errors.New(tmeId + " is not a valid TME Id")
	} else {
		return uuid.NewMD5(uuid.UUID{}, []byte(tmeId)).String(), nil
	}
}

func validateSubstrings(subStrings []string) bool {
	subStringIsEmpty := false
	for _, string := range subStrings {
		if string == "" {
			subStringIsEmpty = true
		}
	}
	return subStringIsEmpty
}

func (ts *TransformerService) makeRelevantRequest(uuid string, uppConcordance UppConcordance, tid string) (status, error) {
	var err error
	var reqStatus status
	if len(uppConcordance.ConcordedIds) > 0 {
		log.WithFields(log.Fields{"Transaction Id": tid, "uuid": uuid}).Info("Concordance found; forwarding request to writer")
		reqStatus, err = ts.makeWriteRequest(uuid, uppConcordance, tid)
	} else {
		log.WithFields(log.Fields{"Transaction Id": tid, "uuid": uuid}).Info("No concordance found; making delete request")
		reqStatus, err = ts.makeDeleteRequest(uuid, tid)
	}
	return reqStatus, err
}

func (ts *TransformerService) makeWriteRequest(uuid string, uppConcordance UppConcordance, tid string) (status, error) {
	reqURL := ts.writerAddress + "concordance/" + uuid
	concordedJson, err := json.Marshal(uppConcordance)
	if err != nil {
		return SYNTACTICALLY_INCORRECT, logAndReturnTheError("Error whilst marshalling upp concordance model to json: %s", err)
	}
	request, err := http.NewRequest("PUT", reqURL, strings.NewReader(string(concordedJson)))
	if err != nil {
		return INTERNAL_ERROR, logAndReturnTheError("Failed to create GET request to " + reqURL + " with body " + string(concordedJson), "")
	}
	request.ContentLength = -1
	request.Header.Set("X-Request-Id", tid)

	resp, reqErr := ts.httpClient.Do(request)
	if reqErr != nil {
		return INTERNAL_ERROR, logAndReturnTheError("Get request resulted in error: %v", reqErr)
	} else if resp.StatusCode != 200 {
		return status(resp.StatusCode), logAndReturnTheError("Get request returned status: %d", strconv.Itoa(resp.StatusCode))
	}

	defer resp.Body.Close()
	return status(resp.StatusCode), nil
}

func (ts *TransformerService) makeDeleteRequest(uuid string, tid string) (status, error) {
	reqURL := ts.writerAddress + "concordances/" + uuid
	request, err := http.NewRequest("DELETE", reqURL, nil)
	if err != nil {
		return INTERNAL_ERROR, logAndReturnTheError("Failed to create Delete request to %s", reqURL)
	}
	request.ContentLength = -1
	request.Header.Set("X-Request-Id", tid)

	resp, reqErr := ts.httpClient.Do(request)

	if reqErr != nil {
		return INTERNAL_ERROR, logAndReturnTheError("Delete request resulted in error: %v", reqErr)
	} else if resp.StatusCode != 204 && resp.StatusCode != 404 {
		return status(resp.StatusCode), logAndReturnTheError("Delete request returned status: %d", strconv.Itoa(resp.StatusCode))
	}
	defer resp.Body.Close()
	return status(resp.StatusCode), nil
}

func extractUuid(url string) string {
	if strings.HasPrefix(url, THING_URI_PREFIX) {
		extractedUuid := strings.TrimPrefix(url, THING_URI_PREFIX)
		if uuidMatcher.MatchString(extractedUuid) != true {
			return ""
		}
		return extractedUuid
	}
	return ""
}

func logAndReturnTheError(message string, variable interface{}) error {
	if variable == "" {
		simpleErr := errors.New(message)
		log.WithError(simpleErr)
		return simpleErr
	} else {
		err := fmt.Errorf(message, variable)
		log.WithError(err)
		return err
	}
}