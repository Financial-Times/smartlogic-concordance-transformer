package smartlogic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/pborman/uuid"

	uuidUtils "github.com/Financial-Times/uuid-utils-go"
)

var uuidMatcher = regexp.MustCompile(`^[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}$`)

type status int

const (
	ConcordanceAuthorityTme             = "TME"
	ConcordanceAuthorityFactset         = "FACTSET"
	ConcordanceAuthorityDbpedia         = "DBPedia"
	ConcordanceAuthorityGeonames        = "Geonames"
	ConcordanceAuthorityWikidata        = "Wikidata"
	ConcordanceAuthoritySmartlogic      = "Smartlogic"
	ConcordanceAuthorityManagedLocation = "ManagedLocation"

	ThingURIPrefix           = "http://www.ft.com/thing/"
	LocationURIPrefix        = "http://www.ft.com/ontology/managedlocation/"
	NotFound          status = iota
	SyntacticallyIncorrect
	SemanticallyIncorrect
	ValidConcept
	InternalError
	ServiceUnavailable
	NoContent

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
	log           *logger.UPPLogger
}

type httpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

func NewTransformerService(topic string, writerAddress string, httpClient httpClient, log *logger.UPPLogger) TransformerService {
	return TransformerService{
		topic:         topic,
		writerAddress: writerAddress,
		httpClient:    httpClient,
		log:           log,
	}
}

func (ts *TransformerService) handleConcordanceEvent(msgBody string, tid string) error {
	ts.log.WithField("transaction_id", tid).Debug("Processing message with body: " + msgBody)
	var smartLogicConceptPayload = ConceptData{}
	decoder := json.NewDecoder(bytes.NewBufferString(msgBody))
	err := decoder.Decode(&smartLogicConceptPayload)
	if err != nil {
		ts.log.WithError(err).WithField("transaction_id", tid).Error("Failed to decode Kafka payload")
		return err
	}

	_, conceptUUID, uppConcordance, err := convertToUppConcordance(smartLogicConceptPayload, tid, ts.log)
	if err != nil {
		return err
	}
	_, err = ts.makeRelevantRequest(conceptUUID, uppConcordance, tid)
	if err != nil {
		return err
	}
	ts.log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID}).Info("Forwarded concordance record to rw")
	return nil
}

