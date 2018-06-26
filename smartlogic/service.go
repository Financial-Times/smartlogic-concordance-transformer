package smartlogic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/pborman/uuid"

	"github.com/Financial-Times/uuid-utils-go"
	log "github.com/sirupsen/logrus"
)

var uuidMatcher = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

type status int

const (
	CONCORDANCE_AUTHORITY_TME              = "FT-TME"
	CONCORDANCE_AUTHORITY_FACTSET          = "FACTSET"
	CONCORDANCE_AUTHORITY_DBPEDIA          = "DBPediaID"
	CONCORDANCE_AUTHORITY_GEONAMES         = "GeoNamesID"
	CONCORDANCE_AUTHORITY_WIKIDATA         = "WikiDataID"
	CONCORDANCE_AUTHORITY_SMARTLOGIC       = "SmartLogic"
	CONCORDANCE_AUTHORITY_MANAGED_LOCATION = "ManagedLocation"

	THING_URI_PREFIX               = "http://www.ft.com/thing/"
	LOCATION_URI_PREFIX            = "http://www.ft.com/managedlocation/"
	NOT_FOUND               status = iota
	SYNTACTICALLY_INCORRECT
	SEMANTICALLY_INCORRECT
	VALID_CONCEPT
	INTERNAL_ERROR
	SERVICE_UNAVAILABLE
	NO_CONTENT

	alertTagConceptTypeNotAllowed = "SmartlogicConcordanceTransformerConceptTypeNotAllowed"
)

var (
	notAllowedConceptTypes = [...]string{
		"skos:Concept",
	}

	errConceptTypeNotAllowed = errors.New("concept type not allowed")
)

type TransformerService struct {
	topic         string
	writerAddress string
	httpClient    httpClient
}

type httpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

func NewTransformerService(topic string, writerAddress string, httpClient httpClient) TransformerService {
	return TransformerService{
		topic:         topic,
		writerAddress: writerAddress,
		httpClient:    httpClient,
	}
}

func (ts *TransformerService) handleConcordanceEvent(msgBody string, tid string) error {
	log.WithField("transaction_id", tid).Debug("Processing message with body: " + msgBody)
	var smartLogicConceptPayload = SmartlogicConcept{}
	decoder := json.NewDecoder(bytes.NewBufferString(msgBody))
	err := decoder.Decode(&smartLogicConceptPayload)
	if err != nil {
		log.WithError(err).WithField("transaction_id", tid).Error("Failed to decode Kafka payload")
		return err
	}

	_, conceptUuid, uppConcordance, err := convertToUppConcordance(smartLogicConceptPayload, tid)
	if err != nil {
		return err
	}
	_, err = ts.makeRelevantRequest(conceptUuid, uppConcordance, tid)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Info("Forwarded concordance record to rw")
	return nil
}

func convertToUppConcordance(smartlogicConcepts SmartlogicConcept, tid string) (status, string, UppConcordance, error) {
	if len(smartlogicConcepts.Concepts) == 0 {
		err := errors.New("Invalid Request Json: Missing/invalid @graph field")
		log.WithField("transaction_id", tid).Error(err)
		return SEMANTICALLY_INCORRECT, "", UppConcordance{}, err
	}
	if len(smartlogicConcepts.Concepts) > 1 {
		err := errors.New("Invalid Request Json: More than 1 concept in smartlogic concept payload which is currently not supported")
		log.WithField("transaction_id", tid).Error(err)
		return SEMANTICALLY_INCORRECT, "", UppConcordance{}, err
	}

	smartlogicConcept := smartlogicConcepts.Concepts[0]

	conceptUuid, uppAuthority := extractUuidAndConcordanceAuthority(smartlogicConcept.Id)
	if conceptUuid == "" {
		err := errors.New("Invalid Request Json: Missing/invalid @id field")
		log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Error(err)
		return SEMANTICALLY_INCORRECT, conceptUuid, UppConcordance{}, err
	}

	if len(smartlogicConcept.Types) == 0 {
		err := fmt.Errorf("Bad Request: Type has not been set for concept: %s)", conceptUuid)
		log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Error(err)
		return SYNTACTICALLY_INCORRECT, conceptUuid, UppConcordance{}, err
	}

	conceptType := smartlogicConcept.Types[0]

	for _, bannedConceptType := range notAllowedConceptTypes {
		if conceptType != bannedConceptType {
			continue
		}
		log.WithFields(log.Fields{
			"transaction_id": tid,
			"UUID":           conceptUuid,
			"concept_type":   bannedConceptType,
			"alert_tag":      alertTagConceptTypeNotAllowed,
		}).Error(errConceptTypeNotAllowed)
		return SEMANTICALLY_INCORRECT, conceptUuid, UppConcordance{}, errConceptTypeNotAllowed
	}

	shortFormType := conceptType[strings.LastIndex(conceptType, "/")+1:]
	if (shortFormType == "Membership" || shortFormType == "MembershipRole") && len(smartlogicConcept.TmeIdentifiers) > 0 {
		err := fmt.Errorf("Bad Request: Concept type %s does not support concordance", shortFormType)
		log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Error(err)
		return SYNTACTICALLY_INCORRECT, conceptUuid, UppConcordance{}, err
	}

	concordances := []ConcordedId{}

	concordances, err := appendTmeConcordances(concordances, smartlogicConcept, conceptUuid, tid)

	if err != nil {
		return SYNTACTICALLY_INCORRECT, conceptUuid, UppConcordance{}, err
	}

	concordances, err = appendFactsetConcordances(concordances, smartlogicConcept, conceptUuid, tid)

	if err != nil {
		return SYNTACTICALLY_INCORRECT, conceptUuid, UppConcordance{}, err
	}

	concordances, err = appendLocationConcordances(concordances, smartlogicConcept.DbpediaIdentifiers, conceptUuid, CONCORDANCE_AUTHORITY_DBPEDIA, tid)
	concordances, err = appendLocationConcordances(concordances, smartlogicConcept.GeonamesIdentifiers, conceptUuid, CONCORDANCE_AUTHORITY_GEONAMES, tid)
	concordances, err = appendLocationConcordances(concordances, smartlogicConcept.WikidataIdentifiers, conceptUuid, CONCORDANCE_AUTHORITY_WIKIDATA, tid)

	uppConcordance := UppConcordance{}
	uppConcordance.ConceptUuid = conceptUuid
	uppConcordance.Authority = uppAuthority
	uppConcordance.ConcordedIds = concordances
	log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Debugf("Concordance record is %s", uppConcordance)

	return VALID_CONCEPT, conceptUuid, uppConcordance, nil
}

