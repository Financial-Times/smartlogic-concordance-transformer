package smartlogic

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

const (
	ExpectedContentType = "application/json"
	WriterAddress       = "http://localhost:8080/__concordance-rw-neo4j/"
	TOPIC               = "TestTopic"
)

type mockHTTPClient struct {
	resp       string
	statusCode int
	err        error
}

func TestTransformAndSendHandlers(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHTTPClient{resp: "", statusCode: 200}
	defaultTransformer := NewTransformerService(TOPIC, WriterAddress, &mockClient, createLogger())
	h := NewHandler(defaultTransformer, mockConsumer{}, createLogger())
	h.RegisterHandlers(r)

	type testStruct struct {
		scenarioName       string
		filePath           string
		endpoint           string
		expectedStatusCode int
		expectedResult     string
	}

	transformUnprocessibleEntityError := testStruct{scenarioName: "transform_unprocessibleEntityError", filePath: "../resources/multipleGraphsInList.json", endpoint: "/transform", expectedStatusCode: 422, expectedResult: "invalid Request Json: More than 1 concept in smartlogic concept payload which is currently not supported"}
	transformConvertingToConcordedJSONError := testStruct{scenarioName: "transform_convertingToConcordedJsonError", filePath: "../resources/invalidTmeId.json", endpoint: "/transform", expectedStatusCode: 400, expectedResult: "is not a valid TME Id"}
	transformDuplicateTMEIDsError := testStruct{scenarioName: "transform_duplicateTmeIdsError", filePath: "../resources/duplicateTmeIds.json", endpoint: "/transform", expectedStatusCode: 400, expectedResult: "contains duplicate TME id values"}
	sendUnprocessibleEntityError := testStruct{scenarioName: "send_unprocessibleEntityError", filePath: "../resources/multipleGraphsInList.json", endpoint: "/transform/send", expectedStatusCode: 422, expectedResult: "invalid Request Json: More than 1 concept in smartlogic concept payload which is currently not supported"}
	sendConvertingToConcordedJSONError := testStruct{scenarioName: "send_convertingToConcordedJsonError", filePath: "../resources/invalidTmeId.json", endpoint: "/transform/send", expectedStatusCode: 400, expectedResult: "is not a valid TME Id"}
	sendConvertsAndFailsForwardToRW := testStruct{scenarioName: "send_convertsAndFailsForwardToRw", filePath: "../resources/noTmeIds.json", endpoint: "/transform/send", expectedStatusCode: 500, expectedResult: "Internal Error: Delete request to writer returned unexpected status:"}

	transformUnprocessibleEntityNotAllowedConceptTypeError := testStruct{
		scenarioName:       "transform_unprocessibleEntityNotAllowedConceptTypeError",
		filePath:           "../resources/notAllowedType.json",
		endpoint:           "/transform",
		expectedStatusCode: 422,
		expectedResult:     "concept type not allowed",
	}
	transformDuplicateFactSetIDsError := testStruct{
		scenarioName:       "transform_duplicateFactsetIdsError",
		filePath:           "../resources/duplicateFactsetIds.json",
		endpoint:           "/transform",
		expectedStatusCode: 400,
		expectedResult:     "contains duplicate FACTSET id values",
	}
	transformConvertsAndReturnsPayload := testStruct{
		scenarioName:       "transform_convertsAndReturnsPayload",
		filePath:           "../resources/multipleTmeIds.json",
		endpoint:           "/transform",
		expectedStatusCode: 200,
		expectedResult:     `{"authority":"Smartlogic","uuid":"20db1bd6-59f9-4404-adb5-3165a448f8b0","concordances":[{"authority":"TME","authorityValue":"AbCdEfgHiJkLMnOpQrStUvWxYz-0123456789","uuid":"e9f4525a-401f-3b23-a68e-e48f314cdce6"},{"authority":"TME","authorityValue":"ZyXwVuTsRqPoNmLkJiHgFeDcBa-0987654321","uuid":"83f63c7e-1641-3c7b-81e4-378ae3c6c2ad"},{"authority":"TME","authorityValue":"abcdefghijklmnopqrstuvwxyz-0123456789","uuid":"e4bc4ac2-0637-3a27-86b1-9589fca6bf2c"},{"authority":"TME","authorityValue":"ABCDEFGHIJKLMNOPQRSTUVWXYZ-0987654321","uuid":"e574b21d-9abc-3d82-a6c0-3e08c85181bf"}]}`,
	}
	transformConvertsFactsETSAndReturnsPayload := testStruct{
		scenarioName:       "transform_convertsFactsetsAndReturnsPayload",
		filePath:           "../resources/multipleFactsetIds.json",
		endpoint:           "/transform",
		expectedStatusCode: 200,
		expectedResult:     `{"authority":"Smartlogic","uuid":"20db1bd6-59f9-4404-adb5-3165a448f8b0","concordances":[{"authority":"FACTSET","authorityValue":"000D63-E","uuid":"8d3aba95-02d9-3802-afc0-b99bb9b1139e"},{"authority":"FACTSET","authorityValue":"023456-E","uuid":"3bc0ab41-c01f-3a0b-aa78-c76438080b52"},{"authority":"FACTSET","authorityValue":"023411-E","uuid":"f777c5af-e0b2-34dc-9102-e346ca2d27aa"}]}`,
	}
	transformConvertsTMEAndFactSetsAndReturnsPayload := testStruct{
		scenarioName:       "transform_convertsTmeAndFactsetsAndReturnsPayload",
		filePath:           "../resources/multipleTmeAndFactsetIds.json",
		endpoint:           "/transform",
		expectedStatusCode: 200,
		expectedResult:     `{"authority":"Smartlogic","uuid":"20db1bd6-59f9-4404-adb5-3165a448f8b0","concordances":[{"authority":"TME","authorityValue":"AbCdEfgHiJkLMnOpQrStUvWxYz-0123456789","uuid":"e9f4525a-401f-3b23-a68e-e48f314cdce6"},{"authority":"TME","authorityValue":"ZyXwVuTsRqPoNmLkJiHgFeDcBa-0987654321","uuid":"83f63c7e-1641-3c7b-81e4-378ae3c6c2ad"},{"authority":"TME","authorityValue":"abcdefghijklmnopqrstuvwxyz-0123456789","uuid":"e4bc4ac2-0637-3a27-86b1-9589fca6bf2c"},{"authority":"FACTSET","authorityValue":"000D63-E","uuid":"8d3aba95-02d9-3802-afc0-b99bb9b1139e"},{"authority":"FACTSET","authorityValue":"023456-E","uuid":"3bc0ab41-c01f-3a0b-aa78-c76438080b52"},{"authority":"FACTSET","authorityValue":"023411-E","uuid":"f777c5af-e0b2-34dc-9102-e346ca2d27aa"}]}`,
	}
	sendConvertsAndForwardsPayloadWithConcordance := testStruct{
		scenarioName:       "send_convertsAndForwardsPayloadWithConcordance",
		filePath:           "../resources/multipleTmeAndFactsetIds.json",
		endpoint:           "/transform/send",
		expectedStatusCode: 200,
		expectedResult:     `{"message":"Concordance record forwarded to writer"}`,
	}

	testScenarios := []testStruct{
		transformUnprocessibleEntityError,
		transformConvertingToConcordedJSONError,
		transformDuplicateTMEIDsError,
		transformDuplicateFactSetIDsError,
		transformConvertsAndReturnsPayload,
		transformConvertsFactsETSAndReturnsPayload,
		transformConvertsTMEAndFactSetsAndReturnsPayload,
		sendUnprocessibleEntityError,
		sendConvertingToConcordedJSONError,
		sendConvertsAndForwardsPayloadWithConcordance,
		sendConvertsAndFailsForwardToRW,
		transformUnprocessibleEntityNotAllowedConceptTypeError,
	}

	for _, scenario := range testScenarios {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, newRequest("POST", scenario.endpoint, readFile(t, scenario.filePath)))
		assert.Equal(t, scenario.expectedStatusCode, rec.Code, scenario.scenarioName)
		assert.Equal(t, rec.Header()["Content-Type"], []string{"application/json"}, scenario.scenarioName)
		assert.Contains(t, rec.Body.String(), scenario.expectedResult, "Failed scenario: "+scenario.scenarioName)
	}
}

