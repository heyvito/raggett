package raggett

import (
	"mime/multipart"
	"reflect"
	"regexp"
	"strings"
)

var requestReflectType = reflect.TypeOf(&Request{})
var customRequestParserType = reflect.TypeOf((*CustomRequestParser)(nil)).Elem()
var multipartFileHeaderField = reflect.TypeOf(multipart.FileHeader{})
var raggettFileHeaderField = reflect.TypeOf(FileHeader{})

type fileFieldKind byte

func (f fileFieldKind) IsFile() bool {
	return (f&FileFieldKindRaggettLib)|(f&FileFieldKindStdLib) != 0
}

func (f fileFieldKind) IsSlice() bool {
	return (f & FileFieldKindSlice) == FileFieldKindSlice
}

const (
	FileFieldKindNone fileFieldKind = 1 << iota
	FileFieldKindStdLib
	FileFieldKindRaggettLib
	FileFieldKindSlice
)

func kindForFileField(input reflect.Type, f reflect.StructField) (fileFieldKind, error) {
	var result fileFieldKind
	switch f.Type.Kind() {
	case reflect.Struct:
		if f.Type == multipartFileHeaderField || f.Type == raggettFileHeaderField {
			return FileFieldKindNone, errInvalidFileField(input, f)
		}
		return FileFieldKindNone, nil
	case reflect.Ptr:
		el := f.Type.Elem()
		if el == multipartFileHeaderField {
			result |= FileFieldKindStdLib
		} else if el == raggettFileHeaderField {
			result |= FileFieldKindRaggettLib
		} else {
			return FileFieldKindNone, nil
		}

	case reflect.Slice:
		el := f.Type.Elem()
		if el.Kind() == reflect.Ptr {
			el := el.Elem()
			if el == multipartFileHeaderField {
				result |= FileFieldKindStdLib
			} else if el == raggettFileHeaderField {
				result |= FileFieldKindRaggettLib
			} else {
				return FileFieldKindNone, nil
			}
		} else if el == multipartFileHeaderField || el == raggettFileHeaderField {
			return FileFieldKindNone, errInvalidFileField(input, f)
		} else {
			return FileFieldKindNone, nil
		}
		result |= FileFieldKindSlice
	}
	return result, nil
}

type requestField struct {
	requestMetadata  *handlerMetadata
	structField      *reflect.StructField
	requestFieldName string
	pattern          *regexp.Regexp
	required         *bool
	blank            *bool
	fileFieldKind    fileFieldKind
}

type handlerMetadata struct {
	structType      reflect.Type
	wantsPtr        bool
	requestField    *reflect.StructField
	customParser    bool
	handlerFunction *reflect.Value
	urlParams       map[string]*requestField
	queryParams     map[string]*requestField
	body            *requestField
	bodyKind        string
	headers         map[string]*requestField
	forms           map[string]*requestField
}

func (hm *handlerMetadata) hasURLParam(name string) bool {
	_, ok := hm.urlParams[name]
	return ok
}

func (hm *handlerMetadata) hasQueryParam(name string) bool {
	_, ok := hm.queryParams[name]
	return ok
}

