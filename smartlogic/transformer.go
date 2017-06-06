package smartlogic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"github.com/coreos/fleet/log"
	"strings"
	"regexp"
	"github.com/pborman/uuid"
)

var uuidMatcher = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

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
	conceptUuid, concordanceFound, uppConcordanceJson, err := convertToUppConcordance(msgBody)
	if err != nil {
		return errors.New("Conversion of payload to upp concordance resulted in error: " + err.Error())
	}
	err = ts.makeRelevantRequest(conceptUuid, concordanceFound, uppConcordanceJson, tid)
	if err != nil {
		return errors.New("Request to concordance rw resulted in error: " + err.Error())
	}
	return nil
}

func convertToUppConcordance(msgBody string) (string, bool, []byte, error) {
	concordanceFound := false
	if !strings.Contains(msgBody, "@graph") || !strings.Contains(msgBody, "@graph") {
		return "", concordanceFound, nil, errors.New("Input: " + msgBody + " is missing @graph and/or @id fields")
	}
	smartLogicConcept := SmartlogicConcept{}
	bodyAsBytes := []byte(msgBody)
	if err := json.Unmarshal(bodyAsBytes, &smartLogicConcept); err != nil {
		return "", concordanceFound, nil, err
	}

	conceptUuid := extractUuid(smartLogicConcept.Concepts[0].Id)
	if conceptUuid == "" {
		return "", concordanceFound, nil, errors.New("Json payload Id field " + smartLogicConcept.Concepts[0].Id + " has invalid/missing url")
	}

	concordanceIds := make([]string, 0)
	for _, id := range smartLogicConcept.Concepts[0].TmeIdentifiers {
		uuidFromTmeId, err := validateIdAndConvertToUuid(id.Value)
		if err != nil {
			return "", concordanceFound, nil, err
		}
		if len(concordanceIds) > 0 {
			alreadyExists := false
			for _, concordedId := range concordanceIds {
				if concordedId == uuidFromTmeId {
					alreadyExists = true
				}
			}
			if alreadyExists != true {
				concordanceIds = append(concordanceIds, uuidFromTmeId)
			}
		} else {
			concordanceIds = append(concordanceIds, uuidFromTmeId)
		}
	}
	if len(concordanceIds) > 0 {
		concordanceFound = true
	}
	uppConcordance := UppConcordance{}
	uppConcordance.ConceptUuid = conceptUuid
	uppConcordance.ConcordedIds = concordanceIds

	concordedJson, err := json.Marshal(uppConcordance)
	if err != nil {
		return "", concordanceFound, nil, err
	}
	return conceptUuid, concordanceFound, concordedJson, nil
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

func (ts *TransformerService) makeRelevantRequest(uuid string, concordanceFound bool, uppConcordanceJson []byte, tid string) error {
	var err error
	if concordanceFound {
		log.Infof("Transaction Id: " + tid + ". Concordance found for %s; forwarding request to writer", uuid)
		err = ts.makeWriteRequest(uuid, uppConcordanceJson, tid)
	} else {
		log.Infof("Transaction Id: " + tid + ". No Concordance found for %s; making delete request", uuid)
		err = ts.makeDeleteRequest(uuid, tid)
	}

	if err != nil {
		return errors.New("Write request resulted in error: " + err.Error())
	}
	return nil
}

func (ts *TransformerService) makeWriteRequest(uuid string, concordedJson []byte, tid string) error {
	reqURL := ts.writerAddress + "concordance/" + uuid
	request, err := http.NewRequest("PUT", reqURL, strings.NewReader(string(concordedJson)))
	if err != nil {
		return errors.New("Failed to create GET request to " + reqURL + " with body " + string(concordedJson))
	}
	request.ContentLength = -1
	request.Header.Set("X-Request-Id", tid)

	resp, reqErr := ts.httpClient.Do(request)

	if reqErr != nil {
		return errors.New("Get request to resulted in error: " + reqErr.Error())
	} else if resp.StatusCode != 200 {
		return errors.New("Get request to returned status: " + strconv.Itoa(resp.StatusCode))
	}

	defer resp.Body.Close()
	return nil
}

func (ts *TransformerService) makeDeleteRequest(uuid string, tid string) error {
	reqURL := ts.writerAddress + "concordances/" + uuid
	request, err := http.NewRequest("DELETE", reqURL, nil)
	if err != nil {
		return errors.New("Failed to create Delete request to " + reqURL)
	}
	request.ContentLength = -1
	request.Header.Set("X-Request-Id", tid)

	resp, reqErr := ts.httpClient.Do(request)

	if reqErr != nil {
		return errors.New("Delete request to resulted in error: " + reqErr.Error())
	} else if resp.StatusCode != 204 && resp.StatusCode != 404 {
		return errors.New("Delete request to returned status: " + strconv.Itoa(resp.StatusCode))
	}
	defer resp.Body.Close()
	return nil
}

func extractUuid(url string) string {
	extractedUuid := strings.Trim(url, "http://www.ft.com/thing/")

	if uuidMatcher.MatchString(extractedUuid) != true {
		return ""
	}
	return extractedUuid
}