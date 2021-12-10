package raggett

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseContentType(t *testing.T) {
	t.Run("basic header with multiple parameters", func(t *testing.T) {
		input := "text/html;q=0.5, application/json"
		ok, medias := parseAcceptHeader(input)
		assert.True(t, ok)
		assert.Len(t, medias, 2)

		a := medias[0]
		assert.Equal(t, "text", a.TypeString)
		assert.Equal(t, "html", a.SubTypeString)
		assert.Len(t, a.Parameters, 1)
		assert.Equal(t, float32(0.5), a.Weight)
		assert.Equal(t, MediaTypeSpecificityFullyDefined, a.Specificity)

		b := medias[1]
		assert.Equal(t, "application", b.TypeString)
		assert.Equal(t, "json", b.SubTypeString)
		assert.Len(t, b.Parameters, 0)
		assert.Equal(t, float32(1), b.Weight)
		assert.Equal(t, MediaTypeSpecificityFullyDefined, b.Specificity)
	})

	t.Run("basic header with multiple parameters", func(t *testing.T) {
		input := "text/html;q=0.5, application/json;q=1.0; version=1"
		ok, medias := parseAcceptHeader(input)
		assert.True(t, ok)
		assert.Len(t, medias, 2)
	})

	t.Run("quoted strings", func(t *testing.T) {
		input := `foo/bar;key="A,B,C";otherKey="foo\"bar"`
		ok, medias := parseAcceptHeader(input)
		assert.True(t, ok)
		assert.Len(t, medias, 1)
	})

	t.Run("quoted strings II", func(t *testing.T) {
		input := `foo/bar;key="A,B,C";otherKey="foo\"bar",test/test;really=yep`
		ok, medias := parseAcceptHeader(input)
		assert.True(t, ok)
		assert.Len(t, medias, 1)
	})

	t.Run("invalid input", func(t *testing.T) {
		input := []string{
			"foo",
			"foo/",
			"foo/bar;",
			"foo/bar;x",
			"foo/bar;x=",
			"foo/bar;x=\"",
			"foo/bar;x=\"baz",
			"foo/bar;x=",
			";foo/bar",
			",",
		}
		for _, i := range input {
			ok, medias := parseAcceptHeader(i)
			assert.False(t, ok, i+" -- should fail")
			assert.Nil(t, medias, i+" -- should produce no output")
		}
	})

	t.Run("RFC Example Accept: headers", func(t *testing.T) {
		ok, medias := parseAcceptHeader("text/*;q=0.3, text/html;q=0.7, text/html;level=1,\n text/html;level=2;q=0.4, */*;q=0.5")
		require.True(t, ok)
		expectedTypes := []MediaType{
			{TypeString: "text", SubTypeString: "html", Parameters: map[string][]string{"level": {"1"}}, Weight: float32(1), Specificity: MediaTypeSpecificityFullyDefined},
			{TypeString: "text", SubTypeString: "html", Parameters: map[string][]string{"q": {"0.7"}}, Weight: float32(0.7), Specificity: MediaTypeSpecificityFullyDefined},
			{TypeString: "*", SubTypeString: "*", Parameters: map[string][]string{"q": {"0.5"}}, Weight: float32(0.5), Specificity: MediaTypeSpecificityUndefined},
			{TypeString: "text", SubTypeString: "html", Parameters: map[string][]string{"level": {"2"}, "q": {"0.4"}}, Weight: float32(0.4), Specificity: MediaTypeSpecificityFullyDefined},
			{TypeString: "text", SubTypeString: "*", Parameters: map[string][]string{"q": {"0.3"}}, Weight: float32(0.3), Specificity: MediaTypeSpecificityPartiallyDefined},
		}

		for _, e := range expectedTypes {
			found := false
			for _, m := range medias {
				if e.equals(m) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected %s to be contained within %v", e, medias)
			}
		}
	})

	t.Run("With invalid chars", func(t *testing.T) {
		invalids := []string{
			"hey/nope∆",
			"∆hey/nope",
			"unexpected/eof;around=\"here",
			"unexpected/eof;around=\"here\\",
			"invalid/;",
		}

		for _, v := range invalids {
			ok, medias := parseAcceptHeader(v)
			assert.False(t, ok, "%s should not be valid", v)
			assert.Empty(t, medias, "%s should not generate outputs", v)
		}
	})

	t.Run("recomposing", func(t *testing.T) {
		media := MediaTypeFromString("text", "plain")
		media.Parameters = map[string][]string{
			"q": {"1"},
		}
		assert.Equal(t, "text/plain;q=1", media.String())
	})

}

