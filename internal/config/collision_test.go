package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestLoadNeverReturnsConfigThatValidateRejects pins the invariant that Load
// fails closed: it must not report success for a configuration that its own
// Validate rejects. A case-variant duplicate key defeats the raw-document
// checks in validateDocument because encoding/json silently overwrites the
// decoded field, so the corpus concentrates on that class.
func TestLoadNeverReturnsConfigThatValidateRejects(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"case variant with a different value", `{"schemaVersion":1,"sChemAVErsion":0}`},
		{"case variant with an equal value", `{"schemaVersion":1,"sChemAVErsion":1}`},
		{"upper case variant", `{"schemaVersion":1,"SCHEMAVERSION":0}`},
		{"yaml block case variant", "schemaVersion: 1\nschemaversion: 0\n"},
		{"case variant of another top level key", `{"schemaVersion":1,"mode":"strict","MODE":"permissive"}`},
		{"case variant of a nested field", `{"schemaVersion":1,"server":{"host":"127.0.0.1","HOST":""}}`},
		{"valid yaml defaults", "schemaVersion: 1\n"},
		{"valid json defaults", `{"schemaVersion":1}`},
		{"truncated json", `{"schemaVersion":1`},
		{"non object document", `[]`},
		{"null document", `null`},
		{"empty document", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := Load([]byte(test.body))
			if err != nil {
				return
			}
			if cfg == nil {
				t.Fatal("Load returned a nil configuration with a nil error")
			}
			if validateErr := Validate(cfg); validateErr != nil {
				t.Fatalf("Load returned %#v with a nil error, but Validate rejects it: %v", cfg, validateErr)
			}
		})
	}
}

