package raggett

import (
	"net/http"
	"net/http/httputil"
	"testing"

	"github.com/stretchr/testify/require"
)

const enableRequestDump = true

func dumpRequest(t *testing.T, r *http.Request) {
	if enableRequestDump {
		d, err := httputil.DumpRequest(r, true)
		require.NoError(t, err)
		t.Logf("Notice: Request dump:\n%s", string(d))
	}
}