func TestSendHandlerSuccessfulDelete(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHTTPClient{resp: "", statusCode: 404}
	defaultTransformer := NewTransformerService(TOPIC, WriterAddress, &mockClient, createLogger())
	h := NewHandler(defaultTransformer, mockConsumer{}, createLogger())
	h.RegisterHandlers(r)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, newRequest("POST", "/transform/send", readFile(t, "../resources/noTmeIds.json")))
	assert.Equal(t, 200, rec.Code, "Unexpected status code")
	assert.Equal(t, rec.Header()["Content-Type"], []string{"application/json"}, "Unexpected Content-Type")
	assert.Contains(t, rec.Body.String(), "Concordance record not found", "Request had unexpected result")
}

func TestSendHandlerRecordNotFound(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHTTPClient{resp: "", statusCode: 204}
	defaultTransformer := NewTransformerService(TOPIC, WriterAddress, &mockClient, createLogger())
	h := NewHandler(defaultTransformer, mockConsumer{}, createLogger())
	h.RegisterHandlers(r)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, newRequest("POST", "/transform/send", readFile(t, "../resources/noTmeIds.json")))
	assert.Equal(t, 200, rec.Code, "Unexpected status code")
	assert.Equal(t, rec.Header()["Content-Type"], []string{"application/json"}, "Unexpected Content-Type")
	assert.Contains(t, rec.Body.String(), "Concordance record successfully deleted", "Request had unexpected result")
}

