package smartlogic

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"testing"
	"net/http/httptest"
	"net/http"
)
func TestAdminHandler_Healthy(t *testing.T) {
	r := mux.NewRouter()
	mockClient := mockHttpClient{resp: "", statusCode: 200}
	defaultTransformer := NewTransformerService(TOPIC, WRITER_ADDRESS, &mockClient)
	h := NewHandler(defaultTransformer, mockConsumer{})
	h.RegisterAdminHandlers(r, "appy-mcappface", "Appy-McAppface", "My first app")

	type testStruct struct {
		endpoint           string
		expectedStatusCode int
		expectedBody       string
		expectedError      string
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
