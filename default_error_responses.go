package raggett

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/go-chi/chi"

	"github.com/heyvito/raggett/templates"
)

type oneToManyMap map[string][]string

func (otm oneToManyMap) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type oneToMany struct {
		XMLName xml.Name `xml:"item"`
		Name    string   `xml:"name"`
		Values  []string `xml:"value"`
	}
	oneToManyVal := make([]oneToMany, 0, len(otm))
	for k, v := range otm {
		st := oneToMany{
			Name:   k,
			Values: make([]string, len(v)),
		}
		for i, v := range v {
			st.Values[i] = v
		}
		oneToManyVal = append(oneToManyVal, st)
	}
	return e.EncodeElement(oneToManyVal, start)
}

type oneToOneMap map[string]string

func (oto oneToOneMap) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type oneToMany struct {
		XMLName xml.Name `xml:"item"`
		Name    string   `xml:"name"`
		Value   string   `xml:"value"`
	}
	type oneToManyWrapper struct {
		XMLName xml.Name    `xml:"environments"`
		Items   []oneToMany `xml:"item"`
	}

	oneToManyVal := make([]oneToMany, 0, len(oto))
	for k, v := range oto {
		st := oneToMany{
			Name:  k,
			Value: v,
		}
		oneToManyVal = append(oneToManyVal, st)
	}
	return e.EncodeElement(oneToManyWrapper{Items: oneToManyVal}, start)
}

type errorTemplate struct {
	XMLName        xml.Name     `json:"-" xml:"error"`
	Code           int          `json:"code,omitempty" xml:"code"`
	StatusName     string       `json:"status_name,omitempty" xml:"status_name"`
	Method         string       `json:"method,omitempty" xml:"method"`
	Path           string       `json:"path,omitempty" xml:"path"`
	ErrorType      string       `json:"error_type,omitempty" xml:"error_type"`
	ErrorPackage   string       `json:"error_package,omitempty" xml:"error_package"`
	StackTrace     string       `json:"stack_trace,omitempty" xml:"stack_trace"`
	Headers        oneToManyMap `json:"headers,omitempty" xml:"headers"`
	RequestDetails requestInfo  `json:"request_details" xml:"request_details"`
	Environment    oneToOneMap  `json:"environment,omitempty" xml:"environments"`
	Message        string       `json:"message,omitempty" xml:"message"`
}

type validationErrorTemplate struct {
	XMLName        xml.Name     `json:"-" xml:"validation_error"`
	Code           int          `json:"code,omitempty" xml:"code"`
	StatusName     string       `json:"status_name,omitempty" xml:"status_name"`
	Method         string       `json:"method,omitempty" xml:"method"`
	Path           string       `json:"path,omitempty" xml:"path"`
	ErrorType      string       `json:"error_type,omitempty" xml:"error_type"`
	ErrorPackage   string       `json:"error_package,omitempty" xml:"error_package"`
	Message        string       `json:"message,omitempty" xml:"message"`
	Headers        oneToManyMap `json:"headers,omitempty" xml:"headers"`
	RequestDetails requestInfo  `json:"request_details" xml:"request_details"`
	Environment    oneToOneMap  `json:"environment,omitempty" xml:"environments"`

	StructName       string `json:"struct_name,omitempty" xml:"struct_name"`
	StructField      string `json:"struct_field,omitempty" xml:"struct_field"`
	RequestFieldName string `json:"request_field_name,omitempty" xml:"request_field_name"`
	FieldSource      string `json:"field_source,omitempty" xml:"field_source"`
	ErrorKind        string `json:"error_kind,omitempty" xml:"error_kind"`
	OriginalError    string `json:"original_error,omitempty" xml:"original_error"`
}

type constrainedValidationErrorTemplate struct {
	Code      int    `json:"code,omitempty" xml:"code"`
	Message   string `json:"message,omitempty" xml:"message"`
	RequestID string `json:"request_id,omitempty" xml:"request_id"`
}

