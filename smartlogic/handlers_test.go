package smartlogic

import (
	"net/http"
	"bytes"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"testing"
	"net/http/httptest"
	"io"
	"io/ioutil"
)

const (
	ExpectedContentType = "application/json"
	WRITER_ADDRESS = "http://localhost:8080/__concordance-rw-dynamodb/"
	TOPIC = "TestTopic"
)

type mockHttpClient struct {
	resp       string
	statusCode int
	err        error
}

type mockConsumer struct {
	err error
}

func TestAdminHandler_Healthy(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHttpClient{resp: "", statusCode: 200}
	defaultTransformer := NewTransformerService(TOPIC, WRITER_ADDRESS, &mockClient)
	h := NewHandler(defaultTransformer)
	h.Consumer = mockConsumer{}
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

func TestTransformHandler(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHttpClient{resp: "", statusCode: 200}
	defaultTransformer := NewTransformerService(TOPIC, WRITER_ADDRESS, &mockClient)
	h := NewHandler(defaultTransformer)
	h.Consumer = mockConsumer{}
	h.RegisterHandlers(r)

	type testStruct struct {
		filePath string
		endpoint string
		expectedStatusCode int
		expectedResult string
	}

	transform_convertingToConcordedJsonError := testStruct{filePath: "../resources/sourceJson/invalidTmeId.json", endpoint: "/transform", expectedStatusCode: 422, expectedResult: "{\"message\":\"Error whilst converting to concorded json: "}
	transform_duplicateTmeIdsError := testStruct{filePath: "../resources/sourceJson/duplicateTmeIds.json", endpoint: "/transform", expectedStatusCode: 422, expectedResult: "contains duplicate TME id values"}
	transform_convertsAndReturnsPayload := testStruct{filePath: "../resources/sourceJson/multipleTmeIds.json", endpoint: "/transform", expectedStatusCode: 200, expectedResult: "{\"uuid\":\"20db1bd6-59f9-4404-adb5-3165a448f8b0\",\"concordedIds\":[\"e9f4525a-401f-3b23-a68e-e48f314cdce6\",\"83f63c7e-1641-3c7b-81e4-378ae3c6c2ad\",\"e4bc4ac2-0637-3a27-86b1-9589fca6bf2c\",\"e574b21d-9abc-3d82-a6c0-3e08c85181bf\"]}"}
	send_convertingToConcordedJsonError := testStruct{filePath: "../resources/sourceJson/invalidTmeId.json", endpoint: "/transform/send", expectedStatusCode: 422, expectedResult: "{\"message\":\"Error whilst processing request body:Conversion of payload to upp concordance resulted in error:"}
	send_convertsAndReturnsPayload := testStruct{filePath: "../resources/sourceJson/multipleTmeIds.json", endpoint: "/transform/send", expectedStatusCode: 200, expectedResult: "Concordance successfully written to db"}

	testScenarios := []testStruct{transform_convertingToConcordedJsonError, transform_duplicateTmeIdsError, transform_convertsAndReturnsPayload, send_convertingToConcordedJsonError, send_convertsAndReturnsPayload}

	for _, scenario := range testScenarios {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, newRequest("POST", scenario.endpoint, readFile(t, scenario.filePath)))
		assert.Equal(t, scenario.expectedStatusCode, rec.Code)
		assert.Equal(t, rec.HeaderMap["Content-Type"], []string{"application/json"})
		assert.Contains(t, rec.Body.String(), scenario.expectedResult)
	}


}

func TestSendHandler(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHttpClient{}
	defaultTransformer := NewTransformerService(TOPIC, WRITER_ADDRESS, &mockClient)
	h := NewHandler(defaultTransformer)
	h.Consumer = mockConsumer{}
	h.RegisterHandlers(r)

	type testStruct struct {
		filePath string
		endpoint string
		expectedStatusCode int
		expectedResult string
	}

	convertingToConcordedJsonError := testStruct{filePath: "../resources/sourceJson/invalidTmeId.json", endpoint: "/transform", expectedStatusCode: 422, expectedResult: "{\"message\":\"Error whilst converting to concorded json: "}
	convertsAndReturnsPayload := testStruct{filePath: "../resources/sourceJson/multipleTmeIds.json", endpoint: "/transform", expectedStatusCode: 200, expectedResult: "{\"uuid\":\"20db1bd6-59f9-4404-adb5-3165a448f8b0\",\"concordedIds\":[\"e9f4525a-401f-3b23-a68e-e48f314cdce6\",\"83f63c7e-1641-3c7b-81e4-378ae3c6c2ad\",\"e4bc4ac2-0637-3a27-86b1-9589fca6bf2c\",\"e574b21d-9abc-3d82-a6c0-3e08c85181bf\"]}"}

	testScenarios := []testStruct{convertingToConcordedJsonError, convertsAndReturnsPayload}

	for _, scenario := range testScenarios {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, newRequest("POST", scenario.endpoint, readFile(t, scenario.filePath)))
		assert.Equal(t, scenario.expectedStatusCode, rec.Code)
		assert.Equal(t, rec.HeaderMap["Content-Type"], []string{"application/json"})
		assert.Contains(t, rec.Body.String(), scenario.expectedResult)
	}


}

func (c mockHttpClient) Do(req *http.Request) (resp *http.Response, err error) {
	cb := ioutil.NopCloser(bytes.NewReader([]byte(c.resp)))
	return &http.Response{Body: cb, StatusCode: c.statusCode}, c.err
}

func (mc mockConsumer) ConnectivityCheck() (string, error) {
	return "", mc.err
}

func (mc mockConsumer) Start() {
	return
}

func (mc mockConsumer) Stop() {
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