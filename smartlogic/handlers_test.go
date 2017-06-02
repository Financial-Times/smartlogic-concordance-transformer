package smartlogic

//import (
//	"net/http"
//	"bytes"
//	"github.com/gorilla/mux"
//	"github.com/stretchr/testify/assert"
//	"testing"
//	"net/http/httptest"
//	"io"
//)
//
//const (
//	ExpectedContentType = "application/json"
//)
//
//type mockHttpClient struct {
//	resp       string
//	statusCode int
//	err        error
//}
//
//func TestAdminHandler_PingAndBuildInfo(t *testing.T) {
//	r := mux.NewRouter()
//	mhc := mockHttpClient{}
//	ms3d := mockTransformer{}
//	msQsD := mockSqsDriver{}
//	h := NewHandler(&ms3d, &msQsD, VULCAN, &mhc)
//	h.RegisterAdminHandlers(r)
//	rec := httptest.NewRecorder()
//
//	type testStruct struct {
//		endpoint           string
//		expectedStatusCode int
//		expectedBody       string
//		expectedError      string
//	}
//
//	pingChecker := testStruct{endpoint: "/__ping", expectedStatusCode: 200, expectedBody: "pong"}
//	buildInfoChecker := testStruct{endpoint: "/__build-info", expectedStatusCode: 200, expectedBody: "Version  is not a semantic version"}
//
//	Collections := []testStruct{pingChecker, buildInfoChecker}
//
//	for _, col := range Collections {
//		http.DefaultServeMux.ServeHTTP(rec, newRequest("GET", col.endpoint, ""))
//		assert.Equal(t, col.expectedStatusCode, rec.Code)
//		assert.Contains(t, rec.Body.String(), col.expectedBody)
//	}
//}
//
//func newRequest(method, url string, body string) *http.Request {
//	var payload io.Reader
//	if body != "" {
//		payload = bytes.NewReader([]byte(body))
//	}
//	req, err := http.NewRequest(method, url, payload)
//	req.Header = map[string][]string{
//		"Content-Type": {ExpectedContentType},
//	}
//	if err != nil {
//		panic(err)
//	}
//	return req
//}