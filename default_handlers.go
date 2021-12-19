package raggett

import (
	"go.uber.org/zap"
	"net/http"
)

func (mx *Mux) defaultValidationErrorHandler(err ValidationError, w http.ResponseWriter, r *Request) {
	r.Logger.Error("Validation error serving request", zap.Error(err))

	if r.flushedHeaders {
		// Do not attempt to change the request in case we have already flushed
		// headers.
		return
	}

	vErr := validationErrorResponse{
		err:         err,
		r:           r,
		status:      http.StatusBadRequest,
		constrained: !mx.Development,
	}
	r.SetStatus(vErr.status)

	if r.bodyAllowedForStatus() {
		writeResponder(r, vErr)
	}
}

func (mx *Mux) defaultRuntimeErrorHandler(err error, w http.ResponseWriter, r *Request) {
	r.Logger.Error("Runtime error serving request", zap.Error(err))

	if r.flushedHeaders {
		// Do not attempt to change the request in case we have already flushed
		// headers.
		return
	}

	r.SetStatus(http.StatusInternalServerError)

	writeResponder(r, errorResponse{
		err:         err,
		r:           r,
		status:      http.StatusInternalServerError,
		constrained: !mx.Development,
	})
}
func (mx *Mux) defaultNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	// Here we don't have a Request object, since it didn't hit a responder.
	// Let's create a minimal one and try to move along. The same happens to
	// defaultMethodNotAllowedHandler.
	req := newRequest(mx, w, r)
	req.SetStatus(http.StatusNotFound)
	writeResponder(req, notFoundErrorResponse{
		constrained: !mx.Development,
		r:           req,
	})
}

func (mx *Mux) defaultMethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	req := newRequest(mx, w, r)
	req.SetStatus(http.StatusMethodNotAllowed)
	writeResponder(req, methodNotAllowedErrorResponse{
		constrained: !mx.Development,
		r:           req,
	})
}

// Mark: - validationErrorResponse

type validationErrorResponse struct {
	err         ValidationError
	r           *Request
	status      int
	constrained bool
}

func (v validationErrorResponse) JSON() interface{} {
	if v.constrained {
		return validationErrorToConstrainedTemplate(v.r, v.err, v.status)
	}

	return validationErrorToTemplate(v.r, v.err, v.status)
}

func (v validationErrorResponse) XML() interface{} {
	return v.JSON()
}

func (v validationErrorResponse) HTML() string {
	var (
		r   string
		err error
	)
	if v.constrained {
		r, err = renderConstrainedHTMLValidationErrorTemplate(v.r, v.err, v.status)
	} else {
		r, err = renderHTMLValidationErrorTemplate(v.r, v.err, v.status)
	}

	if err != nil {
		panic("raggett: Failed rendering validation error template: " + err.Error())
	}
	return r
}

func (v validationErrorResponse) PlainText() string {
	var (
		r   string
		err error
	)
	if v.constrained {
		r, err = renderConstrainedTextValidationErrorTemplate(v.r, v.err, v.status)
	} else {
		r, err = renderTextValidationErrorTemplate(v.r, v.err, v.status)
	}
	if err != nil {
		panic("raggett: Failed rendering validation error template: " + err.Error())
	}
	return r
}

// Mark: - errorResponse

type errorResponse struct {
	err         error
	r           *Request
	status      int
	constrained bool
}

func (e errorResponse) JSON() interface{} {
	if e.constrained {
		return errorToConstrainedTemplate(e.r, e.status)
	}
	return errorToTemplate(e.r, e.err, e.status)
}

func (e errorResponse) XML() interface{} {
	return e.JSON()
}

func (e errorResponse) HTML() string {
	var (
		r   string
		err error
	)
	if e.constrained {
		r, err = renderConstrainedHTMLErrorTemplate(e.r, e.status)
	} else {
		r, err = renderHTMLErrorTemplate(e.r, e.err, e.status)
	}
	if err != nil {
		panic("raggett: Failed rendering validation error template: " + err.Error())
	}
	return r
}

func (e errorResponse) PlainText() string {
	var (
		r   string
		err error
	)
	if e.constrained {
		r, err = renderConstrainedTextErrorTemplate(e.r, e.status)
	} else {
		r, err = renderTextErrorTemplate(e.r, e.err, e.status)
	}
	if err != nil {
		panic("raggett: Failed rendering validation error template: " + err.Error())
	}
	return r
}

// Mark: - notFoundErrorResponse

type notFoundErrorResponse struct {
	constrained bool
	r           *Request
}

func (nf notFoundErrorResponse) JSON() interface{} {
	if nf.constrained {
		return notFoundConstrainedValue
	}
	return notFoundToTemplate(http.StatusNotFound, nf.r)
}

func (nf notFoundErrorResponse) XML() interface{} {
	return nf.JSON()
}

func (nf notFoundErrorResponse) HTML() string {
	var (
		r   string
		err error
	)
	if nf.constrained {
		r = notFoundConstrainedHTMLErrorTemplate
	} else {
		r, err = renderNotFoundHTMLErrorTemplate(nf.r)
	}
	if err != nil {
		panic("raggett: Failed rendering NotFound error template: " + err.Error())
	}
	return r
}

func (nf notFoundErrorResponse) PlainText() string {
	var (
		r   string
		err error
	)
	if nf.constrained {
		r = notFoundConstrainedTextErrorTemplate
	} else {
		r, err = renderNotFoundTextErrorTemplate(nf.r)
	}
	if err != nil {
		panic("raggett: Failed rendering NotFound error template: " + err.Error())
	}
	return r
}

// Mark: - methodNotAllowedErrorResponse

type methodNotAllowedErrorResponse struct {
	constrained bool
	r           *Request
}

func (mn methodNotAllowedErrorResponse) JSON() interface{} {
	if mn.constrained {
		return methodNotAllowedConstrainedValue
	}
	return notFoundToTemplate(http.StatusMethodNotAllowed, mn.r)
}

func (mn methodNotAllowedErrorResponse) XML() interface{} {
	return mn.JSON()
}

func (mn methodNotAllowedErrorResponse) HTML() string {
	var (
		r   string
		err error
	)
	if mn.constrained {
		r = methodNotAllowedConstrainedHTMLErrorTemplate
	} else {
		r, err = renderMethodNotAllowedHTMLErrorTemplate(mn.r)
	}
	if err != nil {
		panic("raggett: Failed rendering MethodNotAllowed error template: " + err.Error())
	}
	return r
}

func (mn methodNotAllowedErrorResponse) PlainText() string {
	var (
		r   string
		err error
	)
	if mn.constrained {
		r = methodNotAllowedConstrainedTextErrorTemplate
	} else {
		r, err = renderMethodNotAllowedTextErrorTemplate(mn.r)
	}
	if err != nil {
		panic("raggett: Failed rendering MethodNotAllowed error template: " + err.Error())
	}
	return r
}