func convertToUppConcordance(Concepts ConceptData, tid string, log *logger.UPPLogger) (status, string, UppConcordance, error) {
	if len(Concepts.Concepts) == 0 {
		err := errors.New("invalid Request Json: Missing/invalid @graph field")
		log.WithField("transaction_id", tid).Error(err)
		return SemanticallyIncorrect, "", UppConcordance{}, err
	}
	if len(Concepts.Concepts) > 1 {
		err := errors.New("invalid Request Json: More than 1 concept in smartlogic concept payload which is currently not supported")
		log.WithField("transaction_id", tid).Error(err)
		return SemanticallyIncorrect, "", UppConcordance{}, err
	}

	concept := Concepts.Concepts[0]

	conceptUUID, uppAuthority := extractUUIDAndConcordanceAuthority(concept.ID)
	if conceptUUID == "" {
		err := errors.New("invalid Request Json: Missing/invalid @id field")
		log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID}).Error(err)
		return SemanticallyIncorrect, conceptUUID, UppConcordance{}, err
	}

	if len(concept.Types) == 0 {
		err := fmt.Errorf("bad Request: Type has not been set for concept: %s)", conceptUUID)
		log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID}).Error(err)
		return SyntacticallyIncorrect, conceptUUID, UppConcordance{}, err
	}

	conceptType := concept.Types[0]

	for _, bannedConceptType := range notAllowedConceptTypes {
		if conceptType != bannedConceptType {
			continue
		}
		log.WithFields(map[string]interface{}{
			"transaction_id": tid,
			"UUID":           conceptUUID,
			"concept_type":   bannedConceptType,
			"alert_tag":      alertTagConceptTypeNotAllowed,
		}).Error(errConceptTypeNotAllowed)
		return SemanticallyIncorrect, conceptUUID, UppConcordance{}, errConceptTypeNotAllowed
	}

	shortFormType := conceptType[strings.LastIndex(conceptType, "/")+1:]
	if (shortFormType == "Membership" || shortFormType == "MembershipRole") && len(concept.TmeIdentifiers()) > 0 {
		err := fmt.Errorf("bad Request: Concept type %s does not support concordance", shortFormType)
		log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID}).Error(err)
		return SyntacticallyIncorrect, conceptUUID, UppConcordance{}, err
	}

	//replacing with nil slice breaks tests
	concordances := []ConcordedID{}

	concordances, err := appendTmeConcordances(concordances, concept, conceptUUID, tid, log)

	if err != nil {
		return SyntacticallyIncorrect, conceptUUID, UppConcordance{}, err
	}

	concordances, err = appendFactsetConcordances(concordances, concept, conceptUUID, tid, log)

	if err != nil {
		return SyntacticallyIncorrect, conceptUUID, UppConcordance{}, err
	}

	concordances, err = appendLocationConcordances(concordances, concept.DbpediaIdentifiers(), conceptUUID, ConcordanceAuthorityDbpedia, tid, log)
	if err != nil {
		return SyntacticallyIncorrect, conceptUUID, UppConcordance{}, err
	}

	concordances, err = appendLocationConcordances(concordances, concept.GeonamesIdentifiers(), conceptUUID, ConcordanceAuthorityGeonames, tid, log)
	if err != nil {
		return SyntacticallyIncorrect, conceptUUID, UppConcordance{}, err
	}

	concordances, err = appendLocationConcordances(concordances, concept.WikidataIdentifiers(), conceptUUID, ConcordanceAuthorityWikidata, tid, log)
	if err != nil {
		return SyntacticallyIncorrect, conceptUUID, UppConcordance{}, err
	}

	uppConcordance := UppConcordance{
		ConceptUUID:  conceptUUID,
		Authority:    uppAuthority,
		ConcordedIds: concordances,
	}
	log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID}).Debugf("Concordance record is %s", uppConcordance)

	return ValidConcept, conceptUUID, uppConcordance, nil
}

func appendTmeConcordances(concordances []ConcordedID, concept Concept, conceptUUID string, tid string, log *logger.UPPLogger) ([]ConcordedID, error) {
	for _, id := range concept.TmeIdentifiers() {
		uuidFromTmeID, err := validateTmeIDAndConvertToUUID(id.Value)
		if conceptUUID == uuidFromTmeID {
			err := errors.New("bad Request: Payload from smartlogic has a smartlogic uuid that is the same as the uuid generated from the TME id")
			log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID}).Error(err)
			return nil, err
		}
		if err != nil {
			log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID, "alert_tag": "ConceptLoadingInvalidConcordance"}).Error(err)
			return nil, err
		}
		concordedID := ConcordedID{
			Authority:      ConcordanceAuthorityTme,
			AuthorityValue: id.Value,
			UUID:           uuidFromTmeID,
		}
		if len(concordances) > 0 {
			for _, cid := range concordances {
				if cid.UUID == uuidFromTmeID {
					err := errors.New("bad Request: Payload from smartlogic contains duplicate TME id values")
					log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID}).Error(err)
					return nil, err
				}
			}
			concordances = append(concordances, concordedID)
		} else {
			concordances = append(concordances, concordedID)
		}
	}

	return concordances, nil
}

