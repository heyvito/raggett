package raggett

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRequest_SetStatus(t *testing.T) {
	r := NewRequest(nil, nil)
	r.SetStatus(http.StatusAccepted)
	assert.True(t, r.statusSet)
	assert.Equal(t, http.StatusAccepted, r.responseStatus)
}

func TestRequest_RespondJSON(t *testing.T) {
	r := NewRequest(nil, nil)
	r.RespondJSON(true)
	v, ok := r.response.(*jsonResponse)
	assert.True(t, ok)
	d, ok := v.data.(bool)
	assert.True(t, ok)
	assert.True(t, d)
}

func TestRequest_RespondXML(t *testing.T) {
	r := NewRequest(nil, nil)
	r.RespondXML(true)
	v, ok := r.response.(*xmlResponse)
	assert.True(t, ok)
	d, ok := v.data.(bool)
	assert.True(t, ok)
	assert.True(t, d)
}

func TestRequest_RespondReader(t *testing.T) {
	reader := io.NopCloser(strings.NewReader("hello"))
	r := NewRequest(nil, nil)
	r.RespondReader(reader)
	v, ok := r.response.(*fileResponse)
	assert.True(t, ok)
	assert.Equal(t, reader, v.file)
}

func TestRequest_RespondString(t *testing.T) {
	r := NewRequest(nil, nil)
	r.RespondString("foo")
	v, ok := r.response.(*stringResponse)
	assert.True(t, ok)
	assert.Equal(t, "foo", v.data)
}

func TestRequest_RespondBytes(t *testing.T) {
	r := NewRequest(nil, nil)
	r.RespondBytes([]byte("foo"))
	v, ok := r.response.(*bytesResponse)
	assert.True(t, ok)
	assert.Equal(t, []byte("foo"), v.response)
}

func TestRequest_SetContentType(t *testing.T) {
	w := httptest.NewRecorder()
	r := NewRequest(w, nil)
	r.SetContentType("text/plain")
	assert.True(t, r.setContentType)
	assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
}

func TestRequest_AddHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r := NewRequest(w, nil)
	r.AddHeader("foo", "bar")
	r.AddHeader("foo", "baz")
	assert.Equal(t, http.Header(map[string][]string{"Foo": {"bar", "baz"}}), w.Header())
}

func TestRequest_GetCookie(t *testing.T) {
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest("POST", "/test", nil)
	httpReq.Header.Add("Cookie", "yummy_cookie=choco; tasty_cookie=strawberry")
	r := NewRequest(w, httpReq)
	choco, ok := r.GetCookie("yummy_cookie")
	assert.True(t, ok)
	assert.Equal(t, "yummy_cookie", choco.Name)
	assert.Equal(t, "choco", choco.Value)
	straw, ok := r.GetCookie("tasty_cookie")
	assert.True(t, ok)
	assert.Equal(t, "tasty_cookie", straw.Name)
	assert.Equal(t, "strawberry", straw.Value)

	// No stale cookies!
	stale, ok := r.GetCookie("stale_cookie")
	assert.False(t, ok)
	assert.Nil(t, stale)
}

func TestRequest_AddCookie(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := NewRequest(w, nil)
		noPtrCookie := Cookie("tasty_cookie", "strawberry").ExpiresIn(12 * time.Hour)
		commonCookie := http.Cookie{Name: "sweet_cookie", Value: "vanilla"}
		otherCookie := &http.Cookie{Name: "yummy_cookie", Value: "choco"}

		r.AddCookie(*noPtrCookie)
		r.AddCookie(commonCookie)
		r.AddCookie(otherCookie)
		r.AddCookie(Cookie("stale_cookie", "").ExpiresNow())

		assert.Equal(t, http.Header(map[string][]string{
			"Set-Cookie": {
				"tasty_cookie=strawberry; Max-Age=43200",
				"sweet_cookie=vanilla",
				"yummy_cookie=choco",
				"stale_cookie=; Max-Age=0",
			},
		}), w.Header())
	})

	t.Run("No nil cookies", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := NewRequest(w, nil)
		assert.PanicsWithValue(t, "nil cookie provided to AddCookie", func() {
			r.AddCookie(nil)
		})
	})

	t.Run("No weird cookies", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := NewRequest(w, nil)
		assert.PanicsWithValue(t, "Unexpected value passed to AddCookie. Use http.Cookie or the result of ragget.Cookie(name, value).", func() {
			r.AddCookie(1)
		})
	})
}

