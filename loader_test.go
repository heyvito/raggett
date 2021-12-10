package raggett

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLoader(t *testing.T) {
	type RequestStruct struct {
		*Request
		Foo string `query:"foo"`
	}
	httpReq := httptest.NewRequest("GET", "/foo/bar?foo=bar", nil)
	rec := httptest.NewRecorder()
	var receivedRec *RequestStruct
	handler := func(req *RequestStruct) error {
		receivedRec = req
		return nil
	}
	meta, err := determineFuncParams(handler)
	assert.NoError(t, err)
	req := NewRequest(rec, httpReq)
	err = loadAndApplyMeta(meta, req)
	assert.NoError(t, err)
	assert.Equal(t, "bar", receivedRec.Foo)
	assert.Equal(t, 200, rec.Code)
}

type CustomParserRequestStruct struct {
	*Request
	Foo string
}

func (req *CustomParserRequestStruct) ParseRequest(r *Request) error {
	req.Foo = r.HTTPRequest.URL.Query().Get("foo")
	return nil
}

func TestCustomLoader(t *testing.T) {
	httpReq := httptest.NewRequest("GET", "/foo/bar?foo=bar", nil)
	rec := httptest.NewRecorder()
	var receivedRec *CustomParserRequestStruct
	handler := func(req *CustomParserRequestStruct) error {
		receivedRec = req
		return nil
	}
	meta, err := determineFuncParams(handler)
	assert.NoError(t, err)
	req := NewRequest(rec, httpReq)
	err = loadAndApplyMeta(meta, req)
	assert.NoError(t, err)
	assert.Equal(t, "bar", receivedRec.Foo)
	assert.Equal(t, 200, rec.Code)
}