func appendFactsetConcordances(concordances []ConcordedID, concept Concept, conceptUUID string, tid string, log *logger.UPPLogger) ([]ConcordedID, error) {
	for _, id := range concept.FactsetIdentifiers() {
		uuidFromFactsetID, err := validateFactsetIDAndConvertToUUID(id.Value)
		if conceptUUID == uuidFromFactsetID {
			err := errors.New("bad Request: Payload from smartlogic has a smartlogic uuid that is the same as the uuid generated from the FACTSET id")
			log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID}).Error(err)
			return nil, err
		}
		if err != nil {
			log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID, "alert_tag": "ConceptLoadingInvalidConcordance"}).Error(err)
			return nil, err
		}
		concordedID := ConcordedID{
			Authority:      ConcordanceAuthorityFactset,
			AuthorityValue: id.Value,
			UUID:           uuidFromFactsetID,
		}
		if len(concordances) > 0 {
			for _, cid := range concordances {
				if cid.UUID == uuidFromFactsetID {
					err := errors.New("bad Request: Payload from smartlogic contains duplicate FACTSET id values")
					log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID}).Error(err)
					return nil, err
				}
			}
			concordances = append(concordances, concordedID)
		} else {
			concordances = append(concordances, concordedID)
		}
	}

	return concordances, nil
}

func appendLocationConcordances(concordances []ConcordedID, conceptIdentifiers []LocationType, conceptUUID string, authority string, tid string, log *logger.UPPLogger) ([]ConcordedID, error) {
	for _, id := range conceptIdentifiers {
		if len(strings.TrimSpace(id.Value)) == 0 {
			log.WithFields(map[string]interface{}{"transaction_id": tid, "uuid": conceptUUID}).Warn(fmt.Sprintf("Payload from Smartlogic contains one or more empty %v values. Skipping it", authority))
			continue
		}

		uuidFromConceptIdentifier := convertToUUID(id.Value)
		if conceptUUID == uuidFromConceptIdentifier {
			err := fmt.Errorf("bad Request: Payload from Smartlogic has a Smartlogic uuid that is the same as the uuid generated from %v id", authority)
			log.WithFields(map[string]interface{}{"transaction_id": tid, "uuid": conceptUUID}).Error(err)
			return nil, err
		}
		if concordancesContainValue(concordances, uuidFromConceptIdentifier) {
			log.WithFields(map[string]interface{}{"transaction_id": tid, "uuid": conceptUUID}).Warn(fmt.Sprintf("Payload from Smartlogic contains duplicate %v values. Skipping it", authority))
			continue
		}

		concordedID := ConcordedID{
			Authority:      authority,
			AuthorityValue: id.Value,
			UUID:           uuidFromConceptIdentifier,
		}
		if len(concordances) > 0 {
			for _, cid := range concordances {
				if cid.UUID == uuidFromConceptIdentifier {
					err := errors.New("Bad Request: Payload from smartlogic contains duplicate " + authority + " id values")
					log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": conceptUUID}).Error(err)
					return nil, err
				}
			}
			concordances = append(concordances, concordedID)
		} else {
			concordances = append(concordances, concordedID)
		}
	}

	return concordances, nil
}

func validateTmeIDAndConvertToUUID(tmeID string) (string, error) {
	subStrings := strings.Split(tmeID, "-")
	if len(subStrings) != 2 || !validateSubstrings(subStrings) {
		return "", errors.New("Bad Request: Concordance id " + tmeID + " is not a valid TME Id")
	}
	return uuid.NewMD5(uuid.UUID{}, []byte(tmeID)).String(), nil
}

func validateFactsetIDAndConvertToUUID(factsetID string) (string, error) {
	if len(factsetID) != 8 || factsetID[0] != '0' || factsetID[6:8] != "-E" {
		return "", errors.New("Bad Request: Concordance id " + factsetID + " is not a valid FACTSET Id")
	}
	return uuidUtils.DeriveFactsetUUID(factsetID), nil
}

func convertToUUID(id string) string {
	return uuid.NewMD5(uuid.UUID{}, []byte(id)).String()
}

func validateSubstrings(subStrings []string) bool {
	for _, sub := range subStrings {
		if sub == "" {
			return false
		}
	}
	return true
}

func (ts *TransformerService) makeRelevantRequest(uuid string, uppConcordance UppConcordance, tid string) (status, error) {
	var err error
	var reqStatus status
	if len(uppConcordance.ConcordedIds) > 0 {
		ts.log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": uuid}).Infof("Concordance record is: %s; forwarding request to writer", uppConcordance)
		reqStatus, err = ts.makeWriteRequest(uuid, uppConcordance, tid)
	} else {
		ts.log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": uuid}).Debug("No concordance found; making delete request")
		reqStatus, err = ts.makeDeleteRequest(uuid, tid)
	}

	return reqStatus, err
}

