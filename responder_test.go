package raggett

import (
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestErrorResponder(t *testing.T) {
	mx := NewMux(zap.NewNop())
	type handler struct {
		*Request
	}
	mx.Get("/", func(r *handler) error {
		return Error{
			StackTrace:    getStack(1),
			OriginalError: fmt.Errorf("boom"),
		}
	})
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mx.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)
}

type bytesResponderTest struct{}

func (b bytesResponderTest) Bytes() []byte {
	return []byte("hello")
}

func TestBytesResponder(t *testing.T) {
	mx := NewMux(zap.NewNop())
	type handler struct {
		*Request
	}
	mx.Get("/", func(r *handler) error {
		r.Respond(bytesResponderTest{})
		return nil
	})
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mx.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, bytesContentTypeString, w.Header().Get("Content-Type"))
	assert.Equal(t, []byte("hello"), w.Body.Bytes())
}

type brokenCustomResponder struct{}

func (b brokenCustomResponder) RespondsToMediaTypes() []MediaType {
	return []MediaType{{TypeString: "test", SubTypeString: "foo"}}
}

func (b brokenCustomResponder) HandleMediaTypeResponse(MediaType, io.Writer) error {
	return fmt.Errorf("boom")
}

func TestBrokenCustomResponder(t *testing.T) {
	mx := NewMux(zap.NewNop())
	type handler struct {
		*Request
	}
	mx.Get("/", func(r *handler) error {
		r.Respond(brokenCustomResponder{})
		return nil
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Add("Content-Type", "test/foo")
	w := httptest.NewRecorder()
	mx.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)
}

func TestNoOffers(t *testing.T) {
	mx := NewMux(zap.NewNop())
	type handler struct {
		*Request
	}
	mx.Get("/", func(r *handler) error {
		r.Respond(struct{}{})
		return nil
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Add("Content-Type", "test/test")
	w := httptest.NewRecorder()
	mx.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)
}