func TestRequest_Context(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r := NewRequest(w, req)
	assert.Equal(t, req.Context(), r.Context())
}

func TestRequest_Abort(t *testing.T) {
	r := NewRequest(nil, nil)
	assert.PanicsWithError(t, errAbortRequest.Error(), r.Abort)
}

func TestRequest_NotFound(t *testing.T) {
	r := NewRequest(nil, nil)
	assert.PanicsWithError(t, errAbortNotFound.Error(), r.NotFound)
}

func TestRequest_ClientAccepts(t *testing.T) {
	httpReq := httptest.NewRequest("GET", "/", nil)
	httpReq.Header.Set("Accept", "foo/bar")
	r := NewRequest(nil, httpReq)
	accept := r.ClientAccepts()
	require.Len(t, accept, 1)
	media := accept[0]
	assert.Equal(t, MediaTypeSpecificityFullyDefined, media.Specificity)
	assert.Equal(t, "foo", media.TypeString)
	assert.Equal(t, "bar", media.SubTypeString)
	assert.NotNil(t, r.acceptsMemo)
	assert.Equal(t, accept, r.ClientAccepts())
}

type customJSONEncoder struct{}

func (c *customJSONEncoder) RespondsToMediaTypes() []MediaType {
	return []MediaType{MediaTypeFromString("application", "json")}
}

func (c *customJSONEncoder) HandleMediaTypeResponse(mt MediaType, w io.Writer) error {
	if mt.Type() == "application/json" {
		return json.NewEncoder(w).Encode(map[string]interface{}{"test": "yep"})
	} else {
		return fmt.Errorf("don't know that ditty: %s", mt.String())
	}
}

func TestRequest_Redirect(t *testing.T) {
	m := NewMux(zap.NewNop())
	m.Get("/", func(r *EmptyRequest) error {
		r.Redirect("https://example.org")
		return nil
	})
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest("GET", "/", nil)
	m.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "https://example.org", w.Header().Get("Location"))
}

