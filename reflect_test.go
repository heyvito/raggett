package raggett

import (
	"io"
	"mime/multipart"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type requestWithParserConflict struct {
	*Request
	Data map[string]interface{} `body:"xml"`
}

func (requestWithParserConflict) ParseRequest(r *Request) error {
	panic("implement me")
}

func Test_determineFuncParams(t *testing.T) {
	t.Parallel()

	t.Run("With invalid function", func(t *testing.T) {
		_, err := determineFuncParams("nope")
		assert.Equal(t, ErrNoFunction, err)
	})

	t.Run("With invalid argument arity", func(t *testing.T) {
		_, err := determineFuncParams(func() string { return "foo" })
		assert.Equal(t, ErrIncorrectArgsArity, err)
	})

	t.Run("With invalid output arity", func(t *testing.T) {
		_, err := determineFuncParams(func(foo string) {})
		assert.Equal(t, ErrIncorrectOutArity, err)
	})

	t.Run("With invalid argument type", func(t *testing.T) {
		_, err := determineFuncParams(func(foo string) error { return nil })
		assert.Equal(t, ErrIncorrectArgType, err)
	})

	t.Run("With invalid argument struct", func(t *testing.T) {
		_, err := determineFuncParams(func(foo struct{}) error { return nil })
		assert.Equal(t, ErrIncorrectArgType, err)
	})

	t.Run("With empty pattern", func(t *testing.T) {
		_, err := determineFuncParams(func(foo struct {
			*Request
			Patt string `form:"foo" pattern:""`
		}) error {
			return nil
		})
		assert.IsType(t, ErrEmptyPattern{}, err)
	})

	t.Run("With invalid pattern", func(t *testing.T) {
		_, err := determineFuncParams(func(foo struct {
			*Request
			Patt string `form:"foo" pattern:"(["`
		}) error {
			return nil
		})
		assert.IsType(t, ErrInvalidPattern{}, err)
	})

	t.Run("With multiple resolvers", func(t *testing.T) {
		_, err := determineFuncParams(func(foo struct {
			*Request
			A map[string]interface{} `body:"json"`
			B map[string]interface{} `body:"xml"`
		}) error {
			return nil
		})
		assert.IsType(t, ErrMultipleResolver{}, err)
	})

	t.Run("With fields without resolver", func(t *testing.T) {
		_, err := determineFuncParams(func(foo struct {
			*Request
			A map[string]interface{} `required:"true"`
			B map[string]interface{} `body:"xml"`
		}) error {
			return nil
		})

		assert.IsType(t, ErrFieldsWithoutResolver{}, err)
	})

	t.Run("with fields with multiple resolvers", func(t *testing.T) {
		_, err := determineFuncParams(func(foo struct {
			*Request
			A string `form:"json" header:"foo"`
		}) error {
			return nil
		})
		assert.IsType(t, ErrMultipleResolver{}, err)
	})

	t.Run("With empty body tag", func(t *testing.T) {
		_, err := determineFuncParams(func(foo struct {
			*Request
			A interface{} `body:""`
		}) error {
			return nil
		})
		assert.IsType(t, ErrEmptyBodyTag{}, err)
	})

	t.Run("With empty body tag", func(t *testing.T) {
		_, err := determineFuncParams(func(foo struct {
			*Request
			A interface{} `body:"foobar"`
		}) error {
			return nil
		})
		assert.IsType(t, ErrInvalidBodyParser{}, err)
	})

	t.Run("With request parser conflicts", func(t *testing.T) {
		_, err := determineFuncParams(func(foo requestWithParserConflict) error {
			return nil
		})
		assert.IsType(t, ErrCustomRequestParserConflict{}, err)
	})

	t.Run("With request body-form conflicts", func(t *testing.T) {
		_, err := determineFuncParams(func(foo struct {
			*Request
			A map[string]interface{} `body:"json"`
			B string                 `form:"foo"`
		}) error {
			return nil
		})
		assert.IsType(t, ErrBodyFormsConflict{}, err)
	})

	t.Run("With valid signature", func(t *testing.T) {
		_, err := determineFuncParams(func(foo struct{ *Request }) error { return nil })
		assert.NoError(t, err)
	})

	t.Run("Extracting metadata", func(t *testing.T) {
		type SomeRequestType struct {
			*Request
			User     int                    `url-param:"user"`
			Post     int                    `url-param:"post"`
			Lang     string                 `query:"lang" blank:"false"`
			DateTime time.Time              `query:"timestamp" pattern:"^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}"`
			Struct   map[string]interface{} `body:"json"`
		}

		meta, err := determineFuncParams(func(requestType SomeRequestType) error { return nil })
		require.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Truef(t, meta.hasURLParam("user"), "should have url-param user")
		assert.Truef(t, meta.hasURLParam("post"), "should have url-param post")
		assert.Truef(t, meta.hasQueryParam("lang"), "should have query lang")
		assert.Truef(t, meta.hasQueryParam("timestamp"), "should query timestamp")
	})
}

func TestUnmarshalerTypeValidation(t *testing.T) {
	type InvalidJSONRequest struct {
		*Request
		Field int `body:"json"`
	}

	type InvalidXMLRequest struct {
		*Request
		Field int `body:"xml"`
	}

	type ValidJSONRequest struct {
		*Request
		Field map[string]interface{} `body:"json"`
	}

	type ValidXMLRequest struct {
		*Request
		Field map[string]interface{} `body:"xml"`
	}

	t.Parallel()

	m := NewMux(zap.NewNop())

	t.Run("With Invalid JSON", func(t *testing.T) {
		assert.Panics(t, func() {
			m.Post("/1", func(req *InvalidJSONRequest) error {
				return nil
			})
		})
	})

	t.Run("With invalid XML", func(t *testing.T) {
		assert.Panics(t, func() {
			m.Post("/2", func(req *InvalidXMLRequest) error {
				return nil
			})
		})
	})

	t.Run("With valid JSON", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.Post("/3", func(req *ValidJSONRequest) error {
				return nil
			})
		})
	})

	t.Run("With valid XML", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m.Post("/3", func(req *ValidXMLRequest) error {
				return nil
			})
		})
	})
}

