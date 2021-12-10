package templates

import (
	"bytes"
	_ "embed"
	html "html/template"
	text "text/template"
)

const (
	ServerErrorHTML                = "error.html"
	ServerErrorText                = "error.txt"
	ServerErrorConstrainedHTML     = "error_constrained.html"
	ServerErrorConstrainedText     = "error_constrained.txt"
	ValidationErrorHTML            = "validation_error.html"
	ValidationErrorText            = "validation_error.txt"
	ValidationErrorConstrainedHTML = "validation_error_constrained.html"
	ValidationErrorConstrainedText = "validation_error_constrained.txt"
	NotFoundErrorHTML              = "not_found.html"
	NotFoundErrorText              = "not_found.txt"
	NotFoundErrorConstrainedHTML   = "not_found_constrained.html"
	NotFoundErrorConstrainedText   = "not_found_constrained.txt"
)

//go:embed error.html
var errorTemplateHTMLString string

//go:embed error.txt
var errorTemplateTextString string

//go:embed error_constrained.txt
var errorConstrainedTextString string

//go:embed error_constrained.html
var errorConstrainedHTMLString string

//go:embed validation_error.html
var validationErrorTemplateHTMLString string

//go:embed validation_error.txt
var validationErrorTemplateTextString string

//go:embed validation_error_constrained.txt
var validationErrorConstrainedTextString string

//go:embed validation_error_constrained.html
var validationErrorConstrainedHTMLString string

//go:embed not_found.html
var notFoundErrorHTMLString string

//go:embed not_found.txt
var notFoundErrorTextString string

//go:embed not_found_constrained.html
var notFoundErrorConstrainedHTMLString string

//go:embed not_found_constrained.txt
var notFoundErrorConstrainedTextString string

type Executor func(data interface{}) (string, error)

func mustLoadHTMLTemplate(data string) Executor {
	t, err := html.New("").Parse(data)
	if err != nil {
		panic("raggett: Failed loading template:" + err.Error())
	}
	return func(data interface{}) (string, error) {
		var w bytes.Buffer
		if err := t.Execute(&w, data); err != nil {
			return "", err
		}
		return w.String(), nil
	}
}

func mustLoadTextTemplate(data string) Executor {
	t, err := text.New("").Parse(data)
	if err != nil {
		panic("raggett: Failed loading template:" + err.Error())
	}
	return func(data interface{}) (string, error) {
		var w bytes.Buffer
		if err := t.Execute(&w, data); err != nil {
			return "", err
		}
		return w.String(), nil
	}
}

var templates = map[string]Executor{
	ServerErrorHTML:                mustLoadHTMLTemplate(errorTemplateHTMLString),
	ServerErrorText:                mustLoadTextTemplate(errorTemplateTextString),
	ServerErrorConstrainedHTML:     mustLoadHTMLTemplate(errorConstrainedHTMLString),
	ServerErrorConstrainedText:     mustLoadTextTemplate(errorConstrainedTextString),
	ValidationErrorHTML:            mustLoadHTMLTemplate(validationErrorTemplateHTMLString),
	ValidationErrorText:            mustLoadTextTemplate(validationErrorTemplateTextString),
	ValidationErrorConstrainedHTML: mustLoadHTMLTemplate(validationErrorConstrainedHTMLString),
	ValidationErrorConstrainedText: mustLoadTextTemplate(validationErrorConstrainedTextString),
	NotFoundErrorHTML:              mustLoadHTMLTemplate(notFoundErrorHTMLString),
	NotFoundErrorText:              mustLoadTextTemplate(notFoundErrorTextString),
	NotFoundErrorConstrainedHTML:   mustLoadHTMLTemplate(notFoundErrorConstrainedHTMLString),
	NotFoundErrorConstrainedText:   mustLoadHTMLTemplate(notFoundErrorConstrainedTextString),
}

func TemplateNamed(name string) Executor {
	v, ok := templates[name]
	if !ok {
		panic("Invalid template name " + name)
	}
	return v
}