func TestRequest_DoRespond(t *testing.T) {
	t.Parallel()
	httpReq := httptest.NewRequest("GET", "/", nil)
	t.Run("Without body", func(t *testing.T) {
		m := NewMux(zap.NewNop())
		m.Get("/", func(r *EmptyRequest) error {
			return nil
		})
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httpReq)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("jsonResponse", func(t *testing.T) {
		m := NewMux(zap.NewNop())
		m.Get("/", func(r *EmptyRequest) error {
			r.RespondJSON(map[string]string{"test": "yep"})
			return nil
		})

		w := httptest.NewRecorder()
		m.ServeHTTP(w, httpReq)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, jsonContentTypeString, w.Header().Get("Content-Type"))
		assert.Equal(t, "{\"test\":\"yep\"}\n", w.Body.String())
	})

	t.Run("xmlResponse", func(t *testing.T) {
		m := NewMux(zap.NewNop())
		m.Get("/", func(r *EmptyRequest) error {
			r.RespondXML(struct {
				XMLName xml.Name `xml:"foo"`
				Bar     string   `xml:"bar"`
			}{Bar: "baz"})
			return nil
		})

		w := httptest.NewRecorder()
		m.ServeHTTP(w, httpReq)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, xmlContentTypeString, w.Header().Get("Content-Type"))
		assert.Equal(t, "<foo><bar>baz</bar></foo>", w.Body.String())
	})

	t.Run("fileResponse", func(t *testing.T) {
		m := NewMux(zap.NewNop())
		m.Get("/", func(r *EmptyRequest) error {
			r.RespondReader(io.NopCloser(strings.NewReader("test")))
			return nil
		})

		w := httptest.NewRecorder()
		m.ServeHTTP(w, httpReq)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, bytesContentTypeString, w.Header().Get("Content-Type"))
		assert.Equal(t, "test", w.Body.String())
	})

	t.Run("bytesResponse", func(t *testing.T) {
		m := NewMux(zap.NewNop())
		m.Get("/", func(r *EmptyRequest) error {
			r.RespondBytes([]byte("test"))
			return nil
		})

		w := httptest.NewRecorder()
		m.ServeHTTP(w, httpReq)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, bytesContentTypeString, w.Header().Get("Content-Type"))
		assert.Equal(t, "test", w.Body.String())
	})

	t.Run("stringResponse", func(t *testing.T) {
		m := NewMux(zap.NewNop())
		m.Get("/", func(r *EmptyRequest) error {
			r.RespondString("test")
			return nil
		})

		w := httptest.NewRecorder()
		m.ServeHTTP(w, httpReq)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, plainTextContentTypeString, w.Header().Get("Content-Type"))
		assert.Equal(t, "test", w.Body.String())
	})

	t.Run("default case with JSON responder", func(t *testing.T) {
		m := NewMux(zap.NewNop())
		m.Get("/", func(r *EmptyRequest) error {
			r.SetStatus(http.StatusCreated)
			r.Respond(&customJSONEncoder{})
			return nil
		})

		w := httptest.NewRecorder()
		m.ServeHTTP(w, httpReq)
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, "{\"test\":\"yep\"}\n", w.Body.String())
	})
}

var errStreamWrite = fmt.Errorf("error writing to stream")

type brokenWriter struct {
	header http.Header
	status int
}

func (b *brokenWriter) Header() http.Header       { return b.header }
func (b *brokenWriter) Write([]byte) (int, error) { return 0, errStreamWrite }
func (b *brokenWriter) WriteHeader(s int)         { b.status = s }

type dummyJSONResponder struct{}

func (d dummyJSONResponder) JSON() interface{} {
	return map[string]interface{}{"hello": "test"}
}

func TestEncoderFailure(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	t.Run("jsonResponse", func(t *testing.T) {
		writer := &brokenWriter{header: map[string][]string{}}
		m := NewMux(zap.NewNop())
		m.Get("/", func(r EmptyRequest) error {
			r.RespondJSON(map[string]interface{}{"hello": "test"})
			return nil
		})

		m.ServeHTTP(writer, req)
		assert.Equal(t, http.StatusOK, writer.status)
	})

	t.Run("xmlResponse", func(t *testing.T) {
		writer := &brokenWriter{header: map[string][]string{}}
		m := NewMux(zap.NewNop())
		m.Get("/", func(r EmptyRequest) error {
			r.RespondXML(struct {
				XMLName xml.Name `xml:"test"`
				Hello   string   `xml:"hello"`
			}{Hello: "test"})
			return nil
		})

		m.ServeHTTP(writer, req)
		assert.Equal(t, http.StatusOK, writer.status)
	})

	t.Run("fileResponse", func(t *testing.T) {
		writer := &brokenWriter{header: map[string][]string{}}
		m := NewMux(zap.NewNop())
		m.Get("/", func(r EmptyRequest) error {
			r.RespondReader(io.NopCloser(strings.NewReader("hello")))
			return nil
		})

		m.ServeHTTP(writer, req)
		assert.Equal(t, http.StatusOK, writer.status)
	})

	t.Run("bytesResponse", func(t *testing.T) {
		writer := &brokenWriter{header: map[string][]string{}}
		m := NewMux(zap.NewNop())
		m.Get("/", func(r EmptyRequest) error {
			r.RespondBytes([]byte("hello"))
			return nil
		})

		m.ServeHTTP(writer, req)
		assert.Equal(t, http.StatusOK, writer.status)
	})

	t.Run("stringResponse", func(t *testing.T) {
		writer := &brokenWriter{header: map[string][]string{}}
		m := NewMux(zap.NewNop())
		m.Get("/", func(r EmptyRequest) error {
			r.RespondString("hello")
			return nil
		})

		m.ServeHTTP(writer, req)
		assert.Equal(t, http.StatusOK, writer.status)
	})

	t.Run("genericResponder", func(t *testing.T) {
		writer := &brokenWriter{header: map[string][]string{}}
		m := NewMux(zap.NewNop())
		m.Get("/", func(r EmptyRequest) error {
			r.Respond(dummyJSONResponder{})
			return nil
		})

		m.ServeHTTP(writer, req)
		assert.Equal(t, http.StatusOK, writer.status)
	})
}

