package raggett

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"reflect"

	"go.uber.org/zap"
)

var (
	jsonContentType       = UTF8MediaTypeFromString("application", "json")
	jsonContentTypeString = jsonContentType.String()
	jsonContentTypeName   = jsonContentType.Type()

	xmlContentType       = UTF8MediaTypeFromString("text", "xml")
	xmlContentTypeString = xmlContentType.String()
	xmlContentTypeName   = xmlContentType.Type()

	htmlContentType       = UTF8MediaTypeFromString("text", "html")
	htmlContentTypeString = htmlContentType.String()
	htmlContentTypeName   = htmlContentType.Type()

	plainTextContentType       = UTF8MediaTypeFromString("text", "plain")
	plainTextContentTypeString = plainTextContentType.String()
	plainTextContentTypeName   = plainTextContentType.Type()

	bytesContentType       = MediaTypeFromString("application", "octet-stream")
	bytesContentTypeString = bytesContentType.String()
	bytesContentTypeName   = bytesContentType.Type()
)

type JSONResponder interface {
	JSON() interface{}
}

type XMLResponder interface {
	XML() interface{}
}

type HTMLResponder interface {
	HTML() string
}

type PlainTextResponder interface {
	PlainText() string
}

type BytesResponder interface {
	Bytes() []byte
}

type CustomResponder interface {
	RespondsToMediaTypes() []MediaType
	HandleMediaTypeResponse(mt MediaType, w io.Writer) error
}

func writeResponder(r *Request, response interface{}) {
	types := map[string]func(){}
	var offers []MediaType
	w := r.httpResponse
	if i, ok := response.(HTMLResponder); ok {
		types[htmlContentTypeName] = func() {
			r.setContentTypeNoOverride(htmlContentTypeString)
			r.flushHeaders()
			data := []byte(i.HTML())
			if _, err := w.Write(data); err != nil {
				r.Logger.Error("Error encoding HTML payload to response", zap.Error(err))
				r.AbortError(err)
			}
		}
		offers = append(offers, htmlContentType)
	}

	if i, ok := response.(JSONResponder); ok {
		types[jsonContentTypeName] = func() {
			r.setContentTypeNoOverride(jsonContentTypeString)
			r.flushHeaders()
			if err := json.NewEncoder(w).Encode(i.JSON()); err != nil {
				r.Logger.Error("Error encoding JSON payload to response", zap.Error(err))
				r.AbortError(err)
			}
		}
		offers = append(offers, jsonContentType)
	}

	if i, ok := response.(XMLResponder); ok {
		types[xmlContentTypeName] = func() {
			r.setContentTypeNoOverride(xmlContentTypeString)
			r.flushHeaders()
			if err := xml.NewEncoder(w).Encode(i.XML()); err != nil {
				r.Logger.Error("Error encoding XML payload to response", zap.Error(err))
				r.AbortError(err)
			}
		}
		offers = append(offers, xmlContentType)
	}

	if i, ok := response.(BytesResponder); ok {
		types[bytesContentTypeName] = func() {
			r.setContentTypeNoOverride(bytesContentTypeString)
			r.flushHeaders()
			if _, err := w.Write(i.Bytes()); err != nil {
				r.Logger.Error("Error encoding HTML payload to response", zap.Error(err))
				r.AbortError(err)
			}
		}
		offers = append(offers, bytesContentType)
	}

	if i, ok := response.(PlainTextResponder); ok {
		types[plainTextContentTypeName] = func() {
			r.setContentTypeNoOverride(plainTextContentTypeString)
			r.flushHeaders()
			if _, err := w.Write([]byte(i.PlainText())); err != nil {
				r.Logger.Error("Error encoding Plain Text payload to response", zap.Error(err))
				r.AbortError(err)
			}
		}
		offers = append(offers, plainTextContentType)
	}

	if i, ok := response.(CustomResponder); ok {
		for _, o := range i.RespondsToMediaTypes() {
			types[o.Type()] = func() {
				buf := &bytes.Buffer{}
				if err := i.HandleMediaTypeResponse(o, buf); err != nil {
					r.Logger.Error("Error invoking HandleMediaTypeResponse for CustomResponder",
						zap.String("responder", reflect.TypeOf(i).Name()),
						zap.String("content_type", o.Type()),
						zap.Error(err))
					r.AbortError(err)
				}

				r.setContentTypeNoOverride(o.String())
				r.flushHeaders()

				if _, err := io.Copy(r.httpResponse, buf); err != nil {
					r.Logger.Error("Error copying custom payload to response",
						zap.String("responder", reflect.TypeOf(i).Name()),
						zap.String("content_type", o.Type()),
						zap.Error(err))
					r.AbortError(err)
				}
			}
			offers = append(offers, o)
		}
	}

	if len(offers) == 0 {
		if r.mux.Development {
			panic(fmt.Sprintf("Could not determine how to respond to request using object %#v", response))
		}
		panic(makeError(fmt.Errorf("could not determine how to respond to request using object %#v", response)))
	}

	parsed, selected, media := NegotiateContentTypeWithMediaTypes(r.HTTPRequest, offers)

	// TODO: Handle when parsed/selected is not ok

	_, _ = parsed, selected

	if !r.setContentType {
		w.Header().Set("Content-Type", media.String())
	}

	types[media.Type()]()
}