func appendTmeConcordances(concordances []ConcordedId, concept Concept, conceptUuid string, tid string) ([]ConcordedId, error) {
	for _, id := range concept.TmeIdentifiers {
		uuidFromTmeId, err := validateTmeIdAndConvertToUuid(id.Value)
		if conceptUuid == uuidFromTmeId {
			err := errors.New("Bad Request: Payload from smartlogic has a smartlogic uuid that is the same as the uuid generated from the TME id")
			log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Error(err)
			return nil, err
		}
		if err != nil {
			log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Error(err)
			return nil, err
		}
		concordedId := ConcordedId{
			Authority: CONCORDANCE_AUTHORITY_TME,
			UUID:      uuidFromTmeId,
		}
		if len(concordances) > 0 {
			for _, cid := range concordances {
				if cid.UUID == uuidFromTmeId {
					err := errors.New("Bad Request: Payload from smartlogic contains duplicate TME id values")
					log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Error(err)
					return nil, err
				}
			}
			concordances = append(concordances, concordedId)
		} else {
			concordances = append(concordances, concordedId)
		}
	}

	return concordances, nil
}

func appendFactsetConcordances(concordances []ConcordedId, concept Concept, conceptUuid string, tid string) ([]ConcordedId, error) {
	for _, id := range concept.FactsetIdentifiers {
		uuidFromFactsetId, err := validateFactsetIdAndConvertToUuid(id.Value)
		if conceptUuid == uuidFromFactsetId {
			err := errors.New("Bad Request: Payload from smartlogic has a smartlogic uuid that is the same as the uuid generated from the FACTSET id")
			log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Error(err)
			return nil, err
		}
		if err != nil {
			log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Error(err)
			return nil, err
		}
		concordedId := ConcordedId{
			Authority: CONCORDANCE_AUTHORITY_FACTSET,
			UUID:      uuidFromFactsetId,
		}
		if len(concordances) > 0 {
			for _, cid := range concordances {
				if cid.UUID == uuidFromFactsetId {
					err := errors.New("Bad Request: Payload from smartlogic contains duplicate FACTSET id values")
					log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Error(err)
					return nil, err
				}
			}
			concordances = append(concordances, concordedId)
		} else {
			concordances = append(concordances, concordedId)
		}
	}

	return concordances, nil
}

func appendLocationConcordances(concordances []ConcordedId, conceptIdentifiers []LocationType, conceptUuid string, authority string, tid string) ([]ConcordedId, error) {
	for _, id := range conceptIdentifiers {
		uuidFromConceptIdentifier := convertToUuid(id.Value)
		if conceptUuid == uuidFromConceptIdentifier {
			err := errors.New("Bad Request: Payload from smartlogic has a smartlogic uuid that is the same as the uuid generated from " + authority + " id")
			log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Error(err)
			return nil, err
		}
		concordedId := ConcordedId{
			Authority: authority,
			UUID:      uuidFromConceptIdentifier,
		}
		if len(concordances) > 0 {
			for _, cid := range concordances {
				if cid.UUID == uuidFromConceptIdentifier {
					err := errors.New("Bad Request: Payload from smartlogic contains duplicate " + authority + " id values")
					log.WithFields(log.Fields{"transaction_id": tid, "UUID": conceptUuid}).Error(err)
					return nil, err
				}
			}
			concordances = append(concordances, concordedId)
		} else {
			concordances = append(concordances, concordedId)
		}
	}

	return concordances, nil
}

