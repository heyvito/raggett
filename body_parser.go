package raggett

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

type bodyParser struct {
	typeName      string
	handler       func(r *http.Request, into reflect.Type) (reflect.Value, error)
	typeValidator func(t reflect.Type) error
}

func unmarshalerTypeValidator(name string) func(t reflect.Type) error {
	errorMsg := fmt.Sprintf("cannot use %s body on type %s", name, "%s")
	return func(t reflect.Type) error {
		k := t.Kind()
		if k != reflect.Struct && k != reflect.Slice && k != reflect.Ptr && k != reflect.Map {
			return fmt.Errorf(errorMsg, k)
		}

		return nil
	}
}

func specificTypeValidator(name string, kind reflect.Type) func(t reflect.Type) error {
	errorMsg := fmt.Sprintf("cannot use %s body on type %s. Expected %s.", name, "%s", kind)
	return func(t reflect.Type) error {
		k := t.Kind()
		if k != kind.Kind() {
			return fmt.Errorf(errorMsg, k)
		}

		return nil
	}
}

var jsonBodyParser = bodyParser{
	typeName:      "json",
	typeValidator: unmarshalerTypeValidator("json"),
	handler: func(r *http.Request, into reflect.Type) (reflect.Value, error) {
		inst := reflect.New(into)
		concreteInst := inst.Interface()

		if err := json.NewDecoder(r.Body).Decode(concreteInst); err != nil {
			return reflect.Value{}, err
		}

		return inst, nil
	},
}

var xmlBodyParser = bodyParser{
	typeName:      "xml",
	typeValidator: unmarshalerTypeValidator("xml"),
	handler: func(r *http.Request, into reflect.Type) (reflect.Value, error) {
		inst := reflect.New(into)
		concreteInst := inst.Interface()

		if err := xml.NewDecoder(r.Body).Decode(concreteInst); err != nil {
			return reflect.Value{}, err
		}

		return inst, nil
	},
}

var textBodyParser = bodyParser{
	typeName:      "text",
	typeValidator: specificTypeValidator("text", reflect.TypeOf("")),
	handler: func(r *http.Request, _ reflect.Type) (reflect.Value, error) {
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(string(bytes)), nil
	},
}

var bytesBodyParser = bodyParser{
	typeName: "bytes",
	typeValidator: func(t reflect.Type) error {
		err := fmt.Errorf("cannot use bytes body on type %s. Expected a slice of bytes ([]byte)", t)
		if t.Kind() != reflect.Slice {
			return err
		}
		if t.Elem().Kind() != reflect.Uint8 {
			return err
		}
		return nil
	},
	handler: func(r *http.Request, _ reflect.Type) (reflect.Value, error) {
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(bytes), nil
	},
}

var streamBodyParser = bodyParser{
	typeName: "stream",
	typeValidator: func(t reflect.Type) error {
		if t != reflect.TypeOf((*io.ReadCloser)(nil)).Elem() {
			return fmt.Errorf("cannot use stream body on type %s. Expected io.ReadCloser", t)
		}
		return nil
	},
	handler: func(r *http.Request, _ reflect.Type) (reflect.Value, error) {
		return reflect.ValueOf(r.Body), nil
	},
}

var bodyParsers = map[string]bodyParser{
	"json":   jsonBodyParser,
	"xml":    xmlBodyParser,
	"text":   textBodyParser,
	"stream": streamBodyParser,
	"bytes":  bytesBodyParser,
}

func handleBodyParsing(meta *handlerMetadata, r *Request, instance reflect.Value) error {
	strField := meta.body.structField
	// get handler
	handler := bodyParsers[meta.bodyKind]
	v, err := handler.handler(r.HTTPRequest, meta.body.structField.Type)
	if err != nil {
		return err
	}

	if k := strField.Type.Kind(); k == reflect.Struct {
		v = v.Elem()
	}

	instance.FieldByIndex(strField.Index).Set(v)
	return nil
}