func TestMultipart(t *testing.T) {
	type RequestStruct struct {
		*Request
		Test bool                    `form:"test"`
		File *multipart.FileHeader   `form:"file-field" required:"true"`
		Arr  []*multipart.FileHeader `form:"multi-file-field" required:"true"`
	}

	buffer := bytes.Buffer{}
	writer := multipart.NewWriter(&buffer)
	err := writer.WriteField("test", "true")
	require.NoError(t, err)
	w, err := writer.CreateFormFile("file-field", "raggett.txt")
	require.NoError(t, err)
	_, err = w.Write([]byte("this is a test"))
	require.NoError(t, err)
	w, err = writer.CreateFormFile("multi-file-field", "raggett.txt")
	require.NoError(t, err)
	_, err = w.Write([]byte("this is a test"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", bytes.NewReader(buffer.Bytes()))
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	dumpRequest(t, httpReq)
	var r *RequestStruct
	resp, validationError, runtimeError := testMuxPostWith(t, httpReq, "/foo/bar", func(req *RequestStruct) error {
		r = req
		return nil
	})

	require.NoError(t, validationError)
	require.NoError(t, runtimeError)

	require.Equal(t, 204, resp.Code)
	assert.True(t, r.Test)
	assert.NotNil(t, r.File)
	assert.Equal(t, int64(14), r.File.Size)
	assert.Equal(t, "raggett.txt", r.File.Filename)
}

func TestMultipartMultiFiles(t *testing.T) {
	type RequestStruct struct {
		*Request
		Test bool          `form:"test"`
		File []*FileHeader `form:"files-field" required:"true"`
	}

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

	httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", bytes.NewReader(buffer.Bytes()))
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	dumpRequest(t, httpReq)
	var r *RequestStruct
	resp, validationError, runtimeError := testMuxPostWith(t, httpReq, "/foo/bar", func(req *RequestStruct) error {
		r = req
		return nil
	})

	require.NoError(t, validationError)
	require.NoError(t, runtimeError)

	require.Equal(t, 204, resp.Code)
	assert.True(t, r.Test)
	assert.NotNil(t, r.File)
	assert.Equal(t, 2, len(r.File))
	assert.Equal(t, int64(14), r.File[0].Size)
	assert.Equal(t, "raggett1.txt", r.File[0].Filename)
	assert.Equal(t, int64(20), r.File[1].Size)
	assert.Equal(t, "raggett2.txt", r.File[1].Filename)
}

func TestMultipartSingle(t *testing.T) {
	type RequestStruct struct {
		*Request
		Test bool        `form:"test"`
		File *FileHeader `form:"file-field" required:"true"`
	}

	buffer := bytes.Buffer{}
	writer := multipart.NewWriter(&buffer)
	err := writer.WriteField("test", "true")
	require.NoError(t, err)
	w, err := writer.CreateFormFile("file-field", "raggett1.txt")
	require.NoError(t, err)
	_, err = w.Write([]byte("test"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", bytes.NewReader(buffer.Bytes()))
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	dumpRequest(t, httpReq)
	var r RequestStruct
	resp, validationError, runtimeError := testMuxPostWith(t, httpReq, "/foo/bar", func(req RequestStruct) error {
		r = req
		return nil
	})

	require.NoError(t, validationError)
	require.NoError(t, runtimeError)

	require.Equal(t, 204, resp.Code)
	assert.True(t, r.Test)
	assert.NotNil(t, r.File)
	assert.Equal(t, int64(4), r.File.Size)
	assert.Equal(t, "raggett1.txt", r.File.Filename)
}

func TestDecoderError(t *testing.T) {
	t.Parallel()

	t.Run("Invalid JSON", func(t *testing.T) {
		type RequestStruct struct {
			*Request
			Foo map[string]interface{} `body:"json"`
		}

		httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", strings.NewReader("{rrrrrr"))
		rec := httptest.NewRecorder()
		called := false
		handler := func(req *RequestStruct) error {
			called = true
			return nil
		}
		m := NewMux(zap.NewNop())
		m.Post("/foo/bar", handler)

		m.ServeHTTP(rec, httpReq)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.False(t, called)
	})

	t.Run("Invalid int", func(t *testing.T) {
		type RequestStruct struct {
			*Request
			Value int `form:"val"`
		}

		httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", strings.NewReader("val=@)#"))
		httpReq.Header.Add("content-type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		called := false
		handler := func(req *RequestStruct) error {
			called = true
			return nil
		}
		m := NewMux(zap.NewNop())
		m.Post("/foo/bar", handler)

		m.ServeHTTP(rec, httpReq)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.False(t, called)
	})

	t.Run("Invalid uint", func(t *testing.T) {
		type RequestStruct struct {
			*Request
			Value uint `form:"val"`
		}

		httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", strings.NewReader("val=@)#"))
		httpReq.Header.Add("content-type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		called := false
		handler := func(req *RequestStruct) error {
			called = true
			return nil
		}
		m := NewMux(zap.NewNop())
		m.Post("/foo/bar", handler)

		m.ServeHTTP(rec, httpReq)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.False(t, called)
	})

	t.Run("Invalid float32", func(t *testing.T) {
		type RequestStruct struct {
			*Request
			Value float32 `form:"val"`
		}

		httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", strings.NewReader("val=@)#"))
		httpReq.Header.Add("content-type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		called := false
		handler := func(req *RequestStruct) error {
			called = true
			return nil
		}
		m := NewMux(zap.NewNop())
		m.Post("/foo/bar", handler)

		m.ServeHTTP(rec, httpReq)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.False(t, called)
	})

	t.Run("Invalid float64", func(t *testing.T) {
		type RequestStruct struct {
			*Request
			Value float64 `form:"val"`
		}

		httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", strings.NewReader("val=@)#"))
		httpReq.Header.Add("content-type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		called := false
		handler := func(req *RequestStruct) error {
			called = true
			return nil
		}
		m := NewMux(zap.NewNop())
		m.Post("/foo/bar", handler)

		m.ServeHTTP(rec, httpReq)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.False(t, called)
	})

	t.Run("Invalid bool", func(t *testing.T) {
		type RequestStruct struct {
			*Request
			Value bool `form:"val"`
		}

		httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", strings.NewReader("val=@)#"))
		httpReq.Header.Add("content-type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		called := false
		handler := func(req *RequestStruct) error {
			called = true
			return nil
		}
		m := NewMux(zap.NewNop())
		m.Post("/foo/bar", handler)

		m.ServeHTTP(rec, httpReq)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.False(t, called)
	})

	t.Run("Invalid slice", func(t *testing.T) {
		type RequestStruct struct {
			*Request
			Value []bool `form:"val"`
		}

		httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", strings.NewReader("val=@)#"))
		httpReq.Header.Add("content-type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		called := false
		handler := func(req *RequestStruct) error {
			called = true
			return nil
		}
		m := NewMux(zap.NewNop())
		m.Post("/foo/bar", handler)

		m.ServeHTTP(rec, httpReq)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.False(t, called)
	})

	t.Run("Empty value", func(t *testing.T) {
		type RequestStruct struct {
			*Request
			Value bool `form:"val"`
		}

		httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", strings.NewReader("val="))
		httpReq.Header.Add("content-type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		called := false
		handler := func(req *RequestStruct) error {
			called = true
			return nil
		}
		m := NewMux(zap.NewNop())
		m.Post("/foo/bar", handler)

		m.ServeHTTP(rec, httpReq)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.False(t, called)
	})

	t.Run("Empty string by trim", func(t *testing.T) {
		type RequestStruct struct {
			*Request
			Value string `form:"val" blank:"false"`
		}

		httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", strings.NewReader("val=%20%20%20"))
		httpReq.Header.Add("content-type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		called := false
		handler := func(req *RequestStruct) error {
			called = true
			return nil
		}
		m := NewMux(zap.NewNop())
		m.Post("/foo/bar", handler)

		m.ServeHTTP(rec, httpReq)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.False(t, called)
	})

	t.Run("Invalid string by pattern", func(t *testing.T) {
		type RequestStruct struct {
			*Request
			Value string `form:"val" blank:"false" pattern:"^foo$"`
		}

		httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", strings.NewReader("val=bar"))
		httpReq.Header.Add("content-type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		called := false
		handler := func(req *RequestStruct) error {
			called = true
			return nil
		}
		m := NewMux(zap.NewNop())
		m.Post("/foo/bar", handler)

		m.ServeHTTP(rec, httpReq)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.False(t, called)
	})

	t.Run("Invalid file by size", func(t *testing.T) {
		type RequestStruct struct {
			*Request
			File *FileHeader `form:"files-field" blank:"false" required:"true"`
		}

		buffer := bytes.Buffer{}
		writer := multipart.NewWriter(&buffer)
		err := writer.WriteField("test", "true")
		require.NoError(t, err)
		_, err = writer.CreateFormFile("files-field", "raggett1.txt")
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		httpReq := httptest.NewRequest("POST", "/foo/bar?foo=bar", bytes.NewReader(buffer.Bytes()))
		httpReq.Header.Set("Content-Type", writer.FormDataContentType())
		dumpRequest(t, httpReq)
		resp, validationError, runtimeError := testMuxPostWith(t, httpReq, "/foo/bar", func(req RequestStruct) error {
			return nil
		})

		require.Error(t, validationError)
		require.NoError(t, runtimeError)

		require.Equal(t, http.StatusBadRequest, resp.Code)
	})
}
