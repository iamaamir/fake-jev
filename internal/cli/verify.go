package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	nethttp "net/http"
	"net/url"
	"strings"
	"time"
)

// verificationPath is the control endpoint §15.3 requires the CLI to call.
// Every verification decision — matching, expectation evaluation, failure
// ordering — belongs to the server (§11.4, §41.9, C-CLI-008).
const verificationPath = "/__fake/v1/verify"

// verificationTimeout bounds one verify call. §15.3 defines no timeout; §42.1
// makes an unresponsive endpoint an operational failure, which needs a bound.
const verificationTimeout = 10 * time.Second

// verificationFailure is the fixed §41.9 failure item shape.
type verificationFailure struct {
	Code            string  `json:"code"`
	Message         string  `json:"message"`
	RequestSequence *uint64 `json:"requestSequence"`
	StubID          *string `json:"stubId"`
}

// verificationReport is the §41.9 verify envelope the CLI acts on.
type verificationReport struct {
	Passed   bool
	Failures []verificationFailure
}

// verifyDocument decodes the wire body. Passed is a pointer so an absent or
// non-boolean field is distinguishable from `passed: false`: §41.9 fixes the
// boolean, so anything else means the result could not be obtained.
type verifyDocument struct {
	Passed   *bool                 `json:"passed"`
	Failures []verificationFailure `json:"failures"`
}

// callVerification performs the single §15.3 control call against baseURL and
// decodes the §41.9 result. It transports and decodes only: the returned
// failures are reported in the order the server produced them, and no
// verification logic is reimplemented here (§15.3, C-CLI-008).
func callVerification(client *nethttp.Client, baseURL string) (verificationReport, error) {
	target := strings.TrimSuffix(baseURL, "/") + verificationPath
	request, err := nethttp.NewRequest(nethttp.MethodGet, target, nil)
	if err != nil {
		return verificationReport{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return verificationReport{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != nethttp.StatusOK {
		// §41.9 answers 200 whenever the result can be calculated, so any other
		// status means the result could not be obtained (§42.1).
		return verificationReport{}, fmt.Errorf("GET %s: unexpected status %d", target, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return verificationReport{}, fmt.Errorf("GET %s: %w", target, err)
	}
	var document verifyDocument
	if err := json.Unmarshal(body, &document); err != nil {
		return verificationReport{}, fmt.Errorf("GET %s: %w", target, err)
	}
	if document.Passed == nil {
		return verificationReport{}, fmt.Errorf("GET %s: response carries no boolean \"passed\" field", target)
	}
	return verificationReport{Passed: *document.Passed, Failures: document.Failures}, nil
}

// writeVerificationFailure reports the §41.9 failures on stderr. The wording is
// not contract (§42.5); the stream, the exit code, and the server-supplied
// codes are.
func writeVerificationFailure(w io.Writer, baseURL string, report verificationReport) {
	writef(w, "fake-jev: %s: verification failed\n", baseURL)
	for _, failure := range report.Failures {
		writef(w, "fake-jev: %s: %s: %s", baseURL, failure.Code, failure.Message)
		if failure.RequestSequence != nil {
			writef(w, " (requestSequence %d)", *failure.RequestSequence)
		}
		if failure.StubID != nil {
			writef(w, " (stubId %s)", *failure.StubID)
		}
		writef(w, "\n")
	}
}

// runVerify implements §15.3: one GET <url>/__fake/v1/verify, exit 0 when the
// endpoint reports passed, 3 when it reports a verification failure, and 2 when
// the result cannot be obtained (§42.1). It loads no configuration and computes
// no failures of its own (§15.3).
func runVerify(args []string, stdout, stderr io.Writer) int {
	baseURL, err := parseVerifyFlags(args)
	if errors.Is(err, flag.ErrHelp) {
		writef(stdout, "%s", usage)
		return exitOK
	}
	if err != nil {
		return usageError(stderr, "%s", err)
	}
	report, err := callVerification(&nethttp.Client{Timeout: verificationTimeout}, baseURL)
	if err != nil {
		writef(stderr, "fake-jev: %v\n", err)
		return exitFailure
	}
	if !report.Passed {
		writeVerificationFailure(stderr, baseURL, report)
		return exitVerify
	}
	writef(stdout, "fake-jev: %s: verification passed\n", baseURL)
	return exitOK
}

// parseVerifyFlags parses the §15.3 command line. --url is REQUIRED. A value
// the command cannot call is a usage failure (§42.1): §15.3 shows the plain
// base form, and a path, query, or fragment has no defined meaning for the
// control call, so those fail closed rather than being silently dropped (§4.6).
func parseVerifyFlags(args []string) (string, error) {
	var target string
	set := flag.NewFlagSet("verify", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.Usage = func() {}
	set.StringVar(&target, "url", "", "fake-jev server base URL")
	if err := set.Parse(args); err != nil {
		return "", err
	}
	if extra := set.Args(); len(extra) > 0 {
		return "", fmt.Errorf("unexpected argument %q", extra[0])
	}
	if target == "" {
		return "", errors.New("verify requires --url <url>")
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("--url %q: %v", target, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("--url %q: scheme must be http or https", target)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("--url %q: missing host", target)
	}
	if strings.TrimSuffix(parsed.Path, "/") != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("--url %q: must be a base URL with no path, query, or fragment", target)
	}
	return parsed.String(), nil
}
