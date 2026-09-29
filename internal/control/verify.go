package control

import (
	"net/http"
	"sort"
)

// verificationFailure is the fixed §41.9 failure item shape.
type verificationFailure struct {
	Code            string  `json:"code"`
	Message         string  `json:"message"`
	RequestSequence *uint64 `json:"requestSequence"`
	StubID          *string `json:"stubId"`
}

// verificationResult is the §41.9 envelope. The endpoint returns it with HTTP
// 200; a failed verification is carried in the body, never the status code.
type verificationResult struct {
	Passed   bool                  `json:"passed"`
	Failures []verificationFailure `json:"failures"`
}

// verify calculates the verification result from engine state alone. It never
// derives a failure from an interaction record, so an intentionally configured
// raw provider error (for example HTTP 429) cannot become a failure (§11.4).
func (a *API) verify(w http.ResponseWriter) {
	failures := a.interactionFailures()
	failures = append(failures, a.expectationFailures()...)
	if failures == nil {
		failures = []verificationFailure{}
	}
	writeJSON(w, http.StatusOK, verificationResult{Passed: len(failures) == 0, Failures: failures})
}

// interactionFailures returns engine-recorded failures ordered by request
// sequence ascending (§41.9). Sticky failures with no assigned sequence, such
// as journal_full, carry requestSequence: null and are ordered after the
// sequenced failures.
func (a *API) interactionFailures() []verificationFailure {
	recorded := a.engine.VerificationFailures()
	sort.SliceStable(recorded, func(i, j int) bool {
		left, right := recorded[i].RequestSequence, recorded[j].RequestSequence
		if left == nil || right == nil {
			// A sequenced failure sorts before a null-sequence one; equal keys
			// (including two null keys) keep their recorded order.
			return left != nil
		}
		return *left < *right
	})
	failures := make([]verificationFailure, 0, len(recorded))
	for _, failure := range recorded {
		failures = append(failures, verificationFailure{
			Code:            failure.Code,
			Message:         failure.Message,
			RequestSequence: failure.RequestSequence,
			StubID:          failure.StubID,
		})
	}
	return failures
}

// expectationFailures returns unsatisfied invocation expectations in stub
// registration index ascending order (§41.9).
func (a *API) expectationFailures() []verificationFailure {
	evaluated := a.engine.EvaluateExpectations()
	failures := make([]verificationFailure, 0, len(evaluated))
	for _, failure := range evaluated {
		stubID := failure.StubID
		failures = append(failures, verificationFailure{
			Code:    failure.Code,
			Message: failure.Message,
			StubID:  &stubID,
		})
	}
	return failures
}
