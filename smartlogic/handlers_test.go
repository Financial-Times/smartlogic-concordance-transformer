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

func TestAdminHandler_PingAndBuildInfo(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHttpClient{}
	defaultTransformer := NewTransformerService(TOPIC, WRITER_ADDRESS, &mockClient)
	h := NewHandler(defaultTransformer)
	h.Consumer = mockConsumer{}
	h.RegisterAdminHandlers(r)
	rec := httptest.NewRecorder()

	type testStruct struct {
		endpoint           string
		expectedStatusCode int
		expectedBody       string
		expectedError      string
	}

	pingChecker := testStruct{endpoint: "/__ping", expectedStatusCode: 200, expectedBody: "pong"}
	buildInfoChecker := testStruct{endpoint: "/__build-info", expectedStatusCode: 200, expectedBody: "Version  is not a semantic version"}

	Collections := []testStruct{pingChecker, buildInfoChecker}

	for _, col := range Collections {
		http.DefaultServeMux.ServeHTTP(rec, newRequest("GET", col.endpoint, ""))
		assert.Equal(t, col.expectedStatusCode, rec.Code)
		assert.Contains(t, rec.Body.String(), col.expectedBody)
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