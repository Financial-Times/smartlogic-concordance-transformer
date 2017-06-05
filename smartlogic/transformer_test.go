package smartlogic

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"errors"
	"encoding/json"
	"io/ioutil"
)

var testUuid = "20db1bd6-59f9-4404-adb5-3165a448f8b0"
var concordedId = "d83a4dc1-397e-4f99-8ecf-2f1b15febb7f"
var writerUrl = "http://localhost:8080/"

func TestValidateSubstrings(t *testing.T) {
	type testStruct struct {
		testName string
		tmeIdParts []string
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
		assert.Equal(t, scenario.expectedResult, substringsAreValid, "Scenario: " + scenario.testName + " failed")
	}
}

func TestValidateIdAndConvertToUuid(t *testing.T) {
	type testStruct struct {
		testName string
		tmeId string
		expectedUuid string
		expectedError error
	}

	invalidTmeIdHasNoHyphen := testStruct{testName: "invalidTmeId", tmeId: "YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJjYTE5NDEyM2Yw", expectedUuid: "", expectedError: errors.New("YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJjYTE5NDEyM2Yw is not a valid TME Id")}
	invalidTmeIdHasNoTaxonomy := testStruct{testName: "invalidTmeIdHasNoTaxonomy", tmeId: "YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNm-", expectedUuid: "", expectedError: errors.New("YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNm- is not a valid TME Id")}
	invalidTmeIdHasNoValue := testStruct{testName: "invalidTmeIdHasNoValue", tmeId: "-JjYTE5NDEyM2Yw", expectedUuid: "", expectedError: errors.New("-JjYTE5NDEyM2Yw is not a valid TME Id")}
	invalidTmeIdHasTooManyParts := testStruct{testName: "invalidTmeIdHasTooManyParts", tmeId: "YzhlNzZkYTctMDJi-Ny00NTViLTk3NmYtNm-JjYTE5NDEyM2Yw", expectedUuid: "", expectedError: errors.New("YzhlNzZkYTctMDJi-Ny00NTViLTk3NmYtNm-JjYTE5NDEyM2Yw is not a valid TME Id")}
	validTmeIdIsConverted := testStruct{testName: "validTmeIdIsConverted", tmeId: "YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ-jYTE5NDEyM2Yw", expectedUuid: "a50ffd61-e9da-3c71-85ad-81ce983bcbf6", expectedError: nil}

	testScenarios := []testStruct{invalidTmeIdHasNoHyphen, invalidTmeIdHasNoTaxonomy, invalidTmeIdHasNoValue, invalidTmeIdHasTooManyParts, validTmeIdIsConverted}

	for _, scenario := range testScenarios {
		uuid, err := validateIdAndConvertToUuid(scenario.tmeId)
		assert.Equal(t, scenario.expectedUuid, uuid, "Scenario: " + scenario.testName + " failed")
		assert.Equal(t, scenario.expectedError, err, "Scenario: " + scenario.testName + " failed")
	}
}

func TestExtractUuid(t *testing.T) {
	type testStruct struct {
		testName string
		url string
		expectedResult string
	}

	invalidUrlMissingFtPrefix := testStruct{testName: "invalidUrlMissingFtPrefix", url: "www.google.com/2d3e16e0-61cb-4322-8aff-3b01c59f4daa", expectedResult: ""}
	invalidUrlWithInvalidUuid := testStruct{testName: "invalidUrlWithInvalidUuid", url: "http://www.ft.com/thing/2d3e16e061cb43228aff3b01c59f4daa", expectedResult: ""}
	ValidUrlIsConvertedToUuid:= testStruct{testName: "ValidUrlIsConvertedToUuid", url: "http://www.ft.com/thing/2d3e16e0-61cb-4322-8aff-3b01c59f4daa", expectedResult: "2d3e16e0-61cb-4322-8aff-3b01c59f4daa"}

	testScenarios := []testStruct{invalidUrlMissingFtPrefix, invalidUrlWithInvalidUuid, ValidUrlIsConvertedToUuid}

	for _, scenario := range testScenarios {
		uuid := extractUuid(scenario.url)
		assert.Equal(t, scenario.expectedResult, uuid, "Scenario: " + scenario.testName + " failed")
	}
}