func performUnmarshalerTest(t *testing.T, valid, invalid interface{}, expectedType string) {
	meta, err := determineFuncParams(invalid)
	assert.Error(t, err)
	assert.Nil(t, meta)
	meta, err = determineFuncParams(valid)
	assert.NoError(t, err)
	require.NotNil(t, meta)
	assert.Equal(t, expectedType, meta.bodyKind)
}

func TestOtherUnmarshalerValidation(t *testing.T) {
	t.Parallel()

	t.Run("text", func(t *testing.T) {
		type Invalid struct {
			*Request
			Field int `body:"text"`
		}
		type Valid struct {
			*Request
			Field string `body:"text"`
		}
		invalid := func(req *Invalid) error { return nil }
		valid := func(req *Valid) error { return nil }
		performUnmarshalerTest(t, valid, invalid, "text")
	})

	t.Run("stream", func(t *testing.T) {
		type Invalid struct {
			*Request
			Field int `body:"stream"`
		}
		type Valid struct {
			*Request
			Field io.ReadCloser `body:"stream"`
		}
		invalid := func(req *Invalid) error { return nil }
		valid := func(req *Valid) error { return nil }
		performUnmarshalerTest(t, valid, invalid, "stream")
	})

	t.Run("bytes", func(t *testing.T) {
		type Invalid struct {
			*Request
			Field string `body:"bytes"`
		}
		type Invalid2 struct {
			*Request
			Field []string `body:"bytes"`
		}
		type Valid struct {
			*Request
			Field []byte `body:"bytes"`
		}
		invalid := func(req *Invalid) error { return nil }
		invalid2 := func(req *Invalid2) error { return nil }
		valid := func(req *Valid) error { return nil }
		performUnmarshalerTest(t, valid, invalid, "bytes")
		performUnmarshalerTest(t, valid, invalid2, "bytes")
	})
}

func TestCustomRequestParser(t *testing.T) {
	meta, err := determineFuncParams(func(req *CustomParserRequestStruct) error { return nil })
	require.NoError(t, err)
	require.True(t, meta.customParser)
}

func TestFileFieldRaggettLibSingle(t *testing.T) {
	type FileStruct struct {
		*Request
		File *FileHeader `form:"foo"`
	}
	meta, err := determineFuncParams(func(req *FileStruct) error { return nil })
	require.NoError(t, err)
	require.Equal(t, FileFieldKindRaggettLib, meta.forms["foo"].fileFieldKind)
}

func TestFileFieldRaggettLibMulti(t *testing.T) {
	type FileStruct struct {
		*Request
		File []*FileHeader `form:"foo"`
	}
	meta, err := determineFuncParams(func(req *FileStruct) error { return nil })
	require.NoError(t, err)
	require.Equal(t, FileFieldKindRaggettLib|FileFieldKindSlice, meta.forms["foo"].fileFieldKind)
}

func TestFileFieldStdLibSingle(t *testing.T) {
	type FileStruct struct {
		*Request
		File *multipart.FileHeader `form:"foo"`
	}
	meta, err := determineFuncParams(func(req *FileStruct) error { return nil })
	require.NoError(t, err)
	require.Equal(t, FileFieldKindStdLib, meta.forms["foo"].fileFieldKind)
}

func TestFileFieldStdLibMulti(t *testing.T) {
	type FileStruct struct {
		*Request
		File []*multipart.FileHeader `form:"foo"`
	}
	meta, err := determineFuncParams(func(req *FileStruct) error { return nil })
	require.NoError(t, err)
	require.Equal(t, FileFieldKindStdLib|FileFieldKindSlice, meta.forms["foo"].fileFieldKind)
}

func TestFileFieldErrors(t *testing.T) {
	type InvalidNonPointer struct {
		*Request
		File FileHeader `form:"foo"`
	}

	type InvalidSliceNonPointer struct {
		*Request
		File []FileHeader `form:"foo"`
	}

	meta, err := determineFuncParams(func(req *InvalidNonPointer) error { return nil })
	assert.Error(t, err)
	assert.IsType(t, err, ErrInvalidFileField{})
	assert.Nil(t, meta)

	meta, err = determineFuncParams(func(req *InvalidSliceNonPointer) error { return nil })
	assert.Error(t, err)
	assert.IsType(t, err, ErrInvalidFileField{})
	assert.Nil(t, meta)
}