// TestLoadRejectsCaseVariantDuplicateKeys pins that a document holding two keys
// that the decoder cannot tell apart is rejected for every known key, not only
// for schemaVersion, and at nested levels of the schema.
func TestLoadRejectsCaseVariantDuplicateKeys(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"lower camel case variant", `{"schemaVersion":1,"sChemAVErsion":0}`},
		{"upper case variant", `{"schemaVersion":1,"SCHEMAVERSION":0}`},
		{"lower case variant", `{"schemaVersion":1,"schemaversion":1}`},
		{"yaml block case variant", "schemaVersion: 1\nschemaversion: 0\n"},
		{"mode", `{"schemaVersion":1,"mode":"strict","MODE":"strict"}`},
		{"server", `{"schemaVersion":1,"server":{"host":"127.0.0.1","port":8787},"SERVER":{"host":"0.0.0.0","port":80}}`},
		{"limits", `{"schemaVersion":1,"limits":{"maxInteractions":1},"Limits":{"maxInteractions":2}}`},
		{"compatibility", `{"schemaVersion":1,"compatibility":["jev/v1"],"Compatibility":["jev/v1"]}`},
		{"models", `{"schemaVersion":1,"models":[{"name":"a","description":"a","release_date":"1970-01-01"}],"MODELS":[{"name":"b","description":"b","release_date":"1970-01-01"}]}`},
		{"stubs", `{"schemaVersion":1,"stubs":[],"STUBS":[]}`},
		{"nested server field", `{"schemaVersion":1,"server":{"host":"127.0.0.1","HOST":"0.0.0.0"}}`},
		{"nested when field", `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{"questions":{"a":"noul"},"QUESTIONS":{"b":"noul"}},"then":{"answers":{}}}]}`},
		{"nested raw response field", `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{},"HEADERS":{}}}}]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := Load([]byte(test.body))
			if err == nil {
				t.Fatalf("case-variant duplicate keys loaded successfully: %#v", cfg)
			}
			if cfg != nil {
				t.Fatalf("Load returned a non-nil configuration together with error %v: %#v", err, cfg)
			}
		})
	}
}

// foldSample spans the ASCII range, Latin-1, Greek, Cyrillic, Hebrew, CJK,
// emoji, and the non-ASCII members of the simple-fold cycles that contain an
// ASCII letter: U+212A KELVIN SIGN folds with k/K and U+017F LATIN SMALL LETTER
// LONG S folds with s/S, while U+01C4/U+01C5 (DŽ/Dž) fold only with each other.
// It is the rune set used to substitute single runes of a known field name.
var foldSample = []rune("aAzZ019.\u00c9\u00e9\u00df\u017f\u212a\u01c4\u01c5\u0391\u03b1\u0415\u0435\u05d0\u4e2d\U0001f600")

// TestLoadRejectsKeysThatAreNotExactlySpelled pins the exact-key half of
// specification §12.3 for every known top-level key: the only spelling that
// loads is the one the schema uses. encoding/json resolves an object key to a
// struct field case-insensitively, so a spelling that differs only in case (or,
// beyond ASCII, by a simple-fold equivalent rune such as U+017F for "s") is
// accepted by DisallowUnknownFields and silently writes the field; Load must
// reject it instead. The test substitutes one rune of a field name at a time
// with every rune of foldSample, so the whole decoder fold relation is covered,
// not a hand-picked list of cases.
func TestLoadRejectsKeysThatAreNotExactlySpelled(t *testing.T) {
	fields := []struct {
		name  string
		value string
	}{
		{"schemaVersion", "1"},
		{"server", "{}"},
		{"mode", `"strict"`},
		{"compatibility", `["jev/v1"]`},
		{"limits", "{}"},
		{"models", `[{"name":"a","description":"a","release_date":"1970-01-01"}]`},
		{"stubs", "[]"},
	}
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			acceptedBody := exactSpellingDocument(field.name, field.value)
			if field.name == "schemaVersion" {
				// The version key cannot appear twice: an exact repeat is a
				// duplicate key, which is a different rule.
				acceptedBody = `{"schemaVersion":1}`
			}
			accepted, err := Load([]byte(acceptedBody))
			if err != nil || accepted == nil {
				t.Fatalf("Load rejected the exact spelling: %v", err)
			}
			name := []rune(field.name)
			for position := range name {
				original := name[position]
				for _, substitute := range foldSample {
					name[position] = substitute
					candidate := string(name)
					name[position] = original
					if candidate == field.name {
						continue
					}
					body := exactSpellingDocument(candidate, field.value)
					cfg, err := Load([]byte(body))
					if err == nil {
						t.Errorf("Load(%s) = %#v with a nil error, want an error: %q differs from %q", body, cfg, candidate, field.name)
						continue
					}
					if cfg != nil {
						t.Errorf("Load(%s) returned a non-nil configuration with error %v", body, err)
					}
					if strings.EqualFold(candidate, field.name) && !strings.Contains(err.Error(), candidate) {
						t.Errorf("Load(%s) error %v does not name the rejected key %q, so the decoder fold match was not caught", body, err, candidate)
					}
				}
			}
		})
	}
}

// exactSpellingDocument builds a document carrying the exact schemaVersion key
// and one key spelled key, so a substitution of the schemaVersion field name is
// still exercised in a document that has the required version.
func exactSpellingDocument(key, value string) string {
	return fmt.Sprintf(`{"schemaVersion":1,%s:%s}`, mustJSONString(key), value)
}

func mustJSONString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

// TestLoadTerminatesOnUncasedKeys pins that Load answers instead of spinning for
// an object key that is a code point with no case mapping (CJK, emoji, symbols,
// and any rune outside a multi-rune fold set). Such a key matches no schema
// field, so the answer is the ordinary unknown-key error; the point of the test
// is that a call returns at all, because a fold helper that assumed
// unicode.SimpleFold always moves to a different rune never terminated here.
func TestLoadTerminatesOnUncasedKeys(t *testing.T) {
	runes := append([]rune("\u00e9\u0391\u0435"), foldSample...)
	documents := []struct {
		name string
		body string
	}{
		{"top level", `{"schemaVersion":1,%s:1}`},
		{"struct level object", `{"schemaVersion":1,"server":{%s:1}}`},
	}
	for _, document := range documents {
		for _, r := range runes {
			body := fmt.Sprintf(document.body, mustJSONString(string(r)))
			cfg, err := Load([]byte(body))
			if err == nil {
				t.Errorf("Load(%s) = %#v with a nil error, want an error for the unknown key %q", body, cfg, string(r))
				continue
			}
			if cfg != nil {
				t.Errorf("Load(%s) returned a non-nil configuration with error %v", body, err)
			}
		}
	}
}

// TestLoadAcceptsCaseVariantKeysInOpenContainers pins specification §12.3 and
// §38.8: the containers the specification declares open are exempt from
// project-owned key rules, so nothing there is compared against a field name and
// two keys differing only in case stay two distinct keys with both payloads
// preserved. The uncased CJK and emoji keys are included because they are the
// code points on which an earlier fold helper caused Load to hang.
func TestLoadAcceptsCaseVariantKeysInOpenContainers(t *testing.T) {
	t.Run("raw body", func(t *testing.T) {
		const body = `{"a":1,"A":2,"\u4e2d":3,"emoji\ud83d\ude42":4}`
		cfg := loadAccepted(t, `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{},"body":`+body+`}}}]}`)
		assertRawPayload(t, cfg.Stubs[0].Then.Raw.Body, body)
	})

	t.Run("sequence raw body", func(t *testing.T) {
		const body = `{"Ok":1,"ok":2}`
		cfg := loadAccepted(t, `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"sequence":[{"raw":{"status":200,"headers":{},"body":`+body+`}}]}}]}`)
		assertRawPayload(t, cfg.Stubs[0].Then.Sequence[0].Raw.Body, body)
	})

	t.Run("when state", func(t *testing.T) {
		const state = `{"Key":1,"KEY":2}`
		cfg := loadAccepted(t, `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{"state":`+state+`},"then":{"answers":{"a":{"noul":1}}}}]}`)
		assertRawPayload(t, cfg.Stubs[0].When.State, state)
	})

	t.Run("then answers", func(t *testing.T) {
		const answers = `{"Kind":{"noul":1},"KIND":{"noul":1}}`
		cfg := loadAccepted(t, `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{"questions":{"Kind":"noul","KIND":"noul"}},"then":{"answers":`+answers+`}}]}`)
		assertRawPayload(t, cfg.Stubs[0].Then.Answers, answers)
	})

	t.Run("question names", func(t *testing.T) {
		cfg := loadAccepted(t, `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{"questions":{"Kind":"noul","KIND":"choice"}},"then":{"answers":{"Kind":{"noul":1},"KIND":{"choice":"a","probabilities":{"a":1}}}}}]}`)
		questions := cfg.Stubs[0].When.Questions
		if len(questions) != 2 || questions["Kind"] != "noul" || questions["KIND"] != "choice" {
			t.Fatalf("when.questions = %v, want both Kind and KIND", questions)
		}
	})

	t.Run("raw header names", func(t *testing.T) {
		cfg := loadAccepted(t, `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{"X-Rate-Limit":"1","x-rate-limit":"2"},"body":{}}}}]}`)
		headers := cfg.Stubs[0].Then.Raw.Headers
		if len(headers) != 2 || headers["X-Rate-Limit"] != "1" || headers["x-rate-limit"] != "2" {
			t.Fatalf("then.raw.headers = %v, want both spellings preserved", headers)
		}
	})

	t.Run("exact duplicate keys stay rejected", func(t *testing.T) {
		// The document-wide rejection of a repeated exact key predates this
		// change and is not narrowed by it: yaml.v3 rejects the repeated key
		// before any open container is reached.
		if cfg, err := Load([]byte(`{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{},"body":{"a":1,"a":2}}}}]}`)); err == nil {
			t.Fatalf("Load = %#v with a nil error, want the exact duplicate key rejected", cfg)
		}
	})
}

// loadAccepted loads a document that must be accepted and fails the test with
// the diagnostic otherwise.
func loadAccepted(t *testing.T, body string) *Config {
	t.Helper()
	cfg, err := Load([]byte(body))
	if err != nil {
		t.Fatalf("Load(%s) = %v, want success", body, err)
	}
	if cfg == nil {
		t.Fatalf("Load(%s) returned a nil configuration with a nil error", body)
	}
	return cfg
}

// assertRawPayload fails the test unless raw carries want, the payload as the
// document spelled it.
func assertRawPayload(t *testing.T, raw json.RawMessage, want string) {
	t.Helper()
	if string(raw) != want {
		t.Fatalf("payload = %s, want %s", string(raw), want)
	}
}
