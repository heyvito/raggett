package raggett

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"github.com/go-chi/chi"
)

const uintSize = 32 << (^uint(0) >> 32 & 1)
const intSize = 32 << (^int(0) >> 32 & 1)

func parseAndSetInt(bitSize int, val string, inst reflect.Value) error {
	i, err := strconv.ParseInt(val, 10, bitSize)
	if err != nil {
		return err
	}
	inst.SetInt(i)
	return nil
}

func parseAndSetUInt(bitSize int, val string, inst reflect.Value) error {
	i, err := strconv.ParseUint(val, 10, bitSize)
	if err != nil {
		return err
	}
	inst.SetUint(i)
	return nil
}

func doPrimitiveCoercion(val string, kind reflect.Kind, into reflect.Value) error {
	switch kind {
	// Int
	case reflect.Int:
		return parseAndSetInt(intSize, val, into)
	case reflect.Int8:
		return parseAndSetInt(8, val, into)
	case reflect.Int16:
		return parseAndSetInt(16, val, into)
	case reflect.Int32:
		return parseAndSetInt(32, val, into)
	case reflect.Int64:
		return parseAndSetInt(64, val, into)

	// Uint
	case reflect.Uint:
		return parseAndSetUInt(uintSize, val, into)
	case reflect.Uint8:
		return parseAndSetUInt(8, val, into)
	case reflect.Uint16:
		return parseAndSetUInt(16, val, into)
	case reflect.Uint32:
		return parseAndSetUInt(32, val, into)
	case reflect.Uint64:
		return parseAndSetUInt(64, val, into)

	// Float
	case reflect.Float32:
		v, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return err
		}
		into.SetFloat(v)
	case reflect.Float64:
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		into.SetFloat(v)

	// String
	case reflect.String:
		into.SetString(val)

	// Bool
	case reflect.Bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		into.SetBool(b)
	default:
		return fmt.Errorf("unimplemented coersion method for type %s", kind.String())
	}

	return nil
}

func doCoercion(val []string, field *requestField, inst reflect.Value) error {
	instField := inst.FieldByIndex(field.structField.Index)
	switch field.structField.Type.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
		if len(val) == 0 {
			return nil
		}
		err := doPrimitiveCoercion(val[0], field.structField.Type.Kind(), instField)
		if err != nil {
			return makeError(err)
		}
	// Slices
	case reflect.Slice:
		innerType := field.structField.Type.Elem()
		newSlice := reflect.MakeSlice(field.structField.Type, len(val), len(val))
		for i, v := range val {
			inst := reflect.New(innerType).Elem()
			if err := doPrimitiveCoercion(v, innerType.Kind(), inst); err != nil {
				return err
			}
			newSlice.Index(i).Set(inst)
		}
		instField.Set(newSlice)
	}

	return nil
}

func makeValidationErrorWithError(fieldKind fieldKind, errorKind ValidationErrorKind, field *requestField, err error) ValidationError {
	var original error
	if err != nil {
		original = makeError(err)
	}

	return ValidationError{
		StructName:      field.requestMetadata.structType.Name(),
		StructFieldName: field.structField.Name,
		FieldName:       field.requestFieldName,
		FieldKind:       fieldKind,
		ErrorKind:       errorKind,
		OriginalError:   original,
	}
}

func makeValidationError(fieldKind fieldKind, errorKind ValidationErrorKind, field *requestField) error {
	return makeValidationErrorWithError(fieldKind, errorKind, field, nil)
}

func applyParam(exists bool, value []string, fieldKind fieldKind, field *requestField, inst reflect.Value) error {
	if field.required != nil && *field.required && !exists {
		return makeValidationError(fieldKind, ValidationErrorKindRequired, field)
	}

	if field.blank != nil && !*field.blank {
		for _, v := range value {
			if strings.TrimSpace(v) == "" {
				return makeValidationError(fieldKind, ValidationErrorKindBlank, field)
			}
		}
	}

	if field.pattern != nil {
		for _, v := range value {
			if !field.pattern.MatchString(v) {
				return makeValidationError(fieldKind, ValidationErrorKindPattern, field)
			}
		}
	}

	if err := doCoercion(value, field, inst); err != nil {
		return makeValidationErrorWithError(fieldKind, ValidationErrorKindParsing, field, err)
	}
	return nil
}

