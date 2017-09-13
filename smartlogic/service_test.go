package smartlogic

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testUuid = "20db1bd6-59f9-4404-adb5-3165a448f8b0"
var concordedId = "d83a4dc1-397e-4f99-8ecf-2f1b15febb7f"
var writerUrl = "http://localhost:8080/"

func TestValidateSubstrings(t *testing.T) {
	type testStruct struct {
		testName       string
		tmeIdParts     []string
		expectedResult bool
	}

	oneValidSubstring := testStruct{testName: "oneValidSubstring", tmeIdParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ"}, expectedResult: false}
	twoValidSubstring := testStruct{testName: "twoValidSubstring", tmeIdParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ", "jYTE5NDEyM2Yw"}, expectedResult: false}
	thirdSubstringIsEmpty := testStruct{testName: "thirdSubstringIsEmpty", tmeIdParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ", "jYTE5NDEyM2Yw", ""}, expectedResult: true}
	secondSubstringIsEmpty := testStruct{testName: "secondSubstringIsEmpty", tmeIdParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ", ""}, expectedResult: true}
	firstSubstringIsEmpty := testStruct{testName: "firstSubstringIsEmpty", tmeIdParts: []string{"", "jYTE5NDEyM2Yw"}, expectedResult: true}
	onlySubstringIsEmpty := testStruct{testName: "onlySubstringIsEmpty", tmeIdParts: []string{""}, expectedResult: true}

	testScenarios := []testStruct{oneValidSubstring, twoValidSubstring, thirdSubstringIsEmpty, secondSubstringIsEmpty, firstSubstringIsEmpty, onlySubstringIsEmpty}

	for _, scenario := range testScenarios {
		substringsAreValid := validateSubstrings(scenario.tmeIdParts)
		assert.Equal(t, scenario.expectedResult, substringsAreValid, "Scenario: "+scenario.testName+" failed")
	}
}

func TestValidateIdAndConvertToUuid(t *testing.T) {
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
		uuid, err := validateIdAndConvertToUuid(scenario.tmeId)
		assert.Equal(t, scenario.expectedUuid, uuid, "Scenario: "+scenario.testName+" failed")
		assert.Equal(t, scenario.expectedError, err, "Scenario: "+scenario.testName+" failed")
	}
}

func TestExtractUuid(t *testing.T) {
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
		uuid := extractUuid(scenario.url)
		assert.Equal(t, scenario.expectedResult, uuid, "Scenario: "+scenario.testName+" failed")
	}
}

func TestMakeRelevantRequest(t *testing.T) {
	withConcordance := UppConcordance{ConceptUuid: testUuid, ConcordedIds: []string{concordedId}}
	noConcordance := UppConcordance{ConceptUuid: testUuid, ConcordedIds: make([]string, 0)}
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
	emptyConcordance := UppConcordance{ConceptUuid: testUuid, ConcordedIds: make([]string, 0)}
	multiConcordance := UppConcordance{ConceptUuid: testUuid, ConcordedIds: []string{"e9f4525a-401f-3b23-a68e-e48f314cdce6", "83f63c7e-1641-3c7b-81e4-378ae3c6c2ad", "e4bc4ac2-0637-3a27-86b1-9589fca6bf2c", "e574b21d-9abc-3d82-a6c0-3e08c85181bf"}}

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
	invalidTmeId := testStruct{testName: "invalidTmeId", pathToFile: "../resources/invalidTmeId.json", conceptUuid: testUuid, uppConcordance: noConcordance, expectedError: errors.New("is not a valid TME Id")}
	tmeGeneratedUuidEqualConceptUuid := testStruct{testName: "tmeGeneratedUuidEqualConceptUuid", pathToFile: "../resources/tmeGeneratedUuidEqualConceptUuid.json", conceptUuid: "e9f4525a-401f-3b23-a68e-e48f314cdce6", uppConcordance: noConcordance, expectedError: errors.New("smartlogic uuid that is the same as the uuid generated from the TME id")}
	errorOnDuplicateTmeIds := testStruct{testName: "errorOnDuplicateTmeIds", pathToFile: "../resources/duplicateTmeIds.json", conceptUuid: testUuid, uppConcordance: noConcordance, expectedError: errors.New("contains duplicate TME id values")}
	handlesMultipleTmeIds := testStruct{testName: "handlesMultipleTmeIds", pathToFile: "../resources/multipleTmeIds.json", conceptUuid: testUuid, uppConcordance: multiConcordance, expectedError: nil}
	handlesNoTmeIds := testStruct{testName: "handlesNoTmeIds", pathToFile: "../resources/noTmeIds.json", conceptUuid: testUuid, uppConcordance: emptyConcordance, expectedError: nil}

	testScenarios := []testStruct{missingRequiredFieldsJson, invalidTmeListInputJson, invalidIdFieldJson, invalidTmeId, errorOnDuplicateTmeIds, handlesMultipleTmeIds, handlesNoTmeIds, tmeGeneratedUuidEqualConceptUuid}

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
			assert.Equal(t, err, scenario.expectedError, "Scenario: "+scenario.testName+" failed")
		}
	}
}

func readFile(t *testing.T, fileName string) string {
	fullMessage, err := ioutil.ReadFile(fileName)
	assert.NoError(t, err, "Error reading file ")
	return string(fullMessage)
}