type brokenReadCloser struct{}

func (b *brokenReadCloser) Read(p []byte) (n int, err error) {
	p[0] = 'a'
	return 1, io.EOF
}

func (b *brokenReadCloser) Close() error { return fmt.Errorf("boom") }

func TestStreamCloseError(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	t.Run("jsonResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		m := NewMux(zap.NewNop())
		m.Get("/", func(r EmptyRequest) error {
			assert.Equal(t, "*/*", r.ClientAccepts()[0].Type())
			r.RespondReader(&brokenReadCloser{})
			return nil
		})

		m.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, []byte("a"), w.Body.Bytes())
	})
}

func TestWriteError(t *testing.T) {
	// Here we want to flush headers and try to encode something right after.

	t.Run("Runtime Error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		w := httptest.NewRecorder()
		m := NewMux(zap.NewNop())
		m.Get("/", func(r EmptyRequest) error {
			r.SetStatus(http.StatusNoContent)
			r.flushHeaders()
			panic("boom")
		})

		m.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body)
	})

	t.Run("Validation Error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		w := httptest.NewRecorder()
		m := NewMux(zap.NewNop())
		m.Get("/", func(r EmptyRequest) error {
			r.SetStatus(http.StatusNoContent)
			r.flushHeaders()
			return ValidationError{
				StructName:      "foo",
				StructFieldName: "foo",
				FieldName:       "foo",
				FieldKind:       fieldKindHeader,
				ErrorKind:       ValidationErrorKindParsing,
				OriginalError:   nil,
			}
		})

		m.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body)
	})
}

func TestSetHeaderAfterFlush(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	w := httptest.NewRecorder()
	m := NewMux(zap.NewNop())
	m.Get("/", func(r EmptyRequest) error {
		r.SetStatus(http.StatusNoContent)
		r.flushHeaders()
		r.SetHeader("foo", "bar")
		r.SetContentType("foo/bar")
		r.AddHeader("test", "test")
		return nil
	})

	m.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Header().Get("foo"))
	assert.Empty(t, w.Body)
}

func TestResponseWithWrongStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	w := httptest.NewRecorder()
	m := NewMux(zap.NewNop())
	m.Get("/", func(r EmptyRequest) error {
		r.SetStatus(http.StatusNoContent)
		r.RespondString("hello!")
		return nil
	})

	m.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body)
}

func TestResponseDoubleFlush(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	w := httptest.NewRecorder()
	m := NewMux(zap.NewNop())
	m.Get("/", func(r EmptyRequest) error {
		r.SetStatus(http.StatusNoContent)
		r.flushHeaders()
		r.SetStatus(http.StatusOK)
		r.RespondString("hello!")
		return nil
	})

	m.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body)
}

func TestResponseWithCustomStatusNoBody(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "application/json")

	w := httptest.NewRecorder()
	m := NewMux(zap.NewNop())
	m.Get("/", func(r EmptyRequest) error {
		r.SetStatus(http.StatusCreated)
		return nil
	})

	m.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Empty(t, w.Body)
}