func determineFuncParams(fn interface{}) (*handlerMetadata, error) {
	val := reflect.ValueOf(fn)
	fnType := val.Type()

	if fnType.Kind() != reflect.Func {
		return nil, ErrNoFunction
	}

	inArity := fnType.NumIn()
	if inArity != 1 {
		return nil, ErrIncorrectArgsArity
	}

	outArity := fnType.NumOut()
	if outArity != 1 {
		return nil, ErrIncorrectOutArity
	}

	// In must be something inheriting Request
	input := fnType.In(0)
	originalInput := input
	wantsPointer := input.Kind() == reflect.Ptr
	if wantsPointer {
		input = input.Elem()
	}

	if input.Kind() != reflect.Struct {
		return nil, ErrIncorrectArgType
	}

	reqField, ok := input.FieldByName("Request")
	if !ok {
		return nil, ErrIncorrectArgType
	}
	if reqField.Type != requestReflectType {
		return nil, ErrIncorrectArgType
	}

	reqMeta := &handlerMetadata{
		wantsPtr:        wantsPointer,
		handlerFunction: &val,
		structType:      input,
		requestField:    &reqField,
		customParser:    originalInput.Implements(customRequestParserType),
		urlParams:       map[string]*requestField{},
		queryParams:     map[string]*requestField{},
		headers:         map[string]*requestField{},
		forms:           map[string]*requestField{},
	}

	for i := 0; i < input.NumField(); i++ {
		field := input.Field(i)

		// Must have only one set. More than one is an error.
		urlParam, hasURLParam := field.Tag.Lookup("url-param")
		query, hasQuery := field.Tag.Lookup("query")
		body, hasBody := field.Tag.Lookup("body")
		form, hasForm := field.Tag.Lookup("form")
		header, hasHeader := field.Tag.Lookup("header")

		hasResolver := false
		hasMoreThanOneResolver := false
		for _, b := range []bool{hasURLParam, hasQuery, hasBody, hasForm, hasHeader} {
			if b {
				if hasResolver {
					hasMoreThanOneResolver = true
					break
				}
				hasResolver = true
			}
		}

		blank, hasBlank := field.Tag.Lookup("blank")
		pattern, hasPattern := field.Tag.Lookup("pattern")
		required, hasRequired := field.Tag.Lookup("required")

		hasFields := hasBlank || hasPattern || hasRequired

		if !hasResolver && !hasFields {
			continue
		}

		if !hasResolver && hasFields {
			return nil, errFieldsWithoutResolver(input, field)
		}

		if hasMoreThanOneResolver {
			return nil, errMultipleResolver(input, field)
		}

		fileFieldDetectedKind, err := kindForFileField(input, field)
		if err != nil {
			return nil, err
		}

		reqField := &requestField{
			requestMetadata: reqMeta,
			structField:     &field,
			fileFieldKind:   fileFieldDetectedKind,
		}

		if hasBlank {
			blank := strings.EqualFold(blank, "true")
			reqField.blank = &blank
		}

		if hasPattern {
			if pattern == "" {
				return nil, errEmptyPattern(input, field)
			}

			pat, err := regexp.Compile(pattern)
			if err != nil {
				return nil, errInvalidPattern(input, field, err)
			}
			reqField.pattern = pat
		}

		if hasRequired {
			required := strings.EqualFold(required, "true")
			reqField.required = &required
		}

		if hasBody {
			if reqMeta.body != nil {
				return nil, errMultipleResolver(input, field)
			}
			if body == "" {
				return nil, errEmptyBodyTag(input, field)
			}

			parser, valid := bodyParsers[body]

			if !valid {
				return nil, errInvalidBodyParser(input, field)
			}

			if err := parser.typeValidator(field.Type); err != nil {
				return nil, err
			}

			reqMeta.body = reqField
			reqMeta.bodyKind = body
		} else if hasQuery {
			reqField.requestFieldName = query
			reqMeta.queryParams[query] = reqField
		} else if hasURLParam {
			reqField.requestFieldName = urlParam
			reqMeta.urlParams[urlParam] = reqField
		} else if hasHeader {
			reqField.requestFieldName = header
			reqMeta.headers[header] = reqField
		} else if hasForm {
			reqField.requestFieldName = form
			reqMeta.forms[form] = reqField
		}
	}

	if reqMeta.customParser && (reqMeta.body != nil || len(reqMeta.forms) > 0) {
		return nil, errCustomRequestParserConflict(input)
	}

	if reqMeta.body != nil && len(reqMeta.forms) > 0 {
		return nil, errBodyFormsConflict(input)
	}

	return reqMeta, nil
}
