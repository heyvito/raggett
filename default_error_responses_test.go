package raggett

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func doRequest(mux *Mux, accept, method, path string, body io.Reader) (int, string) {
	r := httptest.NewRequest(method, path, body)
	r.Header.Set("Accept", accept)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, r)
	return rw.Code, rw.Body.String()
}

func TestServerErrorDevelopment(t *testing.T) {
	t.Parallel()
	m := NewMux(zap.NewNop())
	m.Development = true
	type RequestType struct {
		*Request
		File *FileHeader `form:"file"`
	}

	m.Post("/", func(r *RequestType) error {
		return fmt.Errorf("boom")
	})

	t.Run("Text", func(t *testing.T) {
		code, body := doRequest(m, "text/plain", "POST", "/", nil)
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.Contains(t, body, "Development = false")
	})

	t.Run("HTML", func(t *testing.T) {
		code, body := doRequest(m, "text/html", "POST", "/", nil)
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.Contains(t, body, "<code>Development = false</code>")
	})

	t.Run("XML", func(t *testing.T) {
		code, body := doRequest(m, "text/xml", "POST", "/", nil)
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.Contains(t, body, "<status_name>Internal Server Error</status_name>")
	})

	t.Run("JSON", func(t *testing.T) {
		code, body := doRequest(m, "application/json", "POST", "/", nil)
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.Contains(t, body, `"status_name":"Internal Server Error"`)
	})

	t.Run("Stack Trace", func(t *testing.T) {
		m.Post("/stack", func(requestType *RequestType) error {
			return makeError(fmt.Errorf("boom"))
		})

		buffer := bytes.Buffer{}
		writer := multipart.NewWriter(&buffer)
		err := writer.WriteField("test", "true")
		require.NoError(t, err)
		w, err := writer.CreateFormFile("files-field", "raggett1.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("this is a test"))
		require.NoError(t, err)
		w, err = writer.CreateFormFile("files-field", "raggett2.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("this is another test"))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		httpReq := httptest.NewRequest("POST", "/stack?foo=bar", bytes.NewReader(buffer.Bytes()))
		httpReq.Header.Set("Content-Type", writer.FormDataContentType())
		httpReq.Header.Set("Accept", "text/html")
		resp := httptest.NewRecorder()
		m.ServeHTTP(resp, httpReq)
		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.Contains(t, resp.Body.String(), "default_error_responses_test.go")
		assert.Contains(t, resp.Body.String(), "<code>Development = false</code>")
	})
}

func TestServerErrorConstrained(t *testing.T) {
	t.Parallel()
	m := NewMux(zap.NewNop())
	m.Development = false
	type RequestType struct {
		*Request
	}

	m.Post("/", func(r *RequestType) error {
		return fmt.Errorf("boom")
	})

	t.Run("Text", func(t *testing.T) {
		code, body := doRequest(m, "text/plain", "POST", "/", nil)
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.NotContains(t, body, "Development = false")
	})

	t.Run("HTML", func(t *testing.T) {
		code, body := doRequest(m, "text/html", "POST", "/", nil)
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.NotContains(t, body, "<code>Development = false</code>")
	})

	t.Run("XML", func(t *testing.T) {
		code, body := doRequest(m, "text/xml", "POST", "/", nil)
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.NotContains(t, body, "<environments>")
	})

	t.Run("JSON", func(t *testing.T) {
		code, body := doRequest(m, "application/json", "POST", "/", nil)
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.NotContains(t, body, `"environments"`)
	})
}

func TestValidationErrorDevelopment(t *testing.T) {
	t.Parallel()
	m := NewMux(zap.NewNop())
	m.Development = true
	type RequestType struct {
		*Request
		SomeFile      *FileHeader `form:"file" required:"true"`
		RequiredValue string      `header:"required-value" required:"true" blank:"false"`
	}

	m.Post("/", func(r *RequestType) error {
		return fmt.Errorf("boom")
	})

	t.Run("Text", func(t *testing.T) {
		code, body := doRequest(m, "text/plain", "POST", "/", nil)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, body, "Development = false")
	})

	t.Run("HTML", func(t *testing.T) {
		code, body := doRequest(m, "text/html", "POST", "/", nil)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, body, "<code>Development = false</code>")
	})

	t.Run("XML", func(t *testing.T) {
		code, body := doRequest(m, "text/xml", "POST", "/", nil)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, body, "<status_name>Bad Request</status_name>")
	})

	t.Run("JSON", func(t *testing.T) {
		code, body := doRequest(m, "application/json", "POST", "/", nil)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, body, `"status_name":"Bad Request"`)
	})

	t.Run("Stack Trace", func(t *testing.T) {
		m.Post("/stack", func(requestType *RequestType) error {
			return fmt.Errorf("boom")
		})

		buffer := bytes.Buffer{}
		writer := multipart.NewWriter(&buffer)
		err := writer.WriteField("test", "true")
		require.NoError(t, err)
		w, err := writer.CreateFormFile("files-field", "raggett1.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("this is a test"))
		require.NoError(t, err)
		w, err = writer.CreateFormFile("files-field", "raggett2.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("this is another test"))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		httpReq := httptest.NewRequest("POST", "/stack?foo=bar", bytes.NewReader(buffer.Bytes()))
		httpReq.Header.Set("Content-Type", writer.FormDataContentType())
		httpReq.Header.Set("Accept", "text/html")
		httpReq.Header.Set("required-value", "true")
		resp := httptest.NewRecorder()
		m.ServeHTTP(resp, httpReq)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "raggett1.txt")
		assert.Contains(t, resp.Body.String(), "<code>Development = false</code>")
	})
}