func TestMakeRelevantRequest(t *testing.T) {
	uppConcordance := UppConcordance{ConceptUuid: testUuid, ConcordedIds: []string{concordedId}}
	jsonWithConcordance, _ := json.Marshal(uppConcordance)
	noConcordance := UppConcordance{ConceptUuid: testUuid, ConcordedIds: make([]string, 0)}
	jsonWithNoConcordance, _ := json.Marshal(noConcordance)
	type testStruct struct {
		testName string
		uuid string
		concordanceFound bool
		concordedJson []byte
		expectedError error
		clientResp string
		statusCode int
		clientErr error
	}

	concordanceFound_WriteError := testStruct{testName: "concordanceFound_WriteError", uuid: testUuid, concordanceFound: true, concordedJson: jsonWithConcordance, expectedError: errors.New("Get request to resulted in error"), clientResp: "", statusCode: 200, clientErr: errors.New("Error")}
	concordanceFound_Status503 := testStruct{testName: "concordanceFound_Status503", uuid: testUuid, concordanceFound: true, concordedJson: jsonWithConcordance, expectedError: errors.New("Get request to returned status"), clientResp: "",  statusCode: 503, clientErr: nil}
	noConcordance_SuccessfulWrite := testStruct{testName: "noConcordance_SuccessfulWrite", uuid: testUuid, concordanceFound: true, concordedJson: jsonWithConcordance, expectedError: nil, clientResp: "",  statusCode: 200, clientErr: nil}
	noConcordance_WriteError := testStruct{testName: "noConcordance_WriteError", uuid: testUuid, concordanceFound: false, concordedJson: jsonWithNoConcordance, expectedError: errors.New("Delete request to resulted in error"), clientResp: "",  statusCode: 200, clientErr: errors.New("Error")}
	noConcordance_Status503 := testStruct{testName: "noConcordance_Status503", uuid: testUuid, concordanceFound: false, concordedJson: jsonWithNoConcordance, expectedError: errors.New("Delete request to returned status"), clientResp: "",  statusCode: 503, clientErr: nil}
	noConcordance_SuccessfulDelete := testStruct{testName: "noConcordance_SuccessfulDelete", uuid: testUuid, concordanceFound: false, concordedJson: jsonWithNoConcordance, expectedError: nil, clientResp: "",  statusCode: 404, clientErr: nil}

	testScenarios := []testStruct{concordanceFound_WriteError, concordanceFound_Status503, noConcordance_SuccessfulWrite, noConcordance_WriteError, noConcordance_Status503, noConcordance_SuccessfulDelete}

	for _, scenario := range testScenarios {
		ts := NewTransformerService("", writerUrl, mockHttpClient{resp: scenario.clientResp, statusCode: scenario.statusCode, err: scenario.clientErr})
		reqErr := ts.makeRelevantRequest(scenario.uuid, scenario.concordanceFound, scenario.concordedJson, "")
		if reqErr != nil {
			assert.Contains(t, reqErr.Error(), scenario.expectedError.Error(), "Scenario: " + scenario.testName + " failed")
		} else {
			assert.Equal(t, reqErr, scenario.expectedError, "Scenario: " + scenario.testName + " failed")
		}
	}
}

func TestConvertToUppConcordance(t *testing.T) {
	type testStruct struct {
		testName string
		pathToFile string
		conceptUuid string
		concordanceFound bool
		concordedJson []byte
		expectedError error
	}

	missingRequiredFieldsJson := testStruct{testName: "missingRequiredFieldsJson", pathToFile: "../resources/sourceJson/missingIdField.json", conceptUuid: "", concordanceFound: false, concordedJson: nil, expectedError: errors.New("missing @graph and/or @id fields")}
	invalidTmeListInputJson := testStruct{testName: "invalidTmeListInputJson", pathToFile: "../resources/sourceJson/invalidTmeListInput.json", conceptUuid: "", concordanceFound: false, concordedJson: nil, expectedError: errors.New("contains list of tmeIds in wrong format; please input values individually")}
	invalidMessageJson := testStruct{testName: "invalidMessageJson", pathToFile: "../resources/sourceJson/invalid.json", conceptUuid: "", concordanceFound: false, concordedJson: nil, expectedError: errors.New("invalid character")}
	invalidIdFieldJson := testStruct{testName: "invalidIdFieldJson", pathToFile: "../resources/sourceJson/invalidIdValue.json", conceptUuid: "", concordanceFound: false, concordedJson: nil, expectedError: errors.New("has invalid/missing url")}
	invalidTmeId := testStruct{testName: "invalidTmeId", pathToFile: "../resources/sourceJson/invalidTmeId.json", conceptUuid: "", concordanceFound: false, concordedJson: nil, expectedError: errors.New("is not a valid TME Id")}
	filtersDuplicateTmeIds := testStruct{testName: "filtersDuplicateTmeIds", pathToFile: "../resources/sourceJson/duplicateTmeIds.json", conceptUuid: testUuid, concordanceFound: true, concordedJson: []byte(readFile(t, "../resources/concordedJson/oneConcordance.json")), expectedError: nil}
	handlesMultipleTmeIds := testStruct{testName: "handlesMultipleTmeIds", pathToFile: "../resources/sourceJson/multipleTmeIds.json", conceptUuid: testUuid, concordanceFound: true, concordedJson: []byte(readFile(t, "../resources/concordedJson/multipleConcordance.json")), expectedError: nil}
	handlesNoTmeIds := testStruct{testName: "handlesNoTmeIds", pathToFile: "../resources/sourceJson/noTmeIds.json", conceptUuid: testUuid, concordanceFound: false, concordedJson: []byte(readFile(t, "../resources/concordedJson/noConcordance.json")), expectedError: nil}

	testScenarios := []testStruct{missingRequiredFieldsJson, invalidTmeListInputJson, invalidMessageJson, invalidIdFieldJson, invalidTmeId, filtersDuplicateTmeIds, handlesMultipleTmeIds, handlesNoTmeIds}

	for _, scenario := range testScenarios {
		uuid, concordanceFound, concordedJson, err := convertToUppConcordance(readFile(t, scenario.pathToFile))
		assert.Equal(t, scenario.conceptUuid, uuid, "Scenario: " + scenario.testName + " failed")
		assert.Equal(t, scenario.concordanceFound, concordanceFound, "Scenario: " + scenario.testName + " failed. No concordances found")
		assert.Equal(t, scenario.concordedJson, concordedJson, "Scenario: " + scenario.testName + " failed. Json output does not match")
		if scenario.expectedError != nil {
			assert.Error(t, err, "Scenario: " + scenario.testName + "should have returned error")
			assert.Contains(t, err.Error(), scenario.expectedError.Error(), "Scenario: " + scenario.testName + " returned unexpected output")
		} else {
			assert.Equal(t, err, scenario.expectedError, "Scenario: " + scenario.testName + " failed")
		}
	}
}

func readFile(t *testing.T, fileName string) string {
	fullMessage, err := ioutil.ReadFile(fileName)
	assert.NoError(t, err, "Error reading file ")
	return string(fullMessage)
}