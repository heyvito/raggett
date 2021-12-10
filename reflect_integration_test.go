package raggett

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestJSONLoader(t *testing.T) {
	type JSONContent struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	type RequestType struct {
		*Request
		User JSONContent `body:"json"`
	}

	data := `{"name": "Raggett", "email": "email@example.org"}`
	req := httptest.NewRequest("POST", "/foo", strings.NewReader(data))
	req.Header.Add("Content-Type", "application/json")

	var r *RequestType
	w, validationError, runtimeError := testMuxPostWith(t, req, "/foo", func(req *RequestType) error {
		r = req
		return nil
	})
	// Validations must pass
	require.NoError(t, validationError)

	// Runtime must succeed
	require.NoError(t, runtimeError)

	// Status must be NoContent
	require.Equal(t, http.StatusNoContent, w.Code)

	assert.Equal(t, "email@example.org", r.User.Email)
	assert.Equal(t, "Raggett", r.User.Name)
}

func TestXMLLoader(t *testing.T) {
	type XMLContent struct {
		XMLName xml.Name `xml:"data"`
		Name    string   `xml:"name"`
		Email   string   `xml:"email"`
	}

	type RequestType struct {
		*Request
		User XMLContent `body:"xml"`
	}

	data := `<data><name>Raggett</name><email>email@example.org</email></data>`
	req := httptest.NewRequest("POST", "/foo", strings.NewReader(data))
	req.Header.Add("Content-Type", "application/xml")

	var r *RequestType
	w, validationError, runtimeError := testMuxPostWith(t, req, "/foo", func(req *RequestType) error {
		r = req
		return nil
	})
	// Validations must pass
	require.NoError(t, validationError)

	// Runtime must succeed
	require.NoError(t, runtimeError)

	// Status must be NoContent
	require.Equal(t, http.StatusNoContent, w.Code)

	assert.Equal(t, "email@example.org", r.User.Email)
	assert.Equal(t, "Raggett", r.User.Name)
}

func TestTextLoader(t *testing.T) {
	type RequestType struct {
		*Request
		Text string `body:"text"`
	}

	data := `Raggett email@example.org`
	req := httptest.NewRequest("POST", "/foo", strings.NewReader(data))
	req.Header.Add("Content-Type", "application/xml")

	var r *RequestType
	w, validationError, runtimeError := testMuxPostWith(t, req, "/foo", func(req *RequestType) error {
		r = req
		return nil
	})
	// Validations must pass
	require.NoError(t, validationError)

	// Runtime must succeed
	require.NoError(t, runtimeError)

	// Status must be NoContent
	require.Equal(t, http.StatusNoContent, w.Code)

	assert.Equal(t, data, r.Text)
}

func TestBytesLoader(t *testing.T) {
	type RequestType struct {
		*Request
		Bytes []byte `body:"bytes"`
	}

	data := `Raggett email@example.org`
	req := httptest.NewRequest("POST", "/foo", strings.NewReader(data))
	req.Header.Add("Content-Type", "application/xml")

	var r *RequestType
	w, validationError, runtimeError := testMuxPostWith(t, req, "/foo", func(req *RequestType) error {
		r = req
		return nil
	})
	// Validations must pass
	require.NoError(t, validationError)

	// Runtime must succeed
	require.NoError(t, runtimeError)

	// Status must be NoContent
	require.Equal(t, http.StatusNoContent, w.Code)

	assert.Equal(t, []byte(data), r.Bytes)
}

func TestStreamLoader(t *testing.T) {
	type RequestType struct {
		*Request
		Sream io.ReadCloser `body:"stream"`
	}

	data := `Raggett email@example.org`
	req := httptest.NewRequest("POST", "/foo", strings.NewReader(data))
	req.Header.Add("Content-Type", "application/xml")

	var r *RequestType
	w, validationError, runtimeError := testMuxPostWith(t, req, "/foo", func(req *RequestType) error {
		r = req
		return nil
	})
	// Validations must pass
	require.NoError(t, validationError)

	// Runtime must succeed
	require.NoError(t, runtimeError)

	// Status must be NoContent
	require.Equal(t, http.StatusNoContent, w.Code)

	res, err := io.ReadAll(r.Sream)
	require.NoError(t, err)
	assert.Equal(t, []byte(data), res)
}

func testMuxPostWith(t *testing.T, request *http.Request, pattern string, handler interface{}) (resp *httptest.ResponseRecorder, validationError, runtimeError error) {
	m := NewMux(zap.NewNop())
	m.HandleValidationError(func(err ValidationError, w http.ResponseWriter, r *Request) {
		validationError = err
		originalErrStr := "«nil»"

		if err.OriginalError != nil {
			if o, ok := err.OriginalError.(Error); ok {
				originalErrStr = o.String()
			} else {
				originalErrStr = err.OriginalError.Error()
			}
		}
		t.Logf("Validation error: %s\nOriginal Error: %s", err.Error(), originalErrStr)
		w.WriteHeader(http.StatusBadRequest)
	})
	m.HandleError(func(err error, w http.ResponseWriter, r *Request) {
		runtimeError = err
		t.Logf("Runtime error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	})

	m.Post(pattern, handler)

	resp = httptest.NewRecorder()
	m.ServeHTTP(resp, request)
	return
}

//go:generate go run generators/type_test_file/generate_type_test_file.go
func TestAllLoaders(t *testing.T) {
	t.Parallel()

	httpRequest := generateIntegrationTestRequest()

	t.Logf("Notice: Registered HTTP endpoint is POST %s", integrationTestURLPattern)
	dumpRequest(t, httpRequest)

	var r *GeneratedRequestType
	w, validationError, runtimeError := testMuxPostWith(t, httpRequest, integrationTestURLPattern, func(req *GeneratedRequestType) error {
		r = req
		return nil
	})
	// Validations must pass
	require.NoError(t, validationError)

	// Runtime must succeed
	require.NoError(t, runtimeError)

	// Status must be NoContent
	require.Equal(t, http.StatusNoContent, w.Code)

	// Then perform other validations
	assertIntegrationRequestData(t, r)
}
