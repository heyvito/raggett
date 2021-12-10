package raggett

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var offersCache = map[string]MediaType{}

// NegotiateContentType takes a http.Request and a list of strings containing
// media types in the "type/subType" format, attempts to negotiate one of the
// offered types with the values provided by the Client.
// Panics in case any content type is invalid, contains a wildcard, or extra
// parameters.
// This function converts the provided list of offers into a slice of MediaType
// and invokes NegotiateContentTypeWithMediaTypes.
func NegotiateContentType(r *http.Request, offers []string) (parsed, valid bool, selected MediaType) {
	for _, o := range offers {
		if strings.ContainsRune(o, '*') {
			panic("raggett: offers cannot contain wildcards")
		}
		if strings.ContainsRune(o, ';') {
			panic("raggett: offers cannot contain semicolons")
		}
	}

	var mediaOffers = make([]MediaType, len(offers))
	for i, o := range offers {
		o = strings.ToLower(o)
		if v, ok := offersCache[o]; ok {
			mediaOffers[i] = v
			continue
		}
		components := strings.SplitN(o, "/", 2)
		if len(components) != 2 {
			panic("raggett: invalid offer format " + o)
		}
		mediaOffers[i] = MediaTypeFromString(components[0], components[1])
		offersCache[o] = mediaOffers[i]
	}

	return NegotiateContentTypeWithMediaTypes(r, mediaOffers)
}

// NegotiateContentTypeWithMediaTypes takes a http.Request and a slice of
// MediaType, and attempts to negotiate one of the offered types with the values
// provided by the Client.
// Panics in case the list of offers is empty.
// Returns whether the Accept header provided by the client could be `parsed`,
// whether one of the offerings matched (`valid`), and either the matched media
// type, or the first non-vendored media type provided through mediaOffers.
func NegotiateContentTypeWithMediaTypes(r *http.Request, mediaOffers []MediaType) (parsed, valid bool, selected MediaType) {
	if len(mediaOffers) == 0 {
		panic("raggett: NegotiateContentType called with empty offers")
	}
	accept := r.Header.Get("Accept")
	if strings.TrimSpace(accept) == "" {
		return true, true, preferablyNonVendorOffer(mediaOffers)
	}

	if accept == "*" {
		// Java's URLConnection class sends an Accept header including
		// single '*'...
		accept = "*/*"
	}

	ok, medias := parseAcceptHeader(accept)
	if !ok {
		return false, false, preferablyNonVendorOffer(mediaOffers)
	}

	sort.SliceStable(medias, func(i, j int) bool {
		return medias[i].rank() < medias[j].rank()
	})

	negotiationOK, result := doNegotiation(medias, mediaOffers)
	return true, negotiationOK, result
}

func preferablyNonVendorOffer(offers []MediaType) MediaType {
	for _, v := range offers {
		if !v.IsVendor {
			return v
		}
	}
	return offers[0]
}

func doNegotiation(medias []MediaType, offers []MediaType) (bool, MediaType) {
	// at this point, medias is already sorted.
	if len(medias) > 1 {
		if medias[0].rank() == medias[1].rank() {
			// In this case we have a tie. Return the more specific one,
			// considering vendored types.
			if medias[0].IsVendor && !medias[1].IsVendor {
				// medias[1] has to go.
				medias = append(medias[0:1], medias[2:]...)
			} else {
				// medias[0] has to go.
				medias = medias[1:]
			}
		}
	}

	for _, m := range medias {
		for _, o := range offers {
			if m.matches(o) {
				return true, MediaType{
					TypeString:    o.TypeString,
					SubTypeString: o.SubTypeString,
					Parameters:    m.Parameters,
					Weight:        m.Weight,
					Specificity:   m.Specificity,
				}
			}
		}
	}

	// At this point, we don't have a better candidate. So let's try to find a
	// non-vendored one, and fallback...
	return false, preferablyNonVendorOffer(offers)

}

func isAllowedTypeChar(c rune) bool {
	return c == '!' || c == '#' || c == '$' || c == '%' || c == '&' ||
		c == '\'' || c == '*' || c == '+' || c == '-' || c == '.' ||
		c == '^' || c == '_' || c == '`' || c == '|' || c == '~' ||
		unicode.IsDigit(c) || unicode.IsLetter(c)
}

