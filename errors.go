package raggett

import (
	"fmt"
	"strings"
)

///////////
// Public

type ValidationErrorKind int

func (v ValidationErrorKind) String() string {
	switch v {
	case ValidationErrorKindBlank:
		return "cannot be blank"
	case ValidationErrorKindPattern:
		return "did not match the expected pattern"
	case ValidationErrorKindRequired:
		return "is required"
	case ValidationErrorKindParsing:
		return "is invalid"
	default:
		return fmt.Sprintf("«Error: Unexpected ValidationErrorKind %d»", v)
	}
}

func (v ValidationErrorKind) Name() string {
	switch v {
	case ValidationErrorKindBlank:
		return "ValidationErrorKindBlank"
	case ValidationErrorKindPattern:
		return "ValidationErrorKindPattern"
	case ValidationErrorKindRequired:
		return "ValidationErrorKindRequired"
	case ValidationErrorKindParsing:
		return "ValidationErrorKindParsing"
	default:
		return fmt.Sprintf("«Error: Unexpected ValidationErrorKind %d»", v)
	}
}

const (
	ValidationErrorKindBlank ValidationErrorKind = iota
	ValidationErrorKindPattern
	ValidationErrorKindRequired
	ValidationErrorKindParsing
)

type ValidationError struct {
	StructName      string
	StructFieldName string
	FieldName       string
	FieldKind       fieldKind
	ErrorKind       ValidationErrorKind
	OriginalError   error
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("Validation of %s failed: Value for field %s %s", v.StructName, v.FieldName, v.ErrorKind)
}

type Error struct {
	StackTrace    []StackFrame
	OriginalError error
}

func (e Error) Error() string {
	return e.OriginalError.Error()
}

func (e Error) Stack() string {
	trace := make([]string, 0, len(e.StackTrace))
	for _, t := range e.StackTrace {
		trace = append(trace, fmt.Sprintf("%s:%d %s 0x%x", t.File, t.Line, t.Func, t.ProgramCounter))
	}
	return strings.Join(trace, "\n")
}

func (e Error) String() string {
	return fmt.Sprintf("Error: %s\nStack Trace:\n%s\n", e.OriginalError.Error(), e.Stack())
}

///////////
// Internal

var errAbortRequest = fmt.Errorf("__raggett_abort_request")
var errAbortNotFound = fmt.Errorf("__raggett_abort_request_not_found")

///////////
// Reflect

var ErrNoFunction = fmt.Errorf("handler must be a function")
var ErrIncorrectArgsArity = fmt.Errorf("invalid arguments arity. Expected 1")
var ErrIncorrectOutArity = fmt.Errorf("invalid returns arity. Expected 1")
var ErrIncorrectArgType = fmt.Errorf("invalid argument type. Must be a struct using a *raggett.Request promoted structField")

//+errGen:ErrCustomRequestParserConflict(structName reflect.Type->Name())
//		  msg: invalid structure definition for \(structName): Can't implement CustomRequestParser and have forms or body fields
// ErrCustomRequestParserConflict indicates that a given structure has a
// ParseRequest method implemented, which makes it implement
// CustomRequestParser. Implementing this interface makes the implementing
// structure ineligible to use fields with tags `form:"*"` or `body:"*"`.

//+errGen:ErrBodyFormsConflict(structName reflect.Type->Name())
//        msg: invalid structure definition for \(structName): Can't have fields tagged as body and form
// ErrBodyFormsConflict indicates that a given structure has fields using mixed
// `body:"*"` and `form:"*"` tags. As forms are parsed from the request's body,
// mixing both may lead to unexpected results, and therefore is not allowed.

//+errGen:ErrFieldsWithoutResolver(structName reflect.Type->Name(), fieldName reflect.StructField->Name)
// 	      msg: invalid structure definition for \(structName): Field \(fieldName) uses at least one tag, but no resolver
// ErrFieldsWithoutResolver indicates that a given structure has inconsistent
// fields using one or more validator tags without using a resolver tag. In
// order to use validations, a resolver must be used. Resolvers are tags such as
// `url-param`, `query`, `body`, `form`, or `header`.

//+errGen:ErrMultipleResolver(structName reflect.Type->Name(), fieldName reflect.StructField->Name)
//        msg: invalid structure definition for \(structName): Field \(fieldName) uses more than one value resolver; url-param, query, and body must be used only once per struct field
// ErrMultipleResolver indicates that a given structure has a field using more
// than one resolver. Fields must not have more than one of the following tags:
// `url-param`, `query`, `body`, `form`, `header`.

//+errGen:ErrEmptyPattern(structName reflect.Type->Name(), fieldName reflect.StructField->Name)
//        msg: invalid structure definition for \(structName): Field \(fieldName) has empty regexp pattern
// ErrEmptyPattern indicates that a given structure contains a field using a
// `pattern` tag with an empty value.

//+errGen:ErrInvalidPattern(structName reflect.Type->Name(), fieldName reflect.StructField->Name, err error->Error())
//        msg: invalid structure definition for \(structName): Field %s has invalid regexp pattern: \(err)
// ErrInvalidPattern indicates that a given structure contains a field using a
// `pattern` tag whose contents could not be compiled into a RegExp instance.

//+errGen:ErrMultipleBodyTags(structName reflect.Type->Name())
//        msg: invalid structure definition for \(structName): Multiple body tags in use
// ErrMultipleBodyTags indicates that a given structure have more than one field
// using a `body` tag.

//+errGen:ErrEmptyBodyTag(structName reflect.Type->Name(), fieldName reflect.StructField->Name)
//        msg: invalid structure definition for \(structName): Field \(fieldName) contains an empty body tag.
// ErrEmptyBodyTag indicates that a given structure has a body field without a
// value. Either provide a valid body format (such as `json`, `xml`, `text`,
// `stream`, or `bytes`), or consider implementing CustomRequestParser.

//+errGen:ErrInvalidBodyParser(structName reflect.Type->Name(), fieldName reflect.StructField->Name)
//        msg: invalid structure definition for \(structName): Field \(fieldName) contains an invalid body parser.
// ErrInvalidBodyParser indicates that a given structure has a field using a
// `body` loader with an invalid parser. Valid values for that tag are `json`,
// `xml`, `text`, `stream`, or `bytes`.

//+errGen:ErrInvalidFileField(structName reflect.Type->Name(), fieldName reflect.StructField->Name)
//        msg: invalid structure definition for \(structName): Field \(fieldName) must use a pointer to multipart.FileHeader or raggett.FileHeader
// ErrInvalidFileField indicates that a given structure has a field attempting
// to receive a File from the request, but has an invalid type. Raggett expects
// the file field be either a pointer, or a pointer slice of multipart.FileHeader
// or raggett.FileHeader, the latter being an alias to the former.

//go:generate go run generators/errors/generate_errors.go
