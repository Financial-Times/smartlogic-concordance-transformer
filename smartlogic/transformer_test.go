package smartlogic

//import (
//	"net/http"
//	"io/ioutil"
//	"bytes"
//	"github.com/gorilla/mux"
//	"github.com/stretchr/testify/assert"
//	"testing"
//	"net/http/httptest"
//	"io"
//)
//
//type mockHttpClient struct {
//	resp       string
//	statusCode int
//	err        error
//}
//
//
//func (c mockHttpClient) Do(req *http.Request) (resp *http.Response, err error) {
//	cb := ioutil.NopCloser(bytes.NewReader([]byte(c.resp)))
//	return &http.Response{Body: cb, StatusCode: c.statusCode}, c.err
//}
//
//const (
//	conceptUuid string = "9a38d2a8-4ebe-49ca-a02f-a8209095b297"
//	parentUuid string = "fd44d98b-50aa-448b-8ebf-ff0d135a30a2"
//	ExpectedContentType = "application/json"
//)