func parseAcceptHeader(s string) (bool, []MediaType) {
	type contentTypeState int
	const (
		contentTypeStateType contentTypeState = iota
		contentTypeStateSubType
		contentTypeStateParametersSemiOrColon
		contentTypeStateParameterKey
		contentTypeStateParameterValue
		contentTypeStateParameterQuote
	)

	st := contentTypeStateType
	cur := 0
	sLen := len(s)

	var tmpStr []rune
	var tmpKey string
	tmpMedia := MediaType{}
	var result []MediaType

	appendResult := func() {
		if tmpMedia.TypeString == "" || tmpMedia.SubTypeString == "" {
			return
		}
		result = append(result, tmpMedia)
		tmpMedia = MediaType{}
	}

	appendParam := func() {
		if tmpMedia.Parameters == nil {
			tmpMedia.Parameters = map[string][]string{
				tmpKey: {string(tmpStr)},
			}
		} else {
			tmpMedia.Parameters[tmpKey] = append(tmpMedia.Parameters[tmpKey], string(tmpStr))
		}
		tmpStr = tmpStr[:0]
		tmpKey = ""
	}

	for cur < sLen {
		c := rune(s[cur])
		if st != contentTypeStateParameterQuote && (c == ' ' || c == '\n') {
			cur++
			continue
		}

		switch st {
		case contentTypeStateType:
			if c == '/' {
				tmpMedia.TypeString = string(tmpStr)
				tmpStr = tmpStr[:0]
				st = contentTypeStateSubType
			} else {
				if !isAllowedTypeChar(c) {
					return false, nil
				}
				tmpStr = append(tmpStr, c)
			}
		case contentTypeStateSubType:
			if c == ',' || c == ';' {
				if len(tmpStr) == 0 {
					return false, nil
				}
				tmpMedia.SubTypeString = string(tmpStr)
				tmpStr = tmpStr[:0]

				if c == ',' {
					st = contentTypeStateType
					appendResult()
					break
				}

				st = contentTypeStateParameterKey
			} else {
				if !isAllowedTypeChar(c) {
					return false, nil
				}
				tmpStr = append(tmpStr, c)
			}
		case contentTypeStateParameterKey:
			if c == '=' {
				tmpKey = string(tmpStr)
				tmpStr = tmpStr[:0]
				st = contentTypeStateParameterValue
			} else {
				tmpStr = append(tmpStr, c)
			}
		case contentTypeStateParameterValue:
			if c == '"' && len(tmpStr) == 0 {
				st = contentTypeStateParameterQuote
				cur++
				continue
			}

			if c == ';' || c == ',' {
				appendParam()

				if c == ',' {
					st = contentTypeStateType
					appendResult()
					cur++
					continue
				}

				st = contentTypeStateParameterKey
				cur++
				continue
			}
			tmpStr = append(tmpStr, c)
		case contentTypeStateParameterQuote:
			if c == '\\' {
				if cur+1 >= sLen {
					// Unexpected EOF
					return false, nil
				}
				next := s[cur+1]
				if next == '"' {
					tmpStr = append(tmpStr, '"')
					cur += 2
					continue
				}
			}
			if c == '"' {
				appendParam()
				st = contentTypeStateParametersSemiOrColon
				break
			}
			tmpStr = append(tmpStr, c)
		case contentTypeStateParametersSemiOrColon:
			if c == ';' {
				st = contentTypeStateParameterKey
			} else if c == ',' {
				st = contentTypeStateType
			} else {
				return false, nil
			}
		}
		cur++

	}

	if st != contentTypeStateType {
		switch st {
		case contentTypeStateParameterValue:
			if len(tmpStr) == 0 {
				return false, nil
			}
			appendParam()
		case contentTypeStateParametersSemiOrColon:
			// When waiting for a semi or colon, we can also reach EOF.
			break
		case contentTypeStateSubType:
			if len(tmpStr) == 0 {
				return false, nil
			}
			tmpMedia.SubTypeString = string(tmpStr)
			tmpStr = tmpStr[:0]
		default:
			return false, nil
		}
	}

	appendResult()
	if len(result) == 0 {
		return false, nil
	}
	arr := make([]MediaType, 0, len(result))

	for _, t := range result {
		if q, ok := t.Parameters["q"]; ok {
			parsed, err := strconv.ParseFloat(q[0], 32)
			if err == nil {
				t.Weight = float32(parsed)
			} else {
				return false, nil
			}
		} else {
			t.Weight = 1
		}

		if t.SubTypeString == "*" && t.TypeString == "*" {
			t.Specificity = MediaTypeSpecificityUndefined
		} else if t.SubTypeString == "*" || t.TypeString == "*" {
			t.Specificity = MediaTypeSpecificityPartiallyDefined
		} else {
			t.Specificity = MediaTypeSpecificityFullyDefined
		}

		t.IsVendor = strings.Contains(t.SubTypeString, "+")

		t.SubTypeString = strings.ToLower(t.SubTypeString)
		t.TypeString = strings.ToLower(t.TypeString)

		arr = append(arr, t)
	}

	return true, arr
}