func validateTmeIdAndConvertToUuid(tmeId string) (string, error) {
	subStrings := strings.Split(tmeId, "-")
	if len(subStrings) != 2 || !validateSubstrings(subStrings) {
		return "", errors.New("Bad Request: Concordance id " + tmeId + " is not a valid TME Id")
	} else {
		return uuid.NewMD5(uuid.UUID{}, []byte(tmeId)).String(), nil
	}
}

func validateFactsetIdAndConvertToUuid(factsetId string) (string, error) {
	if len(factsetId) != 8 || factsetId[0] != '0' || factsetId[6:8] != "-E" {
		return "", errors.New("Bad Request: Concordance id " + factsetId + " is not a valid FACTSET Id")
	}
	return uuidutils.DeriveFactsetUUID(factsetId), nil
}

func convertToUuid(id string) string {
	return uuid.NewMD5(uuid.UUID{}, []byte(id)).String()
}

func validateSubstrings(subStrings []string) bool {
	for _, string := range subStrings {
		if string == "" {
			return false
		}
	}
	return true
}

func (ts *TransformerService) makeRelevantRequest(uuid string, uppConcordance UppConcordance, tid string) (status, error) {
	var err error
	var reqStatus status
	if len(uppConcordance.ConcordedIds) > 0 {
		log.WithFields(log.Fields{"transaction_id": tid, "UUID": uuid}).Infof("Concordance record is: %s; forwarding request to writer", uppConcordance)
		reqStatus, err = ts.makeWriteRequest(uuid, uppConcordance, tid)
	} else {
		log.WithFields(log.Fields{"transaction_id": tid, "UUID": uuid}).Debug("No concordance found; making delete request")
		reqStatus, err = ts.makeDeleteRequest(uuid, tid)
	}

	return reqStatus, err
}

func (ts *TransformerService) makeWriteRequest(uuid string, uppConcordance UppConcordance, tid string) (status, error) {
	reqURL := ts.writerAddress + "branches/" + uuid
	concordedJson, err := json.Marshal(uppConcordance)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"transaction_id": tid, "UUID": uuid}).Error("Bad Request: Could not unmarshall concordance json")
		return SYNTACTICALLY_INCORRECT, err
	}

	request, err := http.NewRequest("PUT", reqURL, strings.NewReader(string(concordedJson)))
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"transaction_id": tid, "UUID": uuid}).Error("Internal Error: Failed to create GET request to " + reqURL + " with body " + string(concordedJson))
		return INTERNAL_ERROR, err
	}
	request.ContentLength = -1
	request.Header.Set("X-Request-Id", tid)

	resp, err := ts.httpClient.Do(request)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"transaction_id": tid, "UUID": uuid}).Error("Service Unavailable: Get request to writer resulted in error")
		return SERVICE_UNAVAILABLE, err
	} else if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 304 {
		err := errors.New("Internal Error: Get request to writer returned unexpected status: " + strconv.Itoa(resp.StatusCode))
		log.WithFields(log.Fields{"transaction_id": tid, "UUID": uuid, "status": resp.StatusCode}).Error(err)
		return INTERNAL_ERROR, err
	}

	defer resp.Body.Close()
	return VALID_CONCEPT, nil
}

func (ts *TransformerService) makeDeleteRequest(uuid string, tid string) (status, error) {
	reqURL := ts.writerAddress + "branches/" + uuid
	request, err := http.NewRequest("DELETE", reqURL, strings.NewReader(""))
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"transaction_id": tid, "UUID": uuid}).Error("Internal Error: Failed to create DELETE request to " + reqURL)
		return INTERNAL_ERROR, err
	}
	request.ContentLength = -1
	request.Header.Set("X-Request-Id", tid)

	resp, err := ts.httpClient.Do(request)

	if err != nil {
		log.WithError(err).WithFields(log.Fields{"transaction_id": tid, "UUID": uuid}).Error("Service Unavailable: Delete request to writer resulted in error")
		return SERVICE_UNAVAILABLE, err
	} else if resp.StatusCode != 204 && resp.StatusCode != 404 {
		err := errors.New("Internal Error: Delete request to writer returned unexpected status: " + strconv.Itoa(resp.StatusCode))
		log.WithFields(log.Fields{"transaction_id": tid, "UUID": uuid, "status": resp.StatusCode}).Error(err)
		return INTERNAL_ERROR, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 204 {
		return NO_CONTENT, nil
	}
	return NOT_FOUND, nil
}

func extractUuidAndConcordanceAuthority(url string) (string, string) {
	if strings.HasPrefix(url, THING_URI_PREFIX) {
		extractedUuid := strings.TrimPrefix(url, THING_URI_PREFIX)
		if uuidMatcher.MatchString(extractedUuid) != true {
			return "", ""
		}
		return extractedUuid, CONCORDANCE_AUTHORITY_SMARTLOGIC
	} else if strings.HasPrefix(url, LOCATION_URI_PREFIX) {
		extractedUuid := strings.TrimPrefix(url, LOCATION_URI_PREFIX)
		if uuidMatcher.MatchString(extractedUuid) != true {
			return "", ""
		}
		return extractedUuid, CONCORDANCE_AUTHORITY_MANAGED_LOCATION

	}
	return "", ""
}
