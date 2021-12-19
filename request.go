package raggett

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type jsonResponse struct {
	data interface{}
}

type fileResponse struct {
	file io.ReadCloser
}

type bytesResponse struct {
	response []byte
}

type xmlResponse struct {
	data interface{}
}

// EmptyRequest is a convenience type for a struct containing only a *Request
// field.
type EmptyRequest struct {
	*Request
}

// Request represents a Raggett request. This struct provides useful
// abstractions for handling requests and responses.
type Request struct {
	// HTTPRequest represents the original http.Request received by the server
	HTTPRequest *http.Request
	// Logger represents the logger for this specific request. Logs emitted
	// through it will contain the generated unique request ID for tracing.
	Logger *zap.Logger

	httpResponse   http.ResponseWriter
	responseStatus int
	response       interface{}
	mux            *Mux
	maxMemory      int64
	requestID      string
	statusSet      bool
	acceptsMemo    []MediaType
	setContentType bool
	flushedHeaders bool
}

// NewRequest creates a new request with an empty mux. This method is intended
// for testing purposes.
func NewRequest(w http.ResponseWriter, r *http.Request) *Request {
	return newRequest(nil, w, r)
}

func newRequest(mux *Mux, w http.ResponseWriter, r *http.Request) *Request {
	if mux == nil {
		mux = &Mux{
			logger:              zap.NewNop(),
			MaxMemory:           defaultMaxMemory,
			identifierGenerator: defaultRequestIDGenerator,
		}
	}

	id := mux.identifierGenerator()
	logger := mux.logger.With(zap.String("request-id", id))

	return &Request{
		HTTPRequest:    r,
		httpResponse:   w,
		responseStatus: 200,
		Logger:         logger,
		maxMemory:      mux.MaxMemory,
		requestID:      id,
		mux:            mux,
	}
}

// SetStatus defines which HTTP Status will be returned to the client. This
// method does not write headers to the client.
func (r *Request) SetStatus(httpStatus int) {
	r.responseStatus = httpStatus
	r.statusSet = true
}

// RespondJSON responds the provided data to the client using a JSON
// Content-Type. The value will be written to the client once the handler
// function returns.
// The provided value must be compatible with encoding/json marshaller.
func (r *Request) RespondJSON(data interface{}) {
	r.response = &jsonResponse{data: data}
}

// RespondXML responds the provided data to the client using an XML
// Content-Type. The value will be written to the client once the handler
// function returns.
// The provided value must be compatible with encoding/xml marshaller.
func (r *Request) RespondXML(data interface{}) {
	r.response = &xmlResponse{data: data}
}

// RespondReader responds the provided io.ReadCloser to the client using an
// application/octet-stream Content-Type (unless SetContentType is used).
// Once the handler function returns, contents of the provided reader are
// copied to the response stream, and the reader is automatically closed.
func (r *Request) RespondReader(file io.ReadCloser) {
	r.response = &fileResponse{file: file}
}

// RespondString returns a provided string to the client as the response body.
// The contents will be sent to the client once the handler function returns.
func (r *Request) RespondString(str string) {
	r.RespondBytes([]byte(str))
}

// RespondBytes returns a provided byte slice to the client as the response
// body. The contents will be sent to the client once the handler function
// returns.
func (r *Request) RespondBytes(buffer []byte) {
	r.response = &bytesResponse{response: buffer}
}

// SetContentType defines the value for the Content-Type header for this
// request's response. Calling this function prevents Raggett from automatically
// inferring the response's Content-Type.
func (r *Request) SetContentType(contentType string) {
	r.SetHeader("content-type", contentType)
	r.setContentType = true
}

// SetHeader invokes http.Header.Set for the current request's response.
func (r *Request) SetHeader(name, value string) {
	r.httpResponse.Header().Set(name, value)
}

// AddHeader invokes http.Header.Add for the current request's response.
func (r *Request) AddHeader(name, value string) {
	r.httpResponse.Header().Add(name, value)
}

// Abort immediately cancels the current Request
func (r *Request) Abort() {
	r.AbortError(errAbortRequest)
}

// NotFound aborts the current request and returns a NotFound error to the
// client.
func (r *Request) NotFound() {
	r.AbortError(errAbortNotFound)
}

// AbortError aborts the current request with a provided error.
func (r *Request) AbortError(err error) {
	panic(err)
}

// Respond sets the response value for this Request to the provided value. Use
// it with structs implementing JSONResponder, XMLResponder, HTMLResponder,
// PlainTextResponder, BytesResponder, and/or CustomResponder. By providing a
// struct implementing one or more responders allows Ragget to automatically
// negotiate which representation will be returned to the client.
func (r *Request) Respond(value interface{}) {
	r.response = value
}

func (r *Request) setContentTypeNoOverride(value string) {
	if !r.statusSet {
		r.httpResponse.Header().Set("Content-Type", value)
	}
}

var defaultMediaType = []MediaType{MediaTypeFromString("*", "*")}

// ClientAccepts parses the contents of the Accept header provided by the client
// and returns a slice of MediaType structs.
func (r *Request) ClientAccepts() []MediaType {
	if r.acceptsMemo != nil {
		return r.acceptsMemo
	}
	ok, head := parseAcceptHeader(r.HTTPRequest.Header.Get("Accept"))
	if !ok {
		head = defaultMediaType
	}
	r.acceptsMemo = head
	return head
}

func (r *Request) flushHeaders() {
	r.httpResponse.WriteHeader(r.statusForRequest())
	r.flushedHeaders = true
}

func (r *Request) doRespond() {
	if r.response == nil {
		r.flushHeaders()
		return
	}

	switch v := r.response.(type) {
	case *jsonResponse:
		r.setContentTypeNoOverride(jsonContentTypeString)
		r.flushHeaders()
		err := json.NewEncoder(r.httpResponse).Encode(v.data)
		if err != nil {
			r.Logger.Error("raggett: Failed to encode JSON data", zap.Error(err))
			return
		}

	case *xmlResponse:
		r.setContentTypeNoOverride(xmlContentTypeString)
		r.flushHeaders()
		err := xml.NewEncoder(r.httpResponse).Encode(v.data)
		if err != nil {
			r.Logger.Error("raggett: Failed to encode XML data", zap.Error(err))
			return
		}

	case *fileResponse:
		r.setContentTypeNoOverride(bytesContentTypeString)
		r.flushHeaders()
		if _, err := io.Copy(r.httpResponse, v.file); err != nil {
			r.Logger.Error("raggett: Failed to copy file contents to response", zap.Error(err))
			return
		}
		if err := v.file.Close(); err != nil {
			r.Logger.Error("raggett: Failed to close file stream", zap.Error(err))
			return
		}

	case *bytesResponse:
		r.setContentTypeNoOverride(bytesContentTypeString)
		r.flushHeaders()
		if _, err := r.httpResponse.Write(v.response); err != nil {
			r.Logger.Error("raggett: Failed to write bytes to response", zap.Error(err))
			return
		}

	default:
		writeResponder(r, r.response)
	}
}
