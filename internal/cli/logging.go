package cli

import (
	"io"
	"log"
	nethttp "net/http"
	"strconv"
)

// redactedHeaderValue replaces a secret header value in an operational line.
// §20 and §42.5 forbid logging an Authorization value; only the header's
// presence is observable.
const redactedHeaderValue = "[redacted]"

// logTruncationMarker terminates a body preview that was cut at the configured
// logBodyBytes limit (§19.4, §20). It is explicit so a reader can tell a
// truncated preview from a complete one.
const logTruncationMarker = "...[truncated]"

// operationalLogger writes the served-request operational line for the HTTP
// host that serve and run share. It enforces Authorization redaction and
// body-preview truncation on that one path, so no served-request line can carry
// a secret header value or an unbounded body (§19.4, §20, §42.5). Lifecycle and
// net/http error diagnostics use the underlying *log.Logger directly and never
// carry request data; nothing here writes to stdout.
type operationalLogger struct {
	logger    *log.Logger
	bodyLimit int
}

// newOperationalLogger binds the resolved logBodyBytes limit to the supplied
// request-line sink. A negative limit is treated as zero, so a preview can
// never be unbounded.
func newOperationalLogger(logger *log.Logger, bodyLimit int) *operationalLogger {
	if bodyLimit < 0 {
		bodyLimit = 0
	}
	return &operationalLogger{logger: logger, bodyLimit: bodyLimit}
}

// logRequest writes one operational line for a served request on behalf of the
// surrounding logHandler. The wrapper reaches it once per request the host
// handler sees, whatever the host decides (matched, unmatched, rejected,
// control, or unknown route), so redaction and truncation hold on those paths
// including failure paths. A request net/http rejects before the handler, a nil
// logger, or a nil request produces no line.
func (operational *operationalLogger) logRequest(request *nethttp.Request, preview string, truncated bool) {
	if operational == nil || operational.logger == nil || request == nil {
		return
	}
	path := ""
	if request.URL != nil {
		path = request.URL.Path
	}
	line := request.Method + " " + path
	if value := request.Header.Get("Authorization"); value != "" {
		line += " authorization=" + redactAuthorization(value)
	}
	if preview != "" || truncated {
		line += " body=" + preview
		if truncated {
			line += logTruncationMarker
		}
	}
	operational.logger.Print(line)
}

// redactAuthorization returns the value recorded for an Authorization header.
// §20/§42.5 forbid logging the value; only its presence is observable.
func redactAuthorization(value string) string {
	if value == "" {
		return ""
	}
	return redactedHeaderValue
}

// logHandler writes the per-request operational line. It attaches a bounded
// observer to the request body before delegating, so the preview is captured
// without reading anything ahead of the host, and it logs after the response so
// the line describes a completed interaction (§19.4, §20).
type logHandler struct {
	next   nethttp.Handler
	logger *operationalLogger
}

func (handler logHandler) ServeHTTP(writer nethttp.ResponseWriter, request *nethttp.Request) {
	if handler.logger == nil {
		// A missing logger must not take the server down: delegate without the
		// operational line rather than panic (fail closed for the request path).
		handler.next.ServeHTTP(writer, request)
		return
	}
	preview := captureBodyPreview(request, handler.logger.bodyLimit)
	handler.next.ServeHTTP(writer, request)
	body, truncated := preview()
	handler.logger.logRequest(request, body, truncated)
}

// captureBodyPreview installs a bounded observer on the request body and returns
// a function that renders the captured preview once the request has been served.
// The observer never reads the body itself: it records only the bytes the host
// consumes, so an oversized request still reaches the host's §19.4/§43.7 early
// rejection before a single body byte is read, and the preview can never hold
// more than the configured logBodyBytes nor more than the host's plane limit
// allowed through (§19.4, §20). A negative or zero limit is safe and yields an
// empty preview; it never panics.
func captureBodyPreview(request *nethttp.Request, bodyLimit int) func() (string, bool) {
	if request == nil || request.Body == nil || request.Body == nethttp.NoBody {
		return func() (string, bool) { return "", false }
	}
	if bodyLimit < 0 {
		bodyLimit = 0
	}
	observer := &bodyPreview{body: request.Body, limit: bodyLimit}
	request.Body = observer
	return func() (string, bool) {
		return renderBodyPreview(observer.buffer), observer.truncated
	}
}

// bodyPreview observes a request body as the host reads it, retaining at most
// limit bytes. Recording is by bytes, matching the byte-counted §19.4 limit; the
// captured bytes are rendered log-safe afterwards so a cut cannot split a log
// line (§20).
type bodyPreview struct {
	body      io.ReadCloser
	buffer    []byte
	limit     int
	truncated bool
}

func (preview *bodyPreview) Read(chunk []byte) (int, error) {
	read, err := preview.body.Read(chunk)
	if read > 0 {
		preview.record(chunk[:read])
	}
	return read, err
}

// Close forwards to the wrapped body, so the host's deferred Close still closes
// the real request body.
func (preview *bodyPreview) Close() error { return preview.body.Close() }

// record keeps the first limit bytes and flags truncation on the first byte
// beyond it. A limit of zero or less flags truncation on the first byte rather
// than slicing, so no call can panic.
func (preview *bodyPreview) record(chunk []byte) {
	if preview.truncated {
		return
	}
	room := preview.limit - len(preview.buffer)
	if room <= 0 {
		preview.truncated = true
		return
	}
	if len(chunk) > room {
		preview.buffer = append(preview.buffer, chunk[:room]...)
		preview.truncated = true
		return
	}
	preview.buffer = append(preview.buffer, chunk...)
}

// renderBodyPreview renders captured request-body bytes for the single-line
// operational log as a Go quoted string, so control characters, newlines, and CR
// cannot forge additional log lines and a byte cut that splits a UTF-8 sequence
// is rendered as \xNN instead of invalid output (§20). The cut itself is by
// bytes — deterministic, and matching the byte-counted §19.4 limit. An empty
// capture renders as "", so an empty body adds no body field to the line.
func renderBodyPreview(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	return strconv.Quote(string(raw))
}