func (ts *TransformerService) makeWriteRequest(uuid string, uppConcordance UppConcordance, tid string) (status, error) {
	reqURL := ts.writerAddress + "branches/" + uuid
	concordedJSON, err := json.Marshal(uppConcordance)
	if err != nil {
		ts.log.WithError(err).WithFields(map[string]interface{}{"transaction_id": tid, "UUID": uuid}).Error("Bad Request: Could not unmarshall concordance json")
		return SyntacticallyIncorrect, err
	}

	request, err := http.NewRequest("PUT", reqURL, strings.NewReader(string(concordedJSON)))
	if err != nil {
		ts.log.WithError(err).WithFields(map[string]interface{}{"transaction_id": tid, "UUID": uuid}).Error("Internal Error: Failed to create GET request to " + reqURL + " with body " + string(concordedJSON))
		return InternalError, err
	}
	request.ContentLength = -1
	request.Header.Set("X-Request-Id", tid)

	resp, err := ts.httpClient.Do(request)
	if err != nil {
		ts.log.WithError(err).WithFields(map[string]interface{}{"transaction_id": tid, "UUID": uuid}).Error("Service Unavailable: Get request to writer resulted in error")
		return ServiceUnavailable, err
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			ts.log.WithError(err).Info("Could not close body")
		}
	}(resp.Body)

	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 304 {
		err := errors.New("Internal Error: Get request to writer returned unexpected status: " + strconv.Itoa(resp.StatusCode))
		ts.log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": uuid, "status": resp.StatusCode}).Error(err)
		return InternalError, err
	}

	return ValidConcept, nil
}

func (ts *TransformerService) makeDeleteRequest(uuid string, tid string) (status, error) {
	reqURL := ts.writerAddress + "branches/" + uuid
	request, err := http.NewRequest("DELETE", reqURL, strings.NewReader(""))
	if err != nil {
		ts.log.WithError(err).WithFields(map[string]interface{}{"transaction_id": tid, "UUID": uuid}).Error("Internal Error: Failed to create DELETE request to " + reqURL)
		return InternalError, err
	}
	request.ContentLength = -1
	request.Header.Set("X-Request-Id", tid)

	resp, err := ts.httpClient.Do(request)
	if err != nil {
		ts.log.WithError(err).WithFields(map[string]interface{}{"transaction_id": tid, "UUID": uuid}).Error("Service Unavailable: Delete request to writer resulted in error")
		return ServiceUnavailable, err
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			ts.log.WithError(err).Info("Could not close body")
		}
	}(resp.Body)

	if resp.StatusCode != 204 && resp.StatusCode != 404 {
		err := errors.New("Internal Error: Delete request to writer returned unexpected status: " + strconv.Itoa(resp.StatusCode))
		ts.log.WithFields(map[string]interface{}{"transaction_id": tid, "UUID": uuid, "status": resp.StatusCode}).Error(err)
		return InternalError, err
	}
	if resp.StatusCode == 204 {
		return NoContent, nil
	}
	return NotFound, nil
}

func extractUUIDAndConcordanceAuthority(url string) (string, string) {
	if strings.HasPrefix(url, ThingURIPrefix) {
		extractedUUID := strings.TrimPrefix(url, ThingURIPrefix)
		if !uuidMatcher.MatchString(extractedUUID) {
			return "", ""
		}
		return extractedUUID, ConcordanceAuthoritySmartlogic
	} else if strings.HasPrefix(url, LocationURIPrefix) {
		extractedUUID := strings.TrimPrefix(url, LocationURIPrefix)
		if !uuidMatcher.MatchString(extractedUUID) {
			return "", ""
		}
		return extractedUUID, ConcordanceAuthorityManagedLocation
	}
	return "", ""
}

func concordancesContainValue(concordances []ConcordedID, value string) bool {
	for _, concordance := range concordances {
		if concordance.UUID == value {
			return true
		}
	}
	return false
}