type constrainedErrorTemplate struct {
	Code       int    `json:"code,omitempty" xml:"code"`
	StatusName string `json:"status_name,omitempty" xml:"status_name"`
	RequestID  string `json:"request_id,omitempty" xml:"request_id"`
}

type requestInfo struct {
	Queries oneToManyMap `json:"queries,omitempty" xml:"queries"`
	Form    oneToManyMap `json:"form,omitempty" xml:"form"`
	Files   oneToManyMap `json:"files,omitempty" xml:"files"`
}

type notFoundTemplate struct {
	XMLName        xml.Name      `json:"-" xml:"error"`
	Code           int           `json:"code,omitempty" xml:"code"`
	StatusName     string        `json:"status_name,omitempty" xml:"status_name"`
	Method         string        `json:"method,omitempty" xml:"method"`
	Path           string        `json:"path,omitempty" xml:"path"`
	Headers        oneToManyMap  `json:"headers,omitempty" xml:"headers"`
	RequestDetails requestInfo   `json:"request_details" xml:"request_details"`
	Environment    oneToOneMap   `json:"environment,omitempty" xml:"environments"`
	Routes         routeInfoList `json:"routes,omitempty" xml:"routes"`
}

type routeInfoList []routeInfo

func (r routeInfoList) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type routeInfoWrapper struct {
		XMLName xml.Name    `xml:"routes"`
		Route   []routeInfo `xml:"route"`
	}

	return e.EncodeElement(routeInfoWrapper{Route: r}, start)
}

type routeInfo struct {
	Method  string `json:"method,omitempty" xml:"method"`
	Pattern string `json:"pattern,omitempty" xml:"pattern"`
	Handler string `json:"handler,omitempty" xml:"handler"`
}

type notFoundConstrainedTemplate struct {
	XMLName    xml.Name `json:"-" xml:"not_found"`
	Code       int      `json:"code,omitempty" xml:"code"`
	Message    string   `json:"message,omitempty" xml:"message"`
	StatusName string   `json:"status_name,omitempty" xml:"status_name"`
}

var notFoundConstrainedValue = notFoundConstrainedTemplate{
	Code:       http.StatusNotFound,
	StatusName: http.StatusText(http.StatusNotFound),
	Message:    "The requested resource was not found.",
}

var methodNotAllowedConstrainedValue = notFoundConstrainedTemplate{
	Code:       http.StatusMethodNotAllowed,
	StatusName: http.StatusText(http.StatusMethodNotAllowed),
	Message:    "This endpoint does not allow this HTTP method.",
}

func validationErrorToTemplate(r *Request, err ValidationError, status int) validationErrorTemplate {
	environment := map[string]string{}
	for _, e := range os.Environ() {
		comps := strings.SplitN(e, "=", 2)
		environment[comps[0]] = comps[1]
	}

	errType := reflect.TypeOf(err)

	tmpl := validationErrorTemplate{
		Code:         status,
		StatusName:   http.StatusText(status),
		Method:       r.HTTPRequest.Method,
		Path:         r.HTTPRequest.URL.Path,
		ErrorType:    errType.Name(),
		ErrorPackage: errType.PkgPath(),
		Message:      err.Error(),
		Headers:      oneToManyMap(r.HTTPRequest.Header),
		RequestDetails: requestInfo{
			Queries: oneToManyMap(r.HTTPRequest.URL.Query()),
			Form:    oneToManyMap(r.HTTPRequest.PostForm),
		},
		Environment:      environment,
		StructName:       err.StructName,
		StructField:      err.StructFieldName,
		RequestFieldName: err.FieldName,
		FieldSource:      err.FieldKind.String(),
		ErrorKind:        err.ErrorKind.Name(),
	}

	if err.OriginalError != nil {
		tmpl.OriginalError = err.OriginalError.Error()
	}

	if r.HTTPRequest.MultipartForm != nil {
		for k, fs := range r.HTTPRequest.MultipartForm.File {
			if tmpl.RequestDetails.Files == nil {
				tmpl.RequestDetails.Files = map[string][]string{}
			}
			for _, f := range fs {
				tmpl.RequestDetails.Files[k] = append(tmpl.RequestDetails.Files[k],
					fmt.Sprintf("%s (Size: %d)", f.Filename, f.Size))
			}
		}
	}

	return tmpl
}