func applyFileParam(exists bool, value []*multipart.FileHeader, fieldKind fieldKind, field *requestField, inst reflect.Value) error {
	if field.required != nil && *field.required && (!exists || len(value) == 0) {
		return makeValidationError(fieldKind, ValidationErrorKindRequired, field)
	}

	if field.blank != nil && !*field.blank {
		for _, v := range value {
			if v.Size == 0 {
				return makeValidationError(fieldKind, ValidationErrorKindBlank, field)
			}
		}
	}

	ff := field.fileFieldKind

	var v reflect.Value
	shouldSet := false
	if ff.IsSlice() {
		if ff&FileFieldKindRaggettLib == FileFieldKindRaggettLib {
			converted := (*[]*FileHeader)(unsafe.Pointer(&value))
			v = reflect.ValueOf(*converted)
		} else {
			v = reflect.ValueOf(value)
		}
		shouldSet = true
	} else {
		if len(value) > 0 {
			if ff&FileFieldKindRaggettLib == FileFieldKindRaggettLib {
				converted := (*[]*FileHeader)(unsafe.Pointer(&value))
				v = reflect.ValueOf((*converted)[0])
			} else {
				v = reflect.ValueOf(value[0])
			}
			shouldSet = true
		}
	}

	if shouldSet {
		inst.FieldByIndex(field.structField.Index).Set(v)
	}

	return nil
}

func loadAndApplyMeta(meta *handlerMetadata, r *Request) error {
	instPtr := reflect.New(meta.structType)
	inst := instPtr.Elem()
	inst.FieldByIndex(meta.requestField.Index).Set(reflect.ValueOf(r))
	httpReq := r.HTTPRequest

	for k, v := range meta.queryParams {
		val, exists := httpReq.URL.Query()[k]
		if err := applyParam(exists, val, fieldKindQuery, v, inst); err != nil {
			return err
		}
	}

	{
		var routeParams chi.RouteParams
		routeParamsLen := 0
		if rctx := chi.RouteContext(httpReq.Context()); rctx != nil {
			routeParams = rctx.URLParams
			routeParamsLen = len(routeParams.Keys) - 1
		}

		for param, v := range meta.urlParams {
			exists := false
			var val string
			for k := routeParamsLen; k >= 0; k-- {
				if routeParams.Keys[k] == param {
					val = routeParams.Values[k]
					exists = true
				}
			}
			if err := applyParam(exists, []string{val}, fieldKindURLParam, v, inst); err != nil {
				return err
			}
		}
	}

	{
		heads := r.HTTPRequest.Header
		for param, v := range meta.headers {
			val, exists := heads[http.CanonicalHeaderKey(param)]
			if err := applyParam(exists, val, fieldKindHeader, v, inst); err != nil {
				return err
			}
		}
	}

	if meta.customParser {
		// instance has a custom parser defined. Just invoke it.
		customParser := inst.Addr().Interface().(CustomRequestParser)
		if err := customParser.ParseRequest(r); err != nil {
			return err
		}
	} else if meta.body != nil {
		err := handleBodyParsing(meta, r, inst)
		if err != nil {
			return makeValidationErrorWithError(fieldKindBody, ValidationErrorKindParsing, meta.body, err)
		}
	} else if len(meta.forms) > 0 {
		err := r.HTTPRequest.ParseMultipartForm(r.maxMemory)
		// ParseMultipartForm will fail with ErrNotMultipart when we don't have
		// a multipart form; this is mostly fine, since it will parse the form
		// itself (x-www-form-urlencoded?) before returning that error. So we
		// will filter it out;
		isMultipart := true
		values := r.HTTPRequest.Form
		if err != nil && err != http.ErrNotMultipart {
			go r.mux.errorHandler(err, r.httpResponse, r)
			return nil
		} else if err == http.ErrNotMultipart {
			isMultipart = false
		} else if err == nil {
			values = r.HTTPRequest.MultipartForm.Value
		}

		for k, v := range meta.forms {
			if v.fileFieldKind.IsFile() && isMultipart {
				val, exists := r.HTTPRequest.MultipartForm.File[k]
				if err := applyFileParam(exists, val, fieldKindForm, v, inst); err != nil {
					return err
				}
			} else {
				val, exists := values[k]
				if err := applyParam(exists, val, fieldKindForm, v, inst); err != nil {
					return err
				}
			}
		}
	}

	instVal := make([]reflect.Value, 1)

	if meta.wantsPtr {
		instVal[0] = reflect.Indirect(inst).Addr()
	} else {
		instVal[0] = inst
	}

	var err error
	var res reflect.Value
	func(callArgs []reflect.Value) {
		defer func() {
			innerErr := recover()
			if innerErr != nil {
				if innerErr == errAbortRequest {
					err = errAbortRequest
					return
				}
				if e, ok := innerErr.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("panic called on non-error type: %s", innerErr)
				}
			}
		}()
		res = meta.handlerFunction.Call(callArgs)[0]
	}(instVal)

	if err != nil {
		// We panicked earlier. Let's ignore the returned result and return our
		// captured exception.
		return err
	}

	if res.IsZero() || res.IsNil() {
		return nil
	}

	return res.Interface().(error)
}