func TestValidationErrorConstrained(t *testing.T) {
	t.Parallel()
	m := NewMux(zap.NewNop())
	m.Development = false
	type RequestType struct {
		*Request
		RequiredValue string `header:"required-value" required:"true" blank:"false"`
	}

	m.Post("/", func(r *RequestType) error {
		return fmt.Errorf("boom")
	})

	t.Run("Text", func(t *testing.T) {
		code, body := doRequest(m, "text/plain", "POST", "/", nil)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.NotContains(t, body, "Development = false")
	})

	t.Run("HTML", func(t *testing.T) {
		code, body := doRequest(m, "text/html", "POST", "/", nil)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.NotContains(t, body, "<code>Development = false</code>")
	})

	t.Run("XML", func(t *testing.T) {
		code, body := doRequest(m, "text/xml", "POST", "/", nil)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.NotContains(t, body, "<environments>")
	})

	t.Run("JSON", func(t *testing.T) {
		code, body := doRequest(m, "application/json", "POST", "/", nil)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.NotContains(t, body, `"environments"`)
	})
}

type notFoundController struct{}
type notFoundRequestType struct {
	*Request
	File *FileHeader `form:"file"`
}

func (notFoundController) Get(*notFoundRequestType) error {
	return fmt.Errorf("boom")
}
func (notFoundController) Post(*notFoundRequestType) error {
	return fmt.Errorf("boom")
}