func validationErrorToConstrainedTemplate(r *Request, err ValidationError, status int) constrainedValidationErrorTemplate {
	return constrainedValidationErrorTemplate{
		Code:      status,
		Message:   err.Error(),
		RequestID: r.requestID,
	}
}

func errorToConstrainedTemplate(r *Request, status int) constrainedErrorTemplate {
	return constrainedErrorTemplate{
		Code:       status,
		StatusName: http.StatusText(status),
		RequestID:  r.requestID,
	}
}

func errorToTemplate(r *Request, err error, status int) errorTemplate {
	environment := map[string]string{}
	for _, e := range os.Environ() {
		comps := strings.SplitN(e, "=", 2)
		environment[comps[0]] = comps[1]
	}

	tmpl := errorTemplate{
		Code:         status,
		StatusName:   http.StatusText(status),
		Method:       r.HTTPRequest.Method,
		Path:         r.HTTPRequest.URL.Path,
		ErrorType:    "",
		ErrorPackage: "",
		StackTrace:   "«Stack Trace not Available»",
		Headers:      oneToManyMap(r.HTTPRequest.Header),
		RequestDetails: requestInfo{
			Queries: oneToManyMap(r.HTTPRequest.URL.Query()),
			Form:    oneToManyMap(r.HTTPRequest.PostForm),
		},
		Environment: environment,
	}

	if ragErr, ok := err.(Error); ok {
		tmpl.StackTrace = ragErr.Stack()
		err = ragErr.OriginalError
	}

	errType := reflect.TypeOf(err)

	tmpl.ErrorType = errType.Name()
	tmpl.ErrorPackage = errType.PkgPath()
	tmpl.Message = err.Error()
	if r.HTTPRequest.MultipartForm != nil {
		for k, fs := range r.HTTPRequest.MultipartForm.File {
			if tmpl.RequestDetails.Files == nil {
				tmpl.RequestDetails.Files = map[string][]string{}
			}
			for _, f := range fs {
				tmpl.RequestDetails.Files[k] = append(tmpl.RequestDetails.Files[k],
					fmt.Sprintf("%s (Size: %d)", f.Filename, f.Size))
			}
		}
	}
	return tmpl
}

func renderErrorTemplate(r *Request, err error, status int, name string) (string, error) {
	tmpl := errorToTemplate(r, err, status)
	return templates.TemplateNamed(name)(tmpl)
}

func renderValidationErrorTemplate(r *Request, err ValidationError, status int, name string) (string, error) {
	tmpl := validationErrorToTemplate(r, err, status)
	return templates.TemplateNamed(name)(tmpl)
}

func renderConstrainedErrorTemplate(r *Request, status int, name string) (string, error) {
	tmpl := errorToConstrainedTemplate(r, status)
	return templates.TemplateNamed(name)(tmpl)
}

func renderConstrainedValidationTemplate(r *Request, err ValidationError, status int, name string) (string, error) {
	tmpl := validationErrorToConstrainedTemplate(r, err, status)
	return templates.TemplateNamed(name)(tmpl)
}

func renderHTMLErrorTemplate(r *Request, err error, status int) (string, error) {
	return renderErrorTemplate(r, err, status, templates.ServerErrorHTML)
}

func renderTextErrorTemplate(r *Request, err error, status int) (string, error) {
	return renderErrorTemplate(r, err, status, templates.ServerErrorText)
}

func renderHTMLValidationErrorTemplate(r *Request, err ValidationError, status int) (string, error) {
	return renderValidationErrorTemplate(r, err, status, templates.ValidationErrorHTML)
}

func renderTextValidationErrorTemplate(r *Request, err ValidationError, status int) (string, error) {
	return renderValidationErrorTemplate(r, err, status, templates.ValidationErrorText)
}