func TestSendHandlerUnavailableWriter(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHTTPClient{resp: "", statusCode: 503}
	defaultTransformer := NewTransformerService(TOPIC, WriterAddress, &mockClient, createLogger())
	h := NewHandler(defaultTransformer, mockConsumer{}, createLogger())
	h.RegisterHandlers(r)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, newRequest("POST", "/transform/send", readFile(t, "../resources/noTmeIds.json")))
	assert.Equal(t, 500, rec.Code, "Unexpected status code")
	assert.Equal(t, rec.Header()["Content-Type"], []string{"application/json"}, "Unexpected Content-Type")
	assert.Contains(t, rec.Body.String(), "Delete request to writer returned unexpected status: 503", "Request had unexpected result")
}

func TestSendHandlerWriteReturnsError(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHTTPClient{resp: "", statusCode: 503, err: errors.New("delete request to writer returned unexpected status: 503")}
	defaultTransformer := NewTransformerService(TOPIC, WriterAddress, &mockClient, createLogger())
	h := NewHandler(defaultTransformer, mockConsumer{}, createLogger())
	h.RegisterHandlers(r)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, newRequest("POST", "/transform/send", readFile(t, "../resources/noTmeIds.json")))
	assert.Equal(t, 503, rec.Code, "Unexpected status code")
	assert.Equal(t, rec.Header()["Content-Type"], []string{"application/json"}, "Unexpected Content-Type")
	assert.Contains(t, rec.Body.String(), "delete request to writer returned unexpected status: 503", "Request had unexpected result")
}

func (c mockHTTPClient) Do(_ *http.Request) (resp *http.Response, err error) {
	cb := ioutil.NopCloser(bytes.NewReader([]byte(c.resp)))
	return &http.Response{Body: cb, StatusCode: c.statusCode}, c.err
}

func createLogger() *logger.UPPLogger {
	return logger.NewUPPLogger("smartlogic-test-logger", "debug")
}

type mockConsumer struct {
	err error
}

func (mc mockConsumer) MonitorCheck() error {
	return mc.err
}

func (mc mockConsumer) ConnectivityCheck() error {
	return mc.err
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
