package smartlogic

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/kafka-client-go/kafka"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

const (
	ExpectedContentType = "application/json"
	WRITER_ADDRESS      = "http://localhost:8080/__concordance-rw-dynamodb/"
	TOPIC               = "TestTopic"
)

type mockHttpClient struct {
	resp       string
	statusCode int
	err        error
}

func TestAdminHandler_Healthy(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHttpClient{resp: "", statusCode: 200}
	defaultTransformer := NewTransformerService(TOPIC, WRITER_ADDRESS, &mockClient)
	h := NewHandler(defaultTransformer, mockConsumer{})
	h.RegisterAdminHandlers(r)

	type testStruct struct {
		endpoint           string
		expectedStatusCode int
		expectedBody       string
		expectedError      string
	}

	pingChecker := testStruct{endpoint: "/__ping", expectedStatusCode: 200, expectedBody: "pong"}
	buildInfoChecker := testStruct{endpoint: "/__build-info", expectedStatusCode: 200, expectedBody: "Version  is not a semantic version"}
	gtgChecker := testStruct{endpoint: "/__gtg", expectedStatusCode: 200, expectedBody: ""}
	healthChecker := testStruct{endpoint: "/__health", expectedStatusCode: 200, expectedBody: ""}

	testScenarios := []testStruct{pingChecker, buildInfoChecker, gtgChecker, healthChecker}

	for _, scenario := range testScenarios {
		rec := httptest.NewRecorder()
		http.DefaultServeMux.ServeHTTP(rec, newRequest("GET", scenario.endpoint, ""))
		assert.Equal(t, scenario.expectedStatusCode, rec.Code)
		assert.Contains(t, rec.Body.String(), scenario.expectedBody)
	}

}

func TestTransformAndSendHandlers(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHttpClient{resp: "", statusCode: 200}
	defaultTransformer := NewTransformerService(TOPIC, WRITER_ADDRESS, &mockClient)
	h := NewHandler(defaultTransformer, mockConsumer{})
	h.RegisterHandlers(r)

	type testStruct struct {
		scenarioName       string
		filePath           string
		endpoint           string
		expectedStatusCode int
		expectedResult     string
	}

	transform_unprocessibleEntityError := testStruct{scenarioName: "transform_unprocessibleEntityError", filePath: "../resources/sourceJson/multipleGraphsInList.json", endpoint: "/transform", expectedStatusCode: 422, expectedResult: "Invalid Request Json: More than 1 concept in smartlogic concept payload which is currently not supported"}
	transform_convertingToConcordedJsonError := testStruct{scenarioName: "transform_convertingToConcordedJsonError", filePath: "../resources/sourceJson/invalidTmeId.json", endpoint: "/transform", expectedStatusCode: 400, expectedResult: "is not a valid TME Id"}
	transform_duplicateTmeIdsError := testStruct{scenarioName: "transform_duplicateTmeIdsError", filePath: "../resources/sourceJson/duplicateTmeIds.json", endpoint: "/transform", expectedStatusCode: 400, expectedResult: "contains duplicate TME id values"}
	transform_convertsAndReturnsPayload := testStruct{scenarioName: "transform_convertsAndReturnsPayload", filePath: "../resources/sourceJson/multipleTmeIds.json", endpoint: "/transform", expectedStatusCode: 200, expectedResult: "{\"uuid\":\"20db1bd6-59f9-4404-adb5-3165a448f8b0\",\"concordedIds\":[\"e9f4525a-401f-3b23-a68e-e48f314cdce6\",\"83f63c7e-1641-3c7b-81e4-378ae3c6c2ad\",\"e4bc4ac2-0637-3a27-86b1-9589fca6bf2c\",\"e574b21d-9abc-3d82-a6c0-3e08c85181bf\"]}"}
	send_unprocessibleEntityError := testStruct{scenarioName: "send_unprocessibleEntityError", filePath: "../resources/sourceJson/multipleGraphsInList.json", endpoint: "/transform/send", expectedStatusCode: 422, expectedResult: "Invalid Request Json: More than 1 concept in smartlogic concept payload which is currently not supported"}
	send_convertingToConcordedJsonError := testStruct{scenarioName: "send_convertingToConcordedJsonError", filePath: "../resources/sourceJson/invalidTmeId.json", endpoint: "/transform/send", expectedStatusCode: 400, expectedResult: "is not a valid TME Id"}
	send_convertsAndForwardsPayloadWithConcordance := testStruct{scenarioName: "send_convertsAndForwardsPayloadWithConcordance", filePath: "../resources/sourceJson/multipleTmeIds.json", endpoint: "/transform/send", expectedStatusCode: 200, expectedResult: "Concordance record for " + testUuid + " forwarded to writer"}
	send_convertsAndFailsForwardToRw := testStruct{scenarioName: "send_convertsAndFailsForwardToRw", filePath: "../resources/sourceJson/noTmeIds.json", endpoint: "/transform/send", expectedStatusCode: 500, expectedResult: "Internal Error: Delete request to writer returned unexpected status:"}

	testScenarios := []testStruct{transform_unprocessibleEntityError, transform_convertingToConcordedJsonError, transform_duplicateTmeIdsError, transform_convertsAndReturnsPayload, send_unprocessibleEntityError, send_convertingToConcordedJsonError, send_convertsAndForwardsPayloadWithConcordance, send_convertsAndFailsForwardToRw}

	for _, scenario := range testScenarios {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, newRequest("POST", scenario.endpoint, readFile(t, scenario.filePath)))
		assert.Equal(t, scenario.expectedStatusCode, rec.Code, scenario.scenarioName)
		assert.Equal(t, rec.HeaderMap["Content-Type"], []string{"application/json"}, scenario.scenarioName)
		assert.Contains(t, rec.Body.String(), scenario.expectedResult, "Failed scenario: "+scenario.scenarioName)
	}

}