func TestNotFoundErrorDevelopment(t *testing.T) {
	t.Parallel()
	m := NewMux(zap.NewNop())
	m.Development = true
	ctrl := notFoundController{}
	m.Post("/test/{name}", ctrl.Post)
	m.Get("/test/{name}", ctrl.Get)

	t.Run("Text", func(t *testing.T) {
		code, body := doRequest(m, "text/plain", "POST", "/", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Contains(t, body, "Development = false")
	})

	t.Run("HTML", func(t *testing.T) {
		buffer := bytes.Buffer{}
		writer := multipart.NewWriter(&buffer)
		err := writer.WriteField("test", "true")
		require.NoError(t, err)
		w, err := writer.CreateFormFile("files-field", "raggett1.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("this is a test"))
		require.NoError(t, err)
		w, err = writer.CreateFormFile("files-field", "raggett2.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("this is another test"))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		code, body := doRequest(m, "text/html", "POST", "/", bytes.NewReader(buffer.Bytes()))
		assert.Equal(t, http.StatusNotFound, code)
		assert.Contains(t, body, "<code>Development = false</code>")
	})

	t.Run("XML", func(t *testing.T) {
		code, body := doRequest(m, "text/xml", "POST", "/", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Contains(t, body, "<environments>")
	})

	t.Run("JSON", func(t *testing.T) {
		code, body := doRequest(m, "application/json", "POST", "/", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Contains(t, body, `"environment"`)
	})
}

func TestNotFoundErrorConstrained(t *testing.T) {
	t.Parallel()
	m := NewMux(zap.NewNop())
	m.Development = false
	ctrl := notFoundController{}
	m.Post("/test/{name}", ctrl.Post)
	m.Get("/test/{name}", ctrl.Get)

	t.Run("Text", func(t *testing.T) {
		code, body := doRequest(m, "text/plain", "POST", "/", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.NotContains(t, body, "Development = false")
	})

	t.Run("HTML", func(t *testing.T) {
		code, body := doRequest(m, "text/html", "POST", "/", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.NotContains(t, body, "<code>Development = false</code>")
	})

	t.Run("XML", func(t *testing.T) {
		code, body := doRequest(m, "text/xml", "POST", "/", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.NotContains(t, body, "<environments>")
	})

	t.Run("JSON", func(t *testing.T) {
		code, body := doRequest(m, "application/json", "POST", "/", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.NotContains(t, body, `"environment"`)
	})
}

func TestMethodNotAllowedErrorDevelopment(t *testing.T) {
	t.Parallel()
	m := NewMux(zap.NewNop())
	m.Development = true
	ctrl := notFoundController{}
	m.Post("/test/{name}", ctrl.Post)
	m.Get("/test/{name}", ctrl.Get)

	t.Run("Text", func(t *testing.T) {
		code, body := doRequest(m, "text/plain", "UPDATE", "/test/test", nil)
		assert.Equal(t, http.StatusMethodNotAllowed, code)
		assert.Contains(t, body, "Development = false")
	})

	t.Run("HTML", func(t *testing.T) {
		buffer := bytes.Buffer{}
		writer := multipart.NewWriter(&buffer)
		err := writer.WriteField("test", "true")
		require.NoError(t, err)
		w, err := writer.CreateFormFile("files-field", "raggett1.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("this is a test"))
		require.NoError(t, err)
		w, err = writer.CreateFormFile("files-field", "raggett2.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("this is another test"))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		code, body := doRequest(m, "text/html", "UPDATE", "/test/test", bytes.NewReader(buffer.Bytes()))
		assert.Equal(t, http.StatusMethodNotAllowed, code)
		assert.Contains(t, body, "<code>Development = false</code>")
	})

	t.Run("XML", func(t *testing.T) {
		code, body := doRequest(m, "text/xml", "UPDATE", "/test/test", nil)
		assert.Equal(t, http.StatusMethodNotAllowed, code)
		assert.Contains(t, body, "<environments>")
	})

	t.Run("JSON", func(t *testing.T) {
		code, body := doRequest(m, "application/json", "UPDATE", "/test/test", nil)
		assert.Equal(t, http.StatusMethodNotAllowed, code)
		assert.Contains(t, body, `"environment"`)
	})
}

func TestMethodNotAllowedErrorConstrained(t *testing.T) {
	t.Parallel()
	m := NewMux(zap.NewNop())
	m.Development = false
	ctrl := notFoundController{}
	m.Post("/test/{name}", ctrl.Post)
	m.Get("/test/{name}", ctrl.Get)

	t.Run("Text", func(t *testing.T) {
		code, body := doRequest(m, "text/plain", "UPDATE", "/test/test", nil)
		assert.Equal(t, http.StatusMethodNotAllowed, code)
		assert.NotContains(t, body, "Development = false")
	})

	t.Run("HTML", func(t *testing.T) {
		code, body := doRequest(m, "text/html", "UPDATE", "/test/test", nil)
		assert.Equal(t, http.StatusMethodNotAllowed, code)
		assert.NotContains(t, body, "<code>Development = false</code>")
	})

	t.Run("XML", func(t *testing.T) {
		code, body := doRequest(m, "text/xml", "UPDATE", "/test/test", nil)
		assert.Equal(t, http.StatusMethodNotAllowed, code)
		assert.NotContains(t, body, "<environments>")
	})

	t.Run("JSON", func(t *testing.T) {
		code, body := doRequest(m, "application/json", "UPDATE", "/test/test", nil)
		assert.Equal(t, http.StatusMethodNotAllowed, code)
		assert.NotContains(t, body, `"environment"`)
	})
}