func renderConstrainedTextValidationErrorTemplate(r *Request, err ValidationError, status int) (string, error) {
	return renderConstrainedValidationTemplate(r, err, status, templates.ValidationErrorConstrainedText)
}

func renderConstrainedHTMLValidationErrorTemplate(r *Request, err ValidationError, status int) (string, error) {
	return renderConstrainedValidationTemplate(r, err, status, templates.ValidationErrorConstrainedHTML)
}

func renderConstrainedTextErrorTemplate(r *Request, status int) (string, error) {
	return renderConstrainedErrorTemplate(r, status, templates.ServerErrorConstrainedText)
}

func renderConstrainedHTMLErrorTemplate(r *Request, status int) (string, error) {
	return renderConstrainedErrorTemplate(r, status, templates.ServerErrorConstrainedHTML)
}

func listRoutes(mx *Mux, routes []chi.Route) []routeInfo {
	var result []routeInfo
	for _, v := range routes {
		for method := range v.Handlers {
			result = append(result, routeInfo{
				Method:  method,
				Pattern: v.Pattern,
				Handler: func() string {
					h := mx.handlerMetaFor(method, v.Pattern)
					if h == nil {
						return "«Unknown»"
					}
					t := h.handlerFunction.Type()
					return t.String()
				}(),
			})
		}
		if v.SubRoutes != nil {
			result = append(result, listRoutes(mx, v.SubRoutes.Routes())...)
		}
	}

	return result
}

func notFoundToTemplate(status int, r *Request) notFoundTemplate {
	environment := map[string]string{}
	for _, e := range os.Environ() {
		comps := strings.SplitN(e, "=", 2)
		environment[comps[0]] = comps[1]
	}

	tmpl := notFoundTemplate{
		Code:       status,
		StatusName: http.StatusText(status),
		Method:     r.HTTPRequest.Method,
		Path:       r.HTTPRequest.URL.Path,
		Headers:    oneToManyMap(r.HTTPRequest.Header),
		RequestDetails: requestInfo{
			Queries: oneToManyMap(r.HTTPRequest.URL.Query()),
			Form:    oneToManyMap(r.HTTPRequest.PostForm),
		},
		Environment: environment,
		Routes:      listRoutes(r.mux, r.mux.internalMux.Routes()),
	}

	return tmpl
}

func renderNotFoundHTMLErrorTemplate(r *Request) (string, error) {
	tmpl := notFoundToTemplate(http.StatusNotFound, r)
	return templates.TemplateNamed(templates.NotFoundErrorHTML)(tmpl)
}

func renderNotFoundTextErrorTemplate(r *Request) (string, error) {
	tmpl := notFoundToTemplate(http.StatusNotFound, r)
	return templates.TemplateNamed(templates.NotFoundErrorText)(tmpl)
}

func renderMethodNotAllowedHTMLErrorTemplate(r *Request) (string, error) {
	tmpl := notFoundToTemplate(http.StatusMethodNotAllowed, r)
	return templates.TemplateNamed(templates.NotFoundErrorHTML)(tmpl)
}

func renderMethodNotAllowedTextErrorTemplate(r *Request) (string, error) {
	tmpl := notFoundToTemplate(http.StatusMethodNotAllowed, r)
	return templates.TemplateNamed(templates.NotFoundErrorText)(tmpl)
}

var notFoundConstrainedHTMLErrorTemplate, _ = templates.TemplateNamed(templates.NotFoundErrorConstrainedHTML)(notFoundConstrainedValue)
var notFoundConstrainedTextErrorTemplate, _ = templates.TemplateNamed(templates.NotFoundErrorConstrainedText)(notFoundConstrainedValue)
var methodNotAllowedConstrainedHTMLErrorTemplate, _ = templates.TemplateNamed(templates.NotFoundErrorConstrainedHTML)(methodNotAllowedConstrainedValue)
var methodNotAllowedConstrainedTextErrorTemplate, _ = templates.TemplateNamed(templates.NotFoundErrorConstrainedText)(methodNotAllowedConstrainedValue)