func TestNegotiateContentType(t *testing.T) {
	t.Run("Wildcard Example", func(t *testing.T) {
		inputs := []string{
			"    */*    ",
			"*/*",
			"        ",
			"",
			"*",
		}

		for _, i := range inputs {
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Accept", i)
			parsed, valid, selected := NegotiateContentType(r, []string{"text/plain"})
			assert.True(t, parsed, "Parse should be successful")
			assert.True(t, valid, "Selection should be successful")
			assert.Equal(t, "text/plain", selected.Type())
		}
	})

	t.Run("RFC Example Accept: headers", func(t *testing.T) {
		var cases = []struct {
			offers        []string
			input         string
			selects       string
			name          string
			failsMatching bool
			failsParsing  bool
		}{
			{
				name:    "common AJAX scenario",
				offers:  []string{"application/json", "text/html"},
				input:   "application/json, text/javascript, */*",
				selects: "application/json",
			},
			{
				offers:  []string{"application/xbel+xml", "application/xml"},
				input:   "application/xbel+xml",
				selects: "application/xbel+xml",
				name:    "direct match",
			},
			{

				offers:  []string{"application/xbel+xml", "application/xml"},
				input:   "application/xbel+xml; q=1",
				selects: "application/xbel+xml",
				name:    "direct match with a q parameter",
			},
			{
				offers:  []string{"application/xbel+xml", "application/xml"},
				input:   "application/xml; q=1",
				selects: "application/xml",
				name:    "direct match of our second choice with a q parameter",
			},
			{
				offers:  []string{"application/xbel+xml", "application/xml"},
				input:   "application/*; q=1",
				selects: "application/xml",
				name:    "match using a subtype wildcard",
			},
			{
				offers:  []string{"application/xbel+xml", "application/xml"},
				input:   "*/*",
				selects: "application/xml",
				name:    "match using a type wildcard",
			},
			{
				offers:  []string{"application/xbel+xml", "text/xml"},
				input:   "text/*;q=0.5,*/*; q=0.1",
				selects: "text/xml",
				name:    "match using a type versus a lower weighted subtype",
			},
			{
				offers:        []string{"application/xbel+xml", "text/xml"},
				input:         "text/html,application/atom+xml; q=0.9",
				selects:       "text/xml",
				name:          "fail to match anything",
				failsMatching: true,
			},
			{
				offers:  []string{"application/json", "text/html"},
				input:   "application/json, text/html;q=0.9",
				selects: "application/json",
				name:    "verify fitness ordering",
			},
			{
				offers:  []string{"image/jpeg", "text/plain"},
				input:   "text/*;q=0.3, text/html;q=0.7, text/html;level=1, text/html;level=2;q=0.4, */*;q=0.5",
				selects: "image/jpeg",
				name:    "media type with highest associated quality factor should win, not necessarily most specific",
			},
			{
				offers:  []string{"text/html", "application/rdf+xml"},
				input:   "text/html, application/rdf+xml",
				selects: "application/rdf+xml",
				name:    "match should use highest order of supported when there is a tie",
			},
			{
				offers:  []string{"application/json", "text/html", "text/plain"},
				input:   "*/*",
				selects: "application/json",
				name:    "*/* match should pick an acceptable type with the highest quality",
			},
			{
				offers:  []string{"text/html", "application/json", "text/plain"},
				input:   "*/*",
				selects: "text/html",
				name:    "*/* match should pick an acceptable type with the highest quality, even if it's implicit",
			},
			{
				offers:        []string{"application/json", "text/html"},
				input:         "text",
				selects:       "application/json",
				name:          "match should use the default if an invalid Accept header is passed",
				failsMatching: true,
				failsParsing:  true,
			},
			{
				offers:  []string{"application/json", "text/html"},
				input:   "*/json",
				selects: "application/json",
				name:    "match should accept wildcards in the type portion",
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Accept", c.input)
				parsed, selected, result := NegotiateContentType(r, c.offers)
				if c.failsParsing {
					assert.False(t, parsed)
				} else {
					assert.True(t, parsed)
				}
				if c.failsMatching {
					assert.False(t, selected)
				} else {
					assert.True(t, selected)
				}
				assert.Equal(t, c.selects, result.Type())
			})
		}
	})
}

func TestErrors(t *testing.T) {
	t.Parallel()

	t.Run("refuses empty offers", func(t *testing.T) {
		assert.Panics(t, func() {
			NegotiateContentTypeWithMediaTypes(nil, nil)
		})
	})

	t.Run("refuses offers with wildcards", func(t *testing.T) {
		assert.Panics(t, func() {
			NegotiateContentType(nil, []string{"text/*"})
		})
	})

	t.Run("refuses offers with params", func(t *testing.T) {
		assert.Panics(t, func() {
			NegotiateContentType(nil, []string{"text/plain;q=1"})
		})
	})

	t.Run("refuses malformed offers", func(t *testing.T) {
		assert.Panics(t, func() {
			NegotiateContentType(nil, []string{"text/"})
		})
		assert.Panics(t, func() {
			NegotiateContentType(nil, []string{"text"})
		})
		assert.Panics(t, func() {
			NegotiateContentType(nil, []string{"text/plain/test"})
		})
	})
}
