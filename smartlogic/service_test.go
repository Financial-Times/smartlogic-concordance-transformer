package smartlogic

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testUuid             = "20db1bd6-59f9-4404-adb5-3165a448f8b0"
	concordedTmeUuid     = "d83a4dc1-397e-4f99-8ecf-2f1b15febb7f"
	concordedFactsetUuid = "asdf"
	writerUrl            = "http://localhost:8080/"
)

var (
	concordedTmeId = ConcordedId{
		Authority: CONCORDANCE_AUTHORITY_TME,
		UUID:      concordedTmeUuid,
	}
	concordedFactsetId = ConcordedId{
		Authority: CONCORDANCE_AUTHORITY_FACTSET,
		UUID:      concordedFactsetUuid,
	}
)

func TestValidateSubstrings(t *testing.T) {
	type testStruct struct {
		testName       string
		tmeIdParts     []string
		expectedResult bool
	}

	oneValidSubstring := testStruct{testName: "oneValidSubstring", tmeIdParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ"}, expectedResult: true}
	twoValidSubstring := testStruct{testName: "twoValidSubstring", tmeIdParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ", "jYTE5NDEyM2Yw"}, expectedResult: true}
	thirdSubstringIsEmpty := testStruct{testName: "thirdSubstringIsEmpty", tmeIdParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ", "jYTE5NDEyM2Yw", ""}, expectedResult: false}
	secondSubstringIsEmpty := testStruct{testName: "secondSubstringIsEmpty", tmeIdParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ", ""}, expectedResult: false}
	firstSubstringIsEmpty := testStruct{testName: "firstSubstringIsEmpty", tmeIdParts: []string{"", "jYTE5NDEyM2Yw"}, expectedResult: false}
	onlySubstringIsEmpty := testStruct{testName: "onlySubstringIsEmpty", tmeIdParts: []string{""}, expectedResult: false}

	testScenarios := []testStruct{oneValidSubstring, twoValidSubstring, thirdSubstringIsEmpty, secondSubstringIsEmpty, firstSubstringIsEmpty, onlySubstringIsEmpty}

	for _, scenario := range testScenarios {
		substringsAreValid := validateSubstrings(scenario.tmeIdParts)
		assert.Equal(t, scenario.expectedResult, substringsAreValid, "Scenario: "+scenario.testName+" failed")
	}
}

func TestValidateTmeIdAndConvertToUuid(t *testing.T) {
	type testStruct struct {
		testName      string
		tmeId         string
		expectedUuid  string
		expectedError error
	}

	invalidTmeIdHasNoHyphen := testStruct{testName: "invalidTmeId", tmeId: "YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJjYTE5NDEyM2Yw", expectedUuid: "", expectedError: errors.New("Bad Request: Concordance id YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJjYTE5NDEyM2Yw is not a valid TME Id")}
	invalidTmeIdHasNoTaxonomy := testStruct{testName: "invalidTmeIdHasNoTaxonomy", tmeId: "YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNm-", expectedUuid: "", expectedError: errors.New("Bad Request: Concordance id YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNm- is not a valid TME Id")}
	invalidTmeIdHasNoValue := testStruct{testName: "invalidTmeIdHasNoValue", tmeId: "-JjYTE5NDEyM2Yw", expectedUuid: "", expectedError: errors.New("Bad Request: Concordance id -JjYTE5NDEyM2Yw is not a valid TME Id")}
	invalidTmeIdHasTooManyParts := testStruct{testName: "invalidTmeIdHasTooManyParts", tmeId: "YzhlNzZkYTctMDJi-Ny00NTViLTk3NmYtNm-JjYTE5NDEyM2Yw", expectedUuid: "", expectedError: errors.New("Bad Request: Concordance id YzhlNzZkYTctMDJi-Ny00NTViLTk3NmYtNm-JjYTE5NDEyM2Yw is not a valid TME Id")}
	validTmeIdIsConverted := testStruct{testName: "validTmeIdIsConverted", tmeId: "YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ-jYTE5NDEyM2Yw", expectedUuid: "a50ffd61-e9da-3c71-85ad-81ce983bcbf6", expectedError: nil}

	testScenarios := []testStruct{invalidTmeIdHasNoHyphen, invalidTmeIdHasNoTaxonomy, invalidTmeIdHasNoValue, invalidTmeIdHasTooManyParts, validTmeIdIsConverted}

	for _, scenario := range testScenarios {
		uuid, err := validateTmeIdAndConvertToUuid(scenario.tmeId)
		assert.Equal(t, scenario.expectedUuid, uuid, "Scenario: "+scenario.testName+" failed")
		assert.Equal(t, scenario.expectedError, err, "Scenario: "+scenario.testName+" failed")
	}
}

func TestValidateFactsetIdAndConvertToUuid(t *testing.T) {
	type testStruct struct {
		testName      string
		factsetId     string
		expectedUuid  string
		expectedError error
	}

	invalidFactsetIdNoZeroPrefix := testStruct{testName: "invalidFactsetIdNoZeroPrefix", factsetId: "123456-E", expectedUuid: "", expectedError: errors.New("Bad Request: Concordance id 123456-E is not a valid FACTSET Id")}
	invalidFactsetINoESuffix := testStruct{testName: "invalidFactsetINoESuffix", factsetId: "023456-A", expectedUuid: "", expectedError: errors.New("Bad Request: Concordance id 023456-A is not a valid FACTSET Id")}
	invalidFactsetIdNoHyphenSuffix := testStruct{testName: "invalidFactsetIdNoHyphenSuffix", factsetId: "0123456E", expectedUuid: "", expectedError: errors.New("Bad Request: Concordance id 0123456E is not a valid FACTSET Id")}
	validFactsetIdIsConverted := testStruct{testName: "validFactsetIdIsConverted", factsetId: "012345-E", expectedUuid: "949a7e7f-2516-30c0-9123-f866601ffbe4", expectedError: nil}

	testScenarios := []testStruct{invalidFactsetIdNoZeroPrefix, invalidFactsetINoESuffix, invalidFactsetIdNoHyphenSuffix, validFactsetIdIsConverted}

	for _, scenario := range testScenarios {
		uuid, err := validateFactsetIdAndConvertToUuid(scenario.factsetId)
		assert.Equal(t, scenario.expectedUuid, uuid, "Scenario: "+scenario.testName+" failed")
		assert.Equal(t, scenario.expectedError, err, "Scenario: "+scenario.testName+" failed")
	}
}

func TestExtractUuidAndConcordanceAuthority(t *testing.T) {
	type testStruct struct {
		testName       string
		url            string
		expectedResult string
	}

	invalidUrlMissingFtPrefix := testStruct{testName: "invalidUrlMissingFtPrefix", url: "www.google.com/2d3e16e0-61cb-4322-8aff-3b01c59f4daa", expectedResult: ""}
	invalidUrlWithInvalidUuid := testStruct{testName: "invalidUrlWithInvalidUuid", url: "http://www.ft.com/thing/2d3e16e061cb43228aff3b01c59f4daa", expectedResult: ""}
	ValidUrlIsConvertedToUuid := testStruct{testName: "ValidUrlIsConvertedToUuid", url: "http://www.ft.com/thing/2d3e16e0-61cb-4322-8aff-3b01c59f4daa", expectedResult: "2d3e16e0-61cb-4322-8aff-3b01c59f4daa"}

	testScenarios := []testStruct{invalidUrlMissingFtPrefix, invalidUrlWithInvalidUuid, ValidUrlIsConvertedToUuid}

	for _, scenario := range testScenarios {
		uuid, _ := extractUuidAndConcordanceAuthority(scenario.url)
		assert.Equal(t, scenario.expectedResult, uuid, "Scenario: "+scenario.testName+" failed")
	}
}

func TestMakeRelevantRequest(t *testing.T) {
	withConcordance := UppConcordance{ConceptUuid: testUuid, ConcordedIds: []ConcordedId{concordedTmeId}}
	noConcordance := UppConcordance{ConceptUuid: testUuid, ConcordedIds: []ConcordedId{}}
	type testStruct struct {
		testName       string
		uuid           string
		uppConcordance UppConcordance
		expectedError  error
		clientResp     string
		statusCode     int
		clientErr      error
	}

	concordanceFound_WriteError := testStruct{testName: "concordanceFound_WriteError", uuid: testUuid, uppConcordance: withConcordance, expectedError: errors.New("Get request resulted in error"), clientResp: "", statusCode: 200, clientErr: errors.New("Get request resulted in error")}
	concordanceFound_Status503 := testStruct{testName: "concordanceFound_Status503", uuid: testUuid, uppConcordance: withConcordance, expectedError: errors.New("Internal Error: Get request to writer returned unexpected status:"), clientResp: "", statusCode: 503, clientErr: nil}
	noConcordance_SuccessfulWrite := testStruct{testName: "noConcordance_SuccessfulWrite", uuid: testUuid, uppConcordance: withConcordance, expectedError: nil, clientResp: "", statusCode: 200, clientErr: nil}
	noConcordance_WriteError := testStruct{testName: "noConcordance_WriteError", uuid: testUuid, uppConcordance: noConcordance, expectedError: errors.New("Delete request resulted in error"), clientResp: "", statusCode: 200, clientErr: errors.New("Delete request resulted in error")}
	noConcordance_Status503 := testStruct{testName: "noConcordance_Status503", uuid: testUuid, uppConcordance: noConcordance, expectedError: errors.New("Internal Error: Delete request to writer returned unexpected status:"), clientResp: "", statusCode: 503, clientErr: nil}
	noConcordance_RecordNotFound := testStruct{testName: "noConcordance_RecordNotFound", uuid: testUuid, uppConcordance: noConcordance, expectedError: nil, clientResp: "", statusCode: 204, clientErr: nil}
	noConcordance_SuccessfulDelete := testStruct{testName: "noConcordance_SuccessfulDelete", uuid: testUuid, uppConcordance: noConcordance, expectedError: nil, clientResp: "", statusCode: 404, clientErr: nil}

	testScenarios := []testStruct{concordanceFound_WriteError, concordanceFound_Status503, noConcordance_SuccessfulWrite, noConcordance_WriteError, noConcordance_Status503, noConcordance_SuccessfulDelete, noConcordance_RecordNotFound}

	for _, scenario := range testScenarios {
		ts := NewTransformerService("", writerUrl, mockHttpClient{resp: scenario.clientResp, statusCode: scenario.statusCode, err: scenario.clientErr})
		_, reqErr := ts.makeRelevantRequest(scenario.uuid, scenario.uppConcordance, "")
		if reqErr != nil {
			assert.Contains(t, reqErr.Error(), scenario.expectedError.Error(), "Scenario: "+scenario.testName+" failed")
		} else {
			assert.Equal(t, reqErr, scenario.expectedError, "Scenario: "+scenario.testName+" failed")
		}
	}
}

func TestConvertToUppConcordance(t *testing.T) {
	noConcordance := UppConcordance{ConceptUuid: ""}
	emptyConcordance := UppConcordance{
		ConceptUuid:  testUuid,
		ConcordedIds: []ConcordedId{},
		Authority:    "SmartLogic",
	}
	multiConcordance := UppConcordance{
		ConceptUuid: testUuid,
		Authority:   "SmartLogic",
		ConcordedIds: []ConcordedId{
			ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_TME,
				AuthorityValue: "AbCdEfgHiJkLMnOpQrStUvWxYz-0123456789",
				UUID:           "e9f4525a-401f-3b23-a68e-e48f314cdce6",
			}, ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_TME,
				AuthorityValue: "ZyXwVuTsRqPoNmLkJiHgFeDcBa-0987654321",
				UUID:           "83f63c7e-1641-3c7b-81e4-378ae3c6c2ad",
			}, ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_TME,
				AuthorityValue: "abcdefghijklmnopqrstuvwxyz-0123456789",
				UUID:           "e4bc4ac2-0637-3a27-86b1-9589fca6bf2c",
			}, ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_TME,
				AuthorityValue: "ABCDEFGHIJKLMNOPQRSTUVWXYZ-0987654321",
				UUID:           "e574b21d-9abc-3d82-a6c0-3e08c85181bf",
			},
		},
	}
	multiFactsetConcordance := UppConcordance{
		ConceptUuid: testUuid,
		Authority:   "SmartLogic",
		ConcordedIds: []ConcordedId{
			ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_FACTSET,
				AuthorityValue: "000D63-E",
				UUID:           "8d3aba95-02d9-3802-afc0-b99bb9b1139e",
			}, ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_FACTSET,
				AuthorityValue: "023456-E",
				UUID:           "3bc0ab41-c01f-3a0b-aa78-c76438080b52",
			}, ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_FACTSET,
				AuthorityValue: "023411-E",
				UUID:           "f777c5af-e0b2-34dc-9102-e346ca2d27aa",
			},
		},
	}
	multiTmeFactsetConcordance := UppConcordance{
		ConceptUuid: testUuid,
		Authority:   "ManagedLocation",
		ConcordedIds: []ConcordedId{
			ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_TME,
				AuthorityValue: "ZyXwVuTsRqPoNmLkJiHgFeDcBa-0987654321",
				UUID:           "83f63c7e-1641-3c7b-81e4-378ae3c6c2ad",
			},
			ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_FACTSET,
				AuthorityValue: "023456-E",
				UUID:           "3bc0ab41-c01f-3a0b-aa78-c76438080b52",
			},
		},
	}
	locationsConcordance := UppConcordance{
		ConceptUuid: testUuid,
		Authority:   "ManagedLocation",
		ConcordedIds: []ConcordedId{
			ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_TME,
				AuthorityValue: "TnN0ZWluX0dMX0dCX0VOR19HX0Vzc2V4-R0w=",
				UUID:           "3f494231-9dc6-3181-8baa-dc9d1cad730f",
			}, ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_DBPEDIA,
				AuthorityValue: "http://dbpedia.org/resource/Essex",
				UUID:           "9567fbd6-f6f3-34f4-9b31-53856d5428a3",
			}, ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_GEONAMES,
				AuthorityValue: "http://sws.geonames.org/2649889/",
				UUID:           "ed78ef90-a160-30d0-8a3b-472a966c5664",
			}, ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_WIKIDATA,
				AuthorityValue: "http://www.wikidata.org/entity/Q23240",
				UUID:           "76754d1e-11f6-3d4f-8e3a-59a5b4e6bdcd",
			},
		},
	}

	editorialConcordance := UppConcordance{
		ConceptUuid: testUuid,
		Authority:   "SmartLogic",
		ConcordedIds: []ConcordedId{
			ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_TME,
				AuthorityValue: "TnN0ZWluX0dMX0dCX0VOR19HX0Vzc2V4-R0w=",
				UUID:           "3f494231-9dc6-3181-8baa-dc9d1cad730f",
			}, ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_WIKIDATA,
				AuthorityValue: "http://www.wikidata.org/entity/Q23240",
				UUID:           "76754d1e-11f6-3d4f-8e3a-59a5b4e6bdcd",
			},
		},
	}

	editorialConcordanceTwoWikidata := UppConcordance{
		ConceptUuid: testUuid,
		Authority:   "SmartLogic",
		ConcordedIds: []ConcordedId{
			ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_TME,
				AuthorityValue: "TnN0ZWluX0dMX0dCX0VOR19HX0Vzc2V4-R0w=",
				UUID:           "3f494231-9dc6-3181-8baa-dc9d1cad730f",
			}, ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_WIKIDATA,
				AuthorityValue: "http://www.wikidata.org/entity/Q23240",
				UUID:           "76754d1e-11f6-3d4f-8e3a-59a5b4e6bdcd",
			},
			ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_WIKIDATA,
				AuthorityValue: "http://www.wikidata.org/entity/Q23245",
				UUID:           "226ee6c7-8e94-3eb8-8370-c89ee9f9f988",
			},
		},
	}

	noWikidataEditorialConcordance := UppConcordance{
		ConceptUuid: testUuid,
		Authority:   "SmartLogic",
		ConcordedIds: []ConcordedId{
			ConcordedId{
				Authority:      CONCORDANCE_AUTHORITY_TME,
				AuthorityValue: "TnN0ZWluX0dMX0dCX0VOR19HX0Vzc2V4-R0w=",
				UUID:           "3f494231-9dc6-3181-8baa-dc9d1cad730f",
			},
		},
	}

	type testStruct struct {
		testName       string
		pathToFile     string
		conceptUuid    string
		uppConcordance UppConcordance
		expectedError  error
	}

	missingRequiredFieldsJson := testStruct{testName: "missingRequiredFieldsJson", pathToFile: "../resources/missingIdField.json", conceptUuid: "", uppConcordance: noConcordance, expectedError: errors.New("Missing/invalid @graph field")}
	invalidTmeListInputJson := testStruct{testName: "invalidTmeListInputJson", pathToFile: "../resources/invalidTmeListInput.json", conceptUuid: testUuid, uppConcordance: noConcordance, expectedError: errors.New("is not a valid TME Id")}
	invalidIdFieldJson := testStruct{testName: "invalidIdFieldJson", pathToFile: "../resources/invalidIdValue.json", conceptUuid: "", uppConcordance: noConcordance, expectedError: errors.New("Missing/invalid @id field")}
	missingTypesField := testStruct{testName: "missingTypesField", pathToFile: "../resources/noTypes.json", conceptUuid: testUuid, uppConcordance: noConcordance, expectedError: errors.New("Bad Request: Type has not been set for concept: 20db1bd6-59f9-4404-adb5-3165a448f8b0")}
	membershipNoConcordanceNoError := testStruct{testName: "membershipNoConcordanceNoError", pathToFile: "../resources/conceptIsMembershipNoConcordance.json", conceptUuid: testUuid, uppConcordance: emptyConcordance, expectedError: nil}
	errorOnMembershipConcept := testStruct{testName: "errorOnMembershipConcept", pathToFile: "../resources/conceptIsMembership.json", conceptUuid: testUuid, uppConcordance: noConcordance, expectedError: errors.New("Bad Request: Concept type Membership does not support concordance")}
	errorOnMembershipRoleConcept := testStruct{testName: "errorOnMembershipRoleConcept", pathToFile: "../resources/conceptIsMembershipRole.json", conceptUuid: testUuid, uppConcordance: noConcordance, expectedError: errors.New("Bad Request: Concept type MembershipRole does not support concordance")}
	invalidTmeId := testStruct{testName: "invalidTmeId", pathToFile: "../resources/invalidTmeId.json", conceptUuid: testUuid, uppConcordance: noConcordance, expectedError: errors.New("is not a valid TME Id")}
	tmeGeneratedUuidEqualConceptUuid := testStruct{testName: "tmeGeneratedUuidEqualConceptUuid", pathToFile: "../resources/tmeGeneratedUuidEqualConceptUuid.json", conceptUuid: "e9f4525a-401f-3b23-a68e-e48f314cdce6", uppConcordance: noConcordance, expectedError: errors.New("smartlogic uuid that is the same as the uuid generated from the TME id")}
	errorOnDuplicateTmeIds := testStruct{testName: "errorOnDuplicateTmeIds", pathToFile: "../resources/duplicateTmeIds.json", conceptUuid: testUuid, uppConcordance: noConcordance, expectedError: errors.New("contains duplicate TME id values")}
	handlesMultipleTmeIds := testStruct{testName: "handlesMultipleTmeIds", pathToFile: "../resources/multipleTmeIds.json", conceptUuid: testUuid, uppConcordance: multiConcordance, expectedError: nil}
	handlesNoTmeIds := testStruct{testName: "handlesNoTmeIds", pathToFile: "../resources/noTmeIds.json", conceptUuid: testUuid, uppConcordance: emptyConcordance, expectedError: nil}
	managedLocationIds := testStruct{testName: "managedLocationIds", pathToFile: "../resources/managedLocationIds.json", conceptUuid: testUuid, uppConcordance: locationsConcordance, expectedError: nil}
	managedLocationDuplicateIds := testStruct{testName: "managedLocationDuplicateIds", pathToFile: "../resources/managedLocationDuplicateIds.json", conceptUuid: testUuid, uppConcordance: locationsConcordance, expectedError: nil}
	managedLocationBlankId := testStruct{testName: "managedLocationBlankId", pathToFile: "../resources/managedLocationBlankId.json", conceptUuid: testUuid, uppConcordance: locationsConcordance, expectedError: nil}
	managedLocationMutuallyExclusiveFields := testStruct{testName: "managedLocationMutuallyExclusiveFields", pathToFile: "../resources/managedLocationMutuallyExclusiveFields.json", conceptUuid: testUuid, uppConcordance: multiTmeFactsetConcordance, expectedError: nil}
	editorialBlankId := testStruct{testName: "editorialBlankId", pathToFile: "../resources/editorialBlankId.json", conceptUuid: testUuid, uppConcordance: noWikidataEditorialConcordance, expectedError: nil}
	editorialDuplicateIds := testStruct{testName: "editorialDuplicateIds", pathToFile: "../resources/editorialDuplicateIds.json", conceptUuid: testUuid, uppConcordance: editorialConcordance, expectedError: nil}
	editorialAndManagedLocationWikidata := testStruct{testName: "editorialAndManagedLocationWikidata", pathToFile: "../resources/editorialAndManagedLocationWikidata.json", conceptUuid: testUuid, uppConcordance: editorialConcordance, expectedError: nil}
	editorialTwoWikidataIds := testStruct{testName: "editorialTwoWikidataIds", pathToFile: "../resources/editorialTwoWikidata.json", conceptUuid: testUuid, uppConcordance: editorialConcordanceTwoWikidata, expectedError: nil}

	invalidFactsetId := testStruct{
		testName:       "invalidFactsetId",
		pathToFile:     "../resources/invalidFactsetId.json",
		conceptUuid:    testUuid,
		uppConcordance: noConcordance,
		expectedError:  errors.New("is not a valid FACTSET Id"),
	}
	errorOnDuplicateFactsetIds := testStruct{
		testName:       "errorOnDuplicateFactsetIds",
		pathToFile:     "../resources/duplicateFactsetIds.json",
		conceptUuid:    testUuid,
		uppConcordance: noConcordance,
		expectedError:  errors.New("contains duplicate FACTSET id values"),
	}
	noErrorOnNotAllowedConceptType := testStruct{
		testName:       "noErrorOnNotAllowedConceptType",
		pathToFile:     "../resources/notAllowedType.json",
		conceptUuid:    testUuid,
		uppConcordance: noConcordance,
		expectedError:  errConceptTypeNotAllowed,
	}
	handlesMultipleFactsetIds := testStruct{
		testName:       "handlesMultipleFactsetIds",
		pathToFile:     "../resources/multipleFactsetIds.json",
		conceptUuid:    testUuid,
		uppConcordance: multiFactsetConcordance,
		expectedError:  nil,
	}
	handlesNoFactsetIds := testStruct{
		testName:       "handlesNoFactsetIds",
		pathToFile:     "../resources/noFactsetIds.json",
		conceptUuid:    testUuid,
		uppConcordance: emptyConcordance,
		expectedError:  nil,
	}

	testScenarios := []testStruct{
		missingRequiredFieldsJson,
		invalidTmeListInputJson,
		invalidIdFieldJson,
		missingTypesField,
		membershipNoConcordanceNoError,
		errorOnMembershipConcept,
		errorOnMembershipRoleConcept,
		invalidTmeId,
		errorOnDuplicateTmeIds,
		handlesMultipleTmeIds,
		handlesNoTmeIds,
		tmeGeneratedUuidEqualConceptUuid,
		invalidFactsetId,
		errorOnDuplicateFactsetIds,
		handlesMultipleFactsetIds,
		handlesNoFactsetIds,
		noErrorOnNotAllowedConceptType,
		managedLocationIds,
		managedLocationDuplicateIds,
		managedLocationBlankId,
		managedLocationMutuallyExclusiveFields,
		editorialBlankId,
		editorialDuplicateIds,
		editorialAndManagedLocationWikidata,
		editorialTwoWikidataIds,
	}

	for _, scenario := range testScenarios {
		var smartLogicConcept = SmartlogicConcept{}
		decoder := json.NewDecoder(bytes.NewBufferString(readFile(t, scenario.pathToFile)))
		err := decoder.Decode(&smartLogicConcept)
		_, uuid, uppConconcordance, err := convertToUppConcordance(smartLogicConcept, "transaction_id")
		assert.Equal(t, scenario.conceptUuid, uuid, "Scenario: "+scenario.testName+" failed")
		assert.Equal(t, scenario.uppConcordance, uppConconcordance, "Scenario: "+scenario.testName+" failed. Json output does not match")
		if scenario.expectedError != nil {
			assert.Error(t, err, "Scenario: "+scenario.testName+"should have returned error")
			assert.Contains(t, err.Error(), scenario.expectedError.Error(), "Scenario: "+scenario.testName+" returned unexpected output")
		} else {
			assert.Equal(t, scenario.expectedError, err, "Scenario: "+scenario.testName+" failed")
		}
	}
}

func readFile(t *testing.T, fileName string) string {
	fullMessage, err := ioutil.ReadFile(fileName)
	assert.NoError(t, err, "Error reading file ")
	return string(fullMessage)
}