// MediaTypeSpecificity indicates the specificity level of a MediaType instance.
type MediaTypeSpecificity int

const (
	// MediaTypeSpecificityFullyDefined indicates the media type is fully
	// defined, containing both a type and subtype.
	MediaTypeSpecificityFullyDefined MediaTypeSpecificity = iota

	// MediaTypeSpecificityPartiallyDefined indicates the media type has a
	// wildcard value on either its Type or SubType.
	MediaTypeSpecificityPartiallyDefined

	// MediaTypeSpecificityUndefined indicates the media type contains wildcards
	// on both Type and SubType.
	MediaTypeSpecificityUndefined
)

// MediaType represents a media type object commonly transferred through Accept
// and Content-Type headers.
type MediaType struct {
	// TypeString represents the main type of the media type. For instance, for
	// "text/plain", TypeString will contain "text".
	TypeString string

	// SubTypeString represents the main type of the media type. For instance,
	// for "text/plain", SubTypeString will contain "plain".
	SubTypeString string

	// Parameters represents the list of parameters for a parsed media type.
	// For instance, for "text/plain;foo=bar", Parameters will have as its
	// contents a map[string][]string{"foo": {"bar"}}
	Parameters map[string][]string

	// Weight indicates the weight of this particular media type. For instance,
	// for "text/plain;q=0.8", Weight will equal 0.8.
	Weight float32

	// Specificity indicates the specificity level of this MediaType. See
	// MediaTypeSpecificity for further information.
	Specificity MediaTypeSpecificity

	// IsVendor indicates whether the media type is vendored. For instance,
	// "application/vnd.raggett.example+json" is considered to be vendored.
	IsVendor bool
}

// MediaTypeFromString takes a type and subType strings, and returns a MediaType
// object representing that type.
func MediaTypeFromString(mediaType, subType string) MediaType {
	return MediaType{
		TypeString:    mediaType,
		SubTypeString: subType,
		IsVendor:      strings.Contains(subType, "+"),
		Parameters:    nil,
		Weight:        1,
		Specificity:   MediaTypeSpecificityFullyDefined,
	}
}

// UTF8MediaTypeFromString is a utility method that invokes MediaTypeFromString
// passing the provided parameters, but adds an extra "charset" parameter
// containing the string "utf-8".
func UTF8MediaTypeFromString(mediaType, subType string) MediaType {
	t := MediaTypeFromString(mediaType, subType)
	t.Parameters = map[string][]string{"charset": {"utf-8"}}
	return t
}

func (mt MediaType) equals(m MediaType) bool {
	return m.TypeString == mt.TypeString &&
		m.SubTypeString == mt.SubTypeString &&
		m.Weight == mt.Weight &&
		m.Specificity == mt.Specificity &&
		reflect.DeepEqual(mt.Parameters, m.Parameters)
}

func (mt MediaType) rank() float32 {
	base := 0
	switch mt.Specificity {
	case MediaTypeSpecificityPartiallyDefined:
		base = 1
	case MediaTypeSpecificityFullyDefined:
		base = 2
	}
	base += len(mt.Parameters)
	return float32(base) * mt.Weight
}

func (mt MediaType) matches(other MediaType) bool {
	// "other" must always be FullyDefined
	switch mt.Specificity {
	case MediaTypeSpecificityFullyDefined:
		return other.TypeString == mt.TypeString && other.SubTypeString == mt.SubTypeString && other.IsVendor == mt.IsVendor
	case MediaTypeSpecificityPartiallyDefined:
		if mt.TypeString == "*" {
			return other.SubTypeString == mt.SubTypeString && other.IsVendor == mt.IsVendor
		} else {
			return other.TypeString == mt.TypeString && other.IsVendor == mt.IsVendor
		}
	default:
		return !other.IsVendor
	}
}

// Type returns the media type's type string without parameters.
func (mt MediaType) Type() string {
	return fmt.Sprintf("%s/%s", mt.TypeString, mt.SubTypeString)
}

// String returns the media type string including any provided parameters.
func (mt MediaType) String() string {
	var params []string
	for p, vArr := range mt.Parameters {
		for _, v := range vArr {
			params = append(params, fmt.Sprintf("%s=%s", p, v))
		}
	}
	paramsStr := ""
	if len(params) > 0 {
		paramsStr = ";" + strings.Join(params, ";")
	}
	return fmt.Sprintf("%s%s", mt.Type(), paramsStr)
}
