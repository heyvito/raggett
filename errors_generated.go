// Code generated by generators/generate_errors.go. DO NOT EDIT.
// This file was generated by a tool. Modifications will be overwritten.
package raggett

import (
	"fmt"
	"reflect"
)

// ErrCustomRequestParserConflict indicates that a given structure has a
// ParseRequest method implemented, which makes it implement
// CustomRequestParser. Implementing this interface makes the implementing
// structure ineligible to use fields with tags `form:"*"` or `body:"*"`.
type ErrCustomRequestParserConflict struct {
	structName string
}

func (e ErrCustomRequestParserConflict) Error() string {
	return fmt.Sprintf("invalid structure definition for %s: Can't implement CustomRequestParser and have forms or body fields", e.structName)
}
func errCustomRequestParserConflict(structName reflect.Type) error {
	return ErrCustomRequestParserConflict{
		structName: structName.Name(),
	}
}

// ErrBodyFormsConflict indicates that a given structure has fields using mixed
// `body:"*"` and `form:"*"` tags. As forms are parsed from the request's body,
// mixing both may lead to unexpected results, and therefore is not allowed.
type ErrBodyFormsConflict struct {
	structName string
}

func (e ErrBodyFormsConflict) Error() string {
	return fmt.Sprintf("invalid structure definition for %s: Can't have fields tagged as body and form", e.structName)
}
func errBodyFormsConflict(structName reflect.Type) error {
	return ErrBodyFormsConflict{
		structName: structName.Name(),
	}
}

// ErrFieldsWithoutResolver indicates that a given structure has inconsistent
// fields using one or more validator tags without using a resolver tag. In
// order to use validations, a resolver must be used. Resolvers are tags such as
// `url-param`, `query`, `body`, `form`, or `header`.
type ErrFieldsWithoutResolver struct {
	structName string
	fieldName  string
}

func (e ErrFieldsWithoutResolver) Error() string {
	return fmt.Sprintf("invalid structure definition for %s: Field %s uses at least one tag, but no resolver", e.structName, e.fieldName)
}
func errFieldsWithoutResolver(structName reflect.Type, fieldName reflect.StructField) error {
	return ErrFieldsWithoutResolver{
		structName: structName.Name(),
		fieldName:  fieldName.Name,
	}
}

// ErrMultipleResolver indicates that a given structure has a field using more
// than one resolver. Fields must not have more than one of the following tags:
// `url-param`, `query`, `body`, `form`, `header`.
type ErrMultipleResolver struct {
	structName string
	fieldName  string
}

func (e ErrMultipleResolver) Error() string {
	return fmt.Sprintf("invalid structure definition for %s: Field %s uses more than one value resolver; url-param, query, and body must be used only once per struct field", e.structName, e.fieldName)
}
func errMultipleResolver(structName reflect.Type, fieldName reflect.StructField) error {
	return ErrMultipleResolver{
		structName: structName.Name(),
		fieldName:  fieldName.Name,
	}
}

// ErrEmptyPattern indicates that a given structure contains a field using a
// `pattern` tag with an empty value.
type ErrEmptyPattern struct {
	structName string
	fieldName  string
}

func (e ErrEmptyPattern) Error() string {
	return fmt.Sprintf("invalid structure definition for %s: Field %s has empty regexp pattern", e.structName, e.fieldName)
}
func errEmptyPattern(structName reflect.Type, fieldName reflect.StructField) error {
	return ErrEmptyPattern{
		structName: structName.Name(),
		fieldName:  fieldName.Name,
	}
}

// ErrInvalidPattern indicates that a given structure contains a field using a
// `pattern` tag whose contents could not be compiled into a RegExp instance.
type ErrInvalidPattern struct {
	structName string
	fieldName  string
	err        string
}

func (e ErrInvalidPattern) Error() string {
	return fmt.Sprintf("invalid structure definition for %s: Field %s has invalid regexp pattern: %s", e.structName, e.fieldName, e.err)
}
func errInvalidPattern(structName reflect.Type, fieldName reflect.StructField, err error) error {
	return ErrInvalidPattern{
		structName: structName.Name(),
		fieldName:  fieldName.Name,
		err:        err.Error(),
	}
}

// ErrMultipleBodyTags indicates that a given structure have more than one field
// using a `body` tag.
type ErrMultipleBodyTags struct {
	structName string
}

func (e ErrMultipleBodyTags) Error() string {
	return fmt.Sprintf("invalid structure definition for %s: Multiple body tags in use", e.structName)
}
func errMultipleBodyTags(structName reflect.Type) error {
	return ErrMultipleBodyTags{
		structName: structName.Name(),
	}
}

// ErrEmptyBodyTag indicates that a given structure has a body field without a
// value. Either provide a valid body format (such as `json`, `xml`, `text`,
// `stream`, or `bytes`), or consider implementing CustomRequestParser.
type ErrEmptyBodyTag struct {
	structName string
	fieldName  string
}

func (e ErrEmptyBodyTag) Error() string {
	return fmt.Sprintf("invalid structure definition for %s: Field %s contains an empty body tag.", e.structName, e.fieldName)
}
func errEmptyBodyTag(structName reflect.Type, fieldName reflect.StructField) error {
	return ErrEmptyBodyTag{
		structName: structName.Name(),
		fieldName:  fieldName.Name,
	}
}

// ErrInvalidBodyParser indicates that a given structure has a field using a
// `body` loader with an invalid parser. Valid values for that tag are `json`,
// `xml`, `text`, `stream`, or `bytes`.
type ErrInvalidBodyParser struct {
	structName string
	fieldName  string
}

func (e ErrInvalidBodyParser) Error() string {
	return fmt.Sprintf("invalid structure definition for %s: Field %s contains an invalid body parser.", e.structName, e.fieldName)
}
func errInvalidBodyParser(structName reflect.Type, fieldName reflect.StructField) error {
	return ErrInvalidBodyParser{
		structName: structName.Name(),
		fieldName:  fieldName.Name,
	}
}

// ErrInvalidFileField indicates that a given structure has a field attempting
// to receive a File from the request, but has an invalid type. Raggett expects
// the file field be either a pointer, or a pointer slice of multipart.FileHeader
// or raggett.FileHeader, the latter being an alias to the former.
type ErrInvalidFileField struct {
	structName string
	fieldName  string
}

func (e ErrInvalidFileField) Error() string {
	return fmt.Sprintf("invalid structure definition for %s: Field %s must use a pointer to multipart.FileHeader or raggett.FileHeader", e.structName, e.fieldName)
}
func errInvalidFileField(structName reflect.Type, fieldName reflect.StructField) error {
	return ErrInvalidFileField{
		structName: structName.Name(),
		fieldName:  fieldName.Name,
	}
}
