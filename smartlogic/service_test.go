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
	testUUID         = "20db1bd6-59f9-4404-adb5-3165a448f8b0"
	concordedTmeUUID = "d83a4dc1-397e-4f99-8ecf-2f1b15febb7f"
	writerURL        = "http://localhost:8080/"
)

var (
	concordedTmeID = ConcordedID{
		Authority: ConcordanceAuthorityTme,
		UUID:      concordedTmeUUID,
	}
)

func TestValidateSubstrings(t *testing.T) {
	type testStruct struct {
		testName       string
		tmeIDParts     []string
		expectedResult bool
	}

	oneValidSubstring := testStruct{testName: "oneValidSubstring", tmeIDParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ"}, expectedResult: true}
	twoValidSubstring := testStruct{testName: "twoValidSubstring", tmeIDParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ", "jYTE5NDEyM2Yw"}, expectedResult: true}
	thirdSubstringIsEmpty := testStruct{testName: "thirdSubstringIsEmpty", tmeIDParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ", "jYTE5NDEyM2Yw", ""}, expectedResult: false}
	secondSubstringIsEmpty := testStruct{testName: "secondSubstringIsEmpty", tmeIDParts: []string{"YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ", ""}, expectedResult: false}
	firstSubstringIsEmpty := testStruct{testName: "firstSubstringIsEmpty", tmeIDParts: []string{"", "jYTE5NDEyM2Yw"}, expectedResult: false}
	onlySubstringIsEmpty := testStruct{testName: "onlySubstringIsEmpty", tmeIDParts: []string{""}, expectedResult: false}

	testScenarios := []testStruct{oneValidSubstring, twoValidSubstring, thirdSubstringIsEmpty, secondSubstringIsEmpty, firstSubstringIsEmpty, onlySubstringIsEmpty}

	for _, scenario := range testScenarios {
		substringsAreValid := validateSubstrings(scenario.tmeIDParts)
		assert.Equal(t, scenario.expectedResult, substringsAreValid, "Scenario: "+scenario.testName+" failed")
	}
}

func TestValidateTmeIdAndConvertToUuid(t *testing.T) {
	type testStruct struct {
		testName      string
		tmeID         string
		expectedUUID  string
		expectedError error
	}

	invalidTmeIDHasNoHyphen := testStruct{testName: "invalidTmeId", tmeID: "YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJjYTE5NDEyM2Yw", expectedUUID: "", expectedError: errors.New("Bad Request: Concordance id YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJjYTE5NDEyM2Yw is not a valid TME Id")}
	invalidTmeIDHasNoTaxonomy := testStruct{testName: "invalidTmeIdHasNoTaxonomy", tmeID: "YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNm-", expectedUUID: "", expectedError: errors.New("Bad Request: Concordance id YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNm- is not a valid TME Id")}
	invalidTmeIDHasNoValue := testStruct{testName: "invalidTmeIdHasNoValue", tmeID: "-JjYTE5NDEyM2Yw", expectedUUID: "", expectedError: errors.New("Bad Request: Concordance id -JjYTE5NDEyM2Yw is not a valid TME Id")}
	invalidTmeIDHasTooManyParts := testStruct{testName: "invalidTmeIdHasTooManyParts", tmeID: "YzhlNzZkYTctMDJi-Ny00NTViLTk3NmYtNm-JjYTE5NDEyM2Yw", expectedUUID: "", expectedError: errors.New("Bad Request: Concordance id YzhlNzZkYTctMDJi-Ny00NTViLTk3NmYtNm-JjYTE5NDEyM2Yw is not a valid TME Id")}
	validTmeIDIsConverted := testStruct{testName: "validTmeIdIsConverted", tmeID: "YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJ-jYTE5NDEyM2Yw", expectedUUID: "a50ffd61-e9da-3c71-85ad-81ce983bcbf6", expectedError: nil}

	testScenarios := []testStruct{invalidTmeIDHasNoHyphen, invalidTmeIDHasNoTaxonomy, invalidTmeIDHasNoValue, invalidTmeIDHasTooManyParts, validTmeIDIsConverted}

	for _, scenario := range testScenarios {
		uuid, err := validateTmeIDAndConvertToUUID(scenario.tmeID)
		assert.Equal(t, scenario.expectedUUID, uuid, "Scenario: "+scenario.testName+" failed")
		assert.Equal(t, scenario.expectedError, err, "Scenario: "+scenario.testName+" failed")
	}
}

func TestValidateFactsetIdAndConvertToUuid(t *testing.T) {
	type testStruct struct {
		testName      string
		factsetID     string
		expectedUUID  string
		expectedError error
	}

	invalidFactsetIDNoZeroPrefix := testStruct{testName: "invalidFactsetIdNoZeroPrefix", factsetID: "123456-E", expectedUUID: "", expectedError: errors.New("Bad Request: Concordance id 123456-E is not a valid FACTSET Id")}
	invalidFactsetINoESuffix := testStruct{testName: "invalidFactsetINoESuffix", factsetID: "023456-A", expectedUUID: "", expectedError: errors.New("Bad Request: Concordance id 023456-A is not a valid FACTSET Id")}
	invalidFactsetIDNoHyphenSuffix := testStruct{testName: "invalidFactsetIdNoHyphenSuffix", factsetID: "0123456E", expectedUUID: "", expectedError: errors.New("Bad Request: Concordance id 0123456E is not a valid FACTSET Id")}
	validFactsetIDIsConverted := testStruct{testName: "validFactsetIdIsConverted", factsetID: "012345-E", expectedUUID: "949a7e7f-2516-30c0-9123-f866601ffbe4", expectedError: nil}

	testScenarios := []testStruct{invalidFactsetIDNoZeroPrefix, invalidFactsetINoESuffix, invalidFactsetIDNoHyphenSuffix, validFactsetIDIsConverted}

	for _, scenario := range testScenarios {
		uuid, err := validateFactsetIDAndConvertToUUID(scenario.factsetID)
		assert.Equal(t, scenario.expectedUUID, uuid, "Scenario: "+scenario.testName+" failed")
		assert.Equal(t, scenario.expectedError, err, "Scenario: "+scenario.testName+" failed")
	}
}

func TestExtractUuidAndConcordanceAuthority(t *testing.T) {
	type testStruct struct {
		testName       string
		url            string
		expectedResult string
	}

	invalidURLMissingFtPrefix := testStruct{testName: "invalidUrlMissingFtPrefix", url: "www.google.com/2d3e16e0-61cb-4322-8aff-3b01c59f4daa", expectedResult: ""}
	invalidURLWithInvalidUUID := testStruct{testName: "invalidUrlWithInvalidUuid", url: "http://www.ft.com/thing/2d3e16e061cb43228aff3b01c59f4daa", expectedResult: ""}
	ValidURLIsConvertedToUUID := testStruct{testName: "ValidUrlIsConvertedToUuid", url: "http://www.ft.com/thing/2d3e16e0-61cb-4322-8aff-3b01c59f4daa", expectedResult: "2d3e16e0-61cb-4322-8aff-3b01c59f4daa"}

	testScenarios := []testStruct{invalidURLMissingFtPrefix, invalidURLWithInvalidUUID, ValidURLIsConvertedToUUID}

	for _, scenario := range testScenarios {
		uuid, _ := extractUUIDAndConcordanceAuthority(scenario.url)
		assert.Equal(t, scenario.expectedResult, uuid, "Scenario: "+scenario.testName+" failed")
	}
}

func TestMakeRelevantRequest(t *testing.T) {
	withConcordance := UppConcordance{ConceptUUID: testUUID, ConcordedIds: []ConcordedID{concordedTmeID}}
	noConcordance := UppConcordance{ConceptUUID: testUUID, ConcordedIds: []ConcordedID{}}
	type testStruct struct {
		testName       string
		uuid           string
		uppConcordance UppConcordance
		expectedError  error
		clientResp     string
		statusCode     int
		clientErr      error
	}

	concordancefoundWriteerror := testStruct{testName: "concordanceFound_WriteError", uuid: testUUID, uppConcordance: withConcordance, expectedError: errors.New("get request resulted in error"), clientResp: "", statusCode: 200, clientErr: errors.New("get request resulted in error")}
	concordancefoundStatus503 := testStruct{testName: "concordanceFound_Status503", uuid: testUUID, uppConcordance: withConcordance, expectedError: errors.New("Internal Error: Get request to writer returned unexpected status"), clientResp: "", statusCode: 503, clientErr: nil}
	noconcordanceSuccessfulwrite := testStruct{testName: "noConcordance_SuccessfulWrite", uuid: testUUID, uppConcordance: withConcordance, expectedError: nil, clientResp: "", statusCode: 200, clientErr: nil}
	noconcordanceWriteerror := testStruct{testName: "noConcordance_WriteError", uuid: testUUID, uppConcordance: noConcordance, expectedError: errors.New("delete request resulted in error"), clientResp: "", statusCode: 200, clientErr: errors.New("delete request resulted in error")}
	noconcordanceStatus503 := testStruct{testName: "noConcordance_Status503", uuid: testUUID, uppConcordance: noConcordance, expectedError: errors.New("Internal Error: Delete request to writer returned unexpected status"), clientResp: "", statusCode: 503, clientErr: nil}
	noconcordanceRecordnotfound := testStruct{testName: "noConcordance_RecordNotFound", uuid: testUUID, uppConcordance: noConcordance, expectedError: nil, clientResp: "", statusCode: 204, clientErr: nil}
	noconcordanceSuccessfuldelete := testStruct{testName: "noConcordance_SuccessfulDelete", uuid: testUUID, uppConcordance: noConcordance, expectedError: nil, clientResp: "", statusCode: 404, clientErr: nil}

	testScenarios := []testStruct{concordancefoundWriteerror, concordancefoundStatus503, noconcordanceSuccessfulwrite, noconcordanceWriteerror, noconcordanceStatus503, noconcordanceSuccessfuldelete, noconcordanceRecordnotfound}

	for _, scenario := range testScenarios {
		ts := NewTransformerService("", writerURL, mockHTTPClient{resp: scenario.clientResp, statusCode: scenario.statusCode, err: scenario.clientErr}, createLogger())
		_, reqErr := ts.makeRelevantRequest(scenario.uuid, scenario.uppConcordance, "")
		if reqErr != nil {
			assert.Contains(t, reqErr.Error(), scenario.expectedError.Error(), "Scenario: "+scenario.testName+" failed")
		} else {
			assert.Equal(t, reqErr, scenario.expectedError, "Scenario: "+scenario.testName+" failed")
		}
	}
}

func TestConvertToUppConcordance(t *testing.T) {
	noConcordance := UppConcordance{ConceptUUID: ""}
	emptyConcordance := UppConcordance{
		ConceptUUID:  testUUID,
		ConcordedIds: []ConcordedID{},
		Authority:    "Smartlogic",
	}
	multiConcordance := UppConcordance{
		ConceptUUID: testUUID,
		Authority:   "Smartlogic",
		ConcordedIds: []ConcordedID{
			{
				Authority:      ConcordanceAuthorityTme,
				AuthorityValue: "AbCdEfgHiJkLMnOpQrStUvWxYz-0123456789",
				UUID:           "e9f4525a-401f-3b23-a68e-e48f314cdce6",
			}, {
				Authority:      ConcordanceAuthorityTme,
				AuthorityValue: "ZyXwVuTsRqPoNmLkJiHgFeDcBa-0987654321",
				UUID:           "83f63c7e-1641-3c7b-81e4-378ae3c6c2ad",
			}, {
				Authority:      ConcordanceAuthorityTme,
				AuthorityValue: "abcdefghijklmnopqrstuvwxyz-0123456789",
				UUID:           "e4bc4ac2-0637-3a27-86b1-9589fca6bf2c",
			}, {
				Authority:      ConcordanceAuthorityTme,
				AuthorityValue: "ABCDEFGHIJKLMNOPQRSTUVWXYZ-0987654321",
				UUID:           "e574b21d-9abc-3d82-a6c0-3e08c85181bf",
			},
		},
	}
	multiFactsetConcordance := UppConcordance{
		ConceptUUID: testUUID,
		Authority:   "Smartlogic",
		ConcordedIds: []ConcordedID{
			{
				Authority:      ConcordanceAuthorityFactset,
				AuthorityValue: "000D63-E",
				UUID:           "8d3aba95-02d9-3802-afc0-b99bb9b1139e",
			}, {
				Authority:      ConcordanceAuthorityFactset,
				AuthorityValue: "023456-E",
				UUID:           "3bc0ab41-c01f-3a0b-aa78-c76438080b52",
			}, {
				Authority:      ConcordanceAuthorityFactset,
				AuthorityValue: "023411-E",
				UUID:           "f777c5af-e0b2-34dc-9102-e346ca2d27aa",
			},
		},
	}
	multiTmeFactsetConcordance := UppConcordance{
		ConceptUUID: testUUID,
		Authority:   "ManagedLocation",
		ConcordedIds: []ConcordedID{
			{
				Authority:      ConcordanceAuthorityTme,
				AuthorityValue: "ZyXwVuTsRqPoNmLkJiHgFeDcBa-0987654321",
				UUID:           "83f63c7e-1641-3c7b-81e4-378ae3c6c2ad",
			},
			{
				Authority:      ConcordanceAuthorityFactset,
				AuthorityValue: "023456-E",
				UUID:           "3bc0ab41-c01f-3a0b-aa78-c76438080b52",
			},
		},
	}
	locationsConcordance := UppConcordance{
		ConceptUUID: testUUID,
		Authority:   "ManagedLocation",
		ConcordedIds: []ConcordedID{
			{
				Authority:      ConcordanceAuthorityTme,
				AuthorityValue: "TnN0ZWluX0dMX0dCX0VOR19HX0Vzc2V4-R0w=",
				UUID:           "3f494231-9dc6-3181-8baa-dc9d1cad730f",
			}, {
				Authority:      ConcordanceAuthorityDbpedia,
				AuthorityValue: "http://dbpedia.org/resource/Essex",
				UUID:           "9567fbd6-f6f3-34f4-9b31-53856d5428a3",
			}, {
				Authority:      ConcordanceAuthorityGeonames,
				AuthorityValue: "http://sws.geonames.org/2649889/",
				UUID:           "ed78ef90-a160-30d0-8a3b-472a966c5664",
			}, {
				Authority:      ConcordanceAuthorityWikidata,
				AuthorityValue: "http://www.wikidata.org/entity/Q23240",
				UUID:           "76754d1e-11f6-3d4f-8e3a-59a5b4e6bdcd",
			},
		},
	}

	editorialConcordance := UppConcordance{
		ConceptUUID: testUUID,
		Authority:   "Smartlogic",
		ConcordedIds: []ConcordedID{
			{
				Authority:      ConcordanceAuthorityTme,
				AuthorityValue: "TnN0ZWluX0dMX0dCX0VOR19HX0Vzc2V4-R0w=",
				UUID:           "3f494231-9dc6-3181-8baa-dc9d1cad730f",
			},
			{
				Authority:      ConcordanceAuthorityGeonames,
				AuthorityValue: "http://sws.geonames.org/2649889/",
				UUID:           "ed78ef90-a160-30d0-8a3b-472a966c5664",
			},
			{
				Authority:      ConcordanceAuthorityWikidata,
				AuthorityValue: "http://www.wikidata.org/entity/Q23240",
				UUID:           "76754d1e-11f6-3d4f-8e3a-59a5b4e6bdcd",
			},
		},
	}

	editorialConcordanceTwoWikidata := UppConcordance{
		ConceptUUID: testUUID,
		Authority:   "Smartlogic",
		ConcordedIds: []ConcordedID{
			{
				Authority:      ConcordanceAuthorityTme,
				AuthorityValue: "TnN0ZWluX0dMX0dCX0VOR19HX0Vzc2V4-R0w=",
				UUID:           "3f494231-9dc6-3181-8baa-dc9d1cad730f",
			},
			{
				Authority:      ConcordanceAuthorityGeonames,
				AuthorityValue: "http://sws.geonames.org/2649889/",
				UUID:           "ed78ef90-a160-30d0-8a3b-472a966c5664",
			},
			{
				Authority:      ConcordanceAuthorityWikidata,
				AuthorityValue: "http://www.wikidata.org/entity/Q23240",
				UUID:           "76754d1e-11f6-3d4f-8e3a-59a5b4e6bdcd",
			},
			{
				Authority:      ConcordanceAuthorityWikidata,
				AuthorityValue: "http://www.wikidata.org/entity/Q23245",
				UUID:           "226ee6c7-8e94-3eb8-8370-c89ee9f9f988",
			},
		},
	}

	noWikidataEditorialConcordance := UppConcordance{
		ConceptUUID: testUUID,
		Authority:   "Smartlogic",
		ConcordedIds: []ConcordedID{
			{
				Authority:      ConcordanceAuthorityTme,
				AuthorityValue: "TnN0ZWluX0dMX0dCX0VOR19HX0Vzc2V4-R0w=",
				UUID:           "3f494231-9dc6-3181-8baa-dc9d1cad730f",
			},
		},
	}

	editorialGeonamesConcordance := UppConcordance{
		ConceptUUID: testUUID,
		Authority:   "Smartlogic",
		ConcordedIds: []ConcordedID{
			{
				Authority:      ConcordanceAuthorityTme,
				AuthorityValue: "TnN0ZWluX0dMX0dCX0VOR19HX0Vzc2V4-R0w=",
				UUID:           "3f494231-9dc6-3181-8baa-dc9d1cad730f",
			}, {
				Authority:      ConcordanceAuthorityGeonames,
				AuthorityValue: "http://sws.geonames.org/2649889/",
				UUID:           "ed78ef90-a160-30d0-8a3b-472a966c5664",
			},
			{
				Authority:      ConcordanceAuthorityWikidata,
				AuthorityValue: "http://www.wikidata.org/entity/Q23240",
				UUID:           "76754d1e-11f6-3d4f-8e3a-59a5b4e6bdcd",
			},
		},
	}

	type testStruct struct {
		testName       string
		pathToFile     string
		conceptUUID    string
		uppConcordance UppConcordance
		expectedError  error
	}

	missingRequiredFieldsJSON := testStruct{testName: "missingRequiredFieldsJson", pathToFile: "../resources/missingIdField.json", conceptUUID: "", uppConcordance: noConcordance, expectedError: errors.New("Missing/invalid @graph field")}
	invalidTmeListInputJSON := testStruct{testName: "invalidTmeListInputJson", pathToFile: "../resources/invalidTmeListInput.json", conceptUUID: testUUID, uppConcordance: noConcordance, expectedError: errors.New("is not a valid TME Id")}
	invalidIDFieldJSON := testStruct{testName: "invalidIdFieldJson", pathToFile: "../resources/invalidIdValue.json", conceptUUID: "", uppConcordance: noConcordance, expectedError: errors.New("Missing/invalid @id field")}
	missingTypesField := testStruct{testName: "missingTypesField", pathToFile: "../resources/noTypes.json", conceptUUID: testUUID, uppConcordance: noConcordance, expectedError: errors.New("bad Request: Type has not been set for concept: 20db1bd6-59f9-4404-adb5-3165a448f8b0")}
	membershipNoConcordanceNoError := testStruct{testName: "membershipNoConcordanceNoError", pathToFile: "../resources/conceptIsMembershipNoConcordance.json", conceptUUID: testUUID, uppConcordance: emptyConcordance, expectedError: nil}
	errorOnMembershipConcept := testStruct{testName: "errorOnMembershipConcept", pathToFile: "../resources/conceptIsMembership.json", conceptUUID: testUUID, uppConcordance: noConcordance, expectedError: errors.New("bad Request: Concept type Membership does not support concordance")}
	errorOnMembershipRoleConcept := testStruct{testName: "errorOnMembershipRoleConcept", pathToFile: "../resources/conceptIsMembershipRole.json", conceptUUID: testUUID, uppConcordance: noConcordance, expectedError: errors.New("bad Request: Concept type MembershipRole does not support concordance")}
	invalidTmeID := testStruct{testName: "invalidTmeId", pathToFile: "../resources/invalidTmeId.json", conceptUUID: testUUID, uppConcordance: noConcordance, expectedError: errors.New("is not a valid TME Id")}
	tmeGeneratedUUIDEqualConceptUUID := testStruct{testName: "tmeGeneratedUuidEqualConceptUuid", pathToFile: "../resources/tmeGeneratedUuidEqualConceptUuid.json", conceptUUID: "e9f4525a-401f-3b23-a68e-e48f314cdce6", uppConcordance: noConcordance, expectedError: errors.New("smartlogic uuid that is the same as the uuid generated from the TME id")}
	errorOnDuplicateTmeIds := testStruct{testName: "errorOnDuplicateTmeIds", pathToFile: "../resources/duplicateTmeIds.json", conceptUUID: testUUID, uppConcordance: noConcordance, expectedError: errors.New("contains duplicate TME id values")}
	handlesMultipleTmeIds := testStruct{testName: "handlesMultipleTmeIds", pathToFile: "../resources/multipleTmeIds.json", conceptUUID: testUUID, uppConcordance: multiConcordance, expectedError: nil}
	handlesNoTmeIds := testStruct{testName: "handlesNoTmeIds", pathToFile: "../resources/noTmeIds.json", conceptUUID: testUUID, uppConcordance: emptyConcordance, expectedError: nil}
	managedLocationIds := testStruct{testName: "managedLocationIds", pathToFile: "../resources/managedLocationIds.json", conceptUUID: testUUID, uppConcordance: locationsConcordance, expectedError: nil}
	managedLocationDuplicateIds := testStruct{testName: "managedLocationDuplicateIds", pathToFile: "../resources/managedLocationDuplicateIds.json", conceptUUID: testUUID, uppConcordance: locationsConcordance, expectedError: nil}
	managedLocationBlankID := testStruct{testName: "managedLocationBlankId", pathToFile: "../resources/managedLocationBlankId.json", conceptUUID: testUUID, uppConcordance: locationsConcordance, expectedError: nil}
	managedLocationMutuallyExclusiveFields := testStruct{testName: "managedLocationMutuallyExclusiveFields", pathToFile: "../resources/managedLocationMutuallyExclusiveFields.json", conceptUUID: testUUID, uppConcordance: multiTmeFactsetConcordance, expectedError: nil}
	editorialBlankID := testStruct{testName: "editorialBlankId", pathToFile: "../resources/editorialBlankId.json", conceptUUID: testUUID, uppConcordance: noWikidataEditorialConcordance, expectedError: nil}
	editorialDuplicateIds := testStruct{testName: "editorialDuplicateIds", pathToFile: "../resources/editorialDuplicateIds.json", conceptUUID: testUUID, uppConcordance: editorialConcordance, expectedError: nil}
	editorialAndManagedLocationWikidata := testStruct{testName: "editorialAndManagedLocationWikidata", pathToFile: "../resources/editorialAndManagedLocationWikidata.json", conceptUUID: testUUID, uppConcordance: editorialConcordance, expectedError: nil}
	editorialTwoWikidataIds := testStruct{testName: "editorialTwoWikidataIds", pathToFile: "../resources/editorialTwoWikidata.json", conceptUUID: testUUID, uppConcordance: editorialConcordanceTwoWikidata, expectedError: nil}
	editorialGeonamesID := testStruct{testName: "editorialGeonamesId", pathToFile: "../resources/editorialGeonames.json", conceptUUID: testUUID, uppConcordance: editorialGeonamesConcordance, expectedError: nil}

	invalidFactsetID := testStruct{
		testName:       "invalidFactsetId",
		pathToFile:     "../resources/invalidFactsetId.json",
		conceptUUID:    testUUID,
		uppConcordance: noConcordance,
		expectedError:  errors.New("is not a valid FACTSET Id"),
	}
	errorOnDuplicateFactsetIds := testStruct{
		testName:       "errorOnDuplicateFactsetIds",
		pathToFile:     "../resources/duplicateFactsetIds.json",
		conceptUUID:    testUUID,
		uppConcordance: noConcordance,
		expectedError:  errors.New("contains duplicate FACTSET id values"),
	}
	noErrorOnNotAllowedConceptType := testStruct{
		testName:       "noErrorOnNotAllowedConceptType",
		pathToFile:     "../resources/notAllowedType.json",
		conceptUUID:    testUUID,
		uppConcordance: noConcordance,
		expectedError:  errConceptTypeNotAllowed,
	}
	handlesMultipleFactsetIds := testStruct{
		testName:       "handlesMultipleFactsetIds",
		pathToFile:     "../resources/multipleFactsetIds.json",
		conceptUUID:    testUUID,
		uppConcordance: multiFactsetConcordance,
		expectedError:  nil,
	}
	handlesNoFactsetIds := testStruct{
		testName:       "handlesNoFactsetIds",
		pathToFile:     "../resources/noFactsetIds.json",
		conceptUUID:    testUUID,
		uppConcordance: emptyConcordance,
		expectedError:  nil,
	}

	testScenarios := []testStruct{
		missingRequiredFieldsJSON,
		invalidTmeListInputJSON,
		invalidIDFieldJSON,
		missingTypesField,
		membershipNoConcordanceNoError,
		errorOnMembershipConcept,
		errorOnMembershipRoleConcept,
		invalidTmeID,
		errorOnDuplicateTmeIds,
		handlesMultipleTmeIds,
		handlesNoTmeIds,
		tmeGeneratedUUIDEqualConceptUUID,
		invalidFactsetID,
		errorOnDuplicateFactsetIds,
		handlesMultipleFactsetIds,
		handlesNoFactsetIds,
		noErrorOnNotAllowedConceptType,
		managedLocationIds,
		managedLocationDuplicateIds,
		managedLocationBlankID,
		managedLocationMutuallyExclusiveFields,
		editorialBlankID,
		editorialDuplicateIds,
		editorialAndManagedLocationWikidata,
		editorialTwoWikidataIds,
		editorialGeonamesID,
	}

	for _, scenario := range testScenarios {
		var smartLogicConcept = ConceptData{}
		decoder := json.NewDecoder(bytes.NewBufferString(readFile(t, scenario.pathToFile)))
		err := decoder.Decode(&smartLogicConcept)
		_, uuid, uppConcordance, err := convertToUppConcordance(smartLogicConcept, "transaction_id", createLogger())
		assert.Equal(t, scenario.conceptUUID, uuid, "Scenario: "+scenario.testName+" failed")
		assert.Equal(t, scenario.uppConcordance, uppConcordance, "Scenario: "+scenario.testName+" failed. Json output does not match")
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

func TestTransformerService_handleConcordanceEvent(t *testing.T) {
	mockClient := mockHTTPClient{resp: "", statusCode: 200}
	defaultTransformer := NewTransformerService(TOPIC, WriterAddress, &mockClient, createLogger())

	type testStruct struct {
		scenarioName  string
		payload       string
		expectedError error
	}

	invalidJSONLd := `{"@graph": [{"@id": "http://www.ft.com/thing/20db1bd6-59f9-4404-adb5-3165a448f8b0"}, {"@id": "http://www.ft.com/thing/20db1bd6-59f9-4404-adb5-3165a448f8b0"}]}`
	validJSONLdNoConcordance := `{"@graph": [{"@id": "http://www.ft.com/thing/20db1bd6-59f9-4404-adb5-3165a448f8b0", "@type": ["http://www.ft.com/ontology/Brand"]}]}`
	validJSONLdWithConcordance := `{"@graph": [{"@id": "http://www.ft.com/thing/20db1bd6-59f9-4404-adb5-3165a448f8b0", "@type": ["http://www.ft.com/ontology/Brand"], "http://www.ft.com/ontology/TMEIdentifier": [{"@value": "AbCdEfgHiJkLMnOpQrStUvWxYz-0123456789"}]}]}`

	failOnInvalidKafkaMessagePayload := testStruct{scenarioName: "failOnInvalidKafkaMessagePayload", payload: "", expectedError: errors.New("EOF")}
	failOnInvalidJSONLdInPayload := testStruct{scenarioName: "failOnInvalidJsonLdInPayload", payload: invalidJSONLd, expectedError: errors.New("invalid Request Json: More than 1 concept in smartlogic concept payload which is currently not supported")}
	failOnWritePayloadToWriter := testStruct{scenarioName: "failOnWritePayloadToWriter", payload: validJSONLdNoConcordance, expectedError: errors.New("Internal Error: Delete request to writer returned unexpected status: 200")}
	successfulRequest := testStruct{scenarioName: "successfulRequest", payload: validJSONLdWithConcordance, expectedError: nil}

	scenarios := []testStruct{failOnInvalidKafkaMessagePayload, failOnInvalidJSONLdInPayload, failOnWritePayloadToWriter, successfulRequest}
	for _, scenario := range scenarios {
		err := defaultTransformer.handleConcordanceEvent(scenario.payload, "test-tid")
		assert.Equal(t, scenario.expectedError, err, "Scenario "+scenario.scenarioName+" failed with unexpected error")
	}
}
