package smartlogic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestAdminHandler_Healthy(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHTTPClient{resp: "", statusCode: 200}
	defaultTransformer := NewTransformerService(TOPIC, WriterAddress, &mockClient, createLogger())
	h := NewHandler(defaultTransformer, mockConsumer{}, createLogger())
	h.RegisterAdminHandlers(r, "appy-mcappface", "Appy-McAppface", "My first app")

	type testStruct struct {
		endpoint           string
		expectedStatusCode int
		expectedBody       string
	}

	buildInfoChecker := testStruct{endpoint: "/__build-info", expectedStatusCode: 200, expectedBody: "Version  is not a semantic version"}
	gtgChecker := testStruct{endpoint: "/__gtg", expectedStatusCode: 200, expectedBody: ""}
	healthChecker := testStruct{endpoint: "/__health", expectedStatusCode: 200, expectedBody: ""}

	testScenarios := []testStruct{buildInfoChecker, gtgChecker, healthChecker}

	for _, scenario := range testScenarios {
		rec := httptest.NewRecorder()
		http.DefaultServeMux.ServeHTTP(rec, newRequest("GET", scenario.endpoint, ""))
		assert.Equal(t, scenario.expectedStatusCode, rec.Code)
		assert.Contains(t, rec.Body.String(), scenario.expectedBody)
	}
}