func TestSendHandlerSuccessfulDelete(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHttpClient{resp: "", statusCode: 404}
	defaultTransformer := NewTransformerService(TOPIC, WRITER_ADDRESS, &mockClient)
	h := NewHandler(defaultTransformer, mockConsumer{})
	h.RegisterHandlers(r)

	type testStruct struct {
		scenarioName       string
		filePath           string
		endpoint           string
		expectedStatusCode int
		expectedResult     string
	}

	send_convertsAndForwardsPayloadWithoutConcordance := testStruct{scenarioName: "send_convertsAndForwardsPayloadWithoutConcordance", filePath: "../resources/sourceJson/noTmeIds.json", endpoint: "/transform/send", expectedStatusCode: 200, expectedResult: "Concordance record for " + testUuid + " forwarded to writer"}

	testScenarios := []testStruct{send_convertsAndForwardsPayloadWithoutConcordance}

	for _, scenario := range testScenarios {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, newRequest("POST", scenario.endpoint, readFile(t, scenario.filePath)))
		assert.Equal(t, scenario.expectedStatusCode, rec.Code, scenario.scenarioName)
		assert.Equal(t, rec.HeaderMap["Content-Type"], []string{"application/json"}, scenario.scenarioName)
		assert.Contains(t, rec.Body.String(), scenario.expectedResult, "Failed scenario: "+scenario.scenarioName)
	}

}

func (c mockHttpClient) Do(req *http.Request) (resp *http.Response, err error) {
	cb := ioutil.NopCloser(bytes.NewReader([]byte(c.resp)))
	return &http.Response{Body: cb, StatusCode: c.statusCode}, c.err
}

type mockConsumer struct {
	err error
}

func (mc mockConsumer) ConnectivityCheck() error {
	return mc.err
}

func (mc mockConsumer) StartListening(messageHandler func(message kafka.FTMessage) error) {
	return
}

func (mc mockConsumer) Shutdown() {
	return
}

func newRequest(method, url string, body string) *http.Request {
	var payload io.Reader
	if body != "" {
		payload = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequest(method, url, payload)
	req.Header = map[string][]string{
		"Content-Type": {ExpectedContentType},
	}
	if err != nil {
		panic(err)
	}
	return req
}
