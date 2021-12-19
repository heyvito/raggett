package raggett

import "net/http"

type responseProxy struct {
	status   int
	original http.ResponseWriter
}

func (p *responseProxy) Header() http.Header {
	return p.original.Header()
}

func (p *responseProxy) Write(bytes []byte) (int, error) {
	return p.original.Write(bytes)
}

func (p *responseProxy) WriteHeader(status int) {
	p.status = status
	p.original.WriteHeader(status)
}
