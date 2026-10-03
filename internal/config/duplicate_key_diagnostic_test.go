package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// maxDuplicateKeyDiagnosticBytes bounds the diagnostic reported for a document
// that repeats one object key. The number is an implementation choice, not a
// specification limit: the specification bounds neither a control-plane
// response nor a diagnostic message. It contains the §41.1 error envelope, the
// fixed error code, and one diagnostic naming a key.
const maxDuplicateKeyDiagnosticBytes = 512

// maxDuplicateKeyDiagnosticAllocationFactor bounds what one bounded diagnostic
// may allocate as a multiple of the document length. Before this bound the same
// document allocated at least the whole quadratic diagnostic, which for a
// 30 921-byte document is the 271 275 340 bytes measured for `validate` at
// c70d55b, roughly 8 700 times the document.
const maxDuplicateKeyDiagnosticAllocationFactor = 256

// repeatedKeyObject returns a JSON object that repeats the key "id" n times.
// This is the shape of the control-plane createStub body: one small object that
// repeats a single key.
func repeatedKeyObject(n int) string {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf(`"id":%d`, i)
	}
	return "{" + strings.Join(keys, ",") + "}"
}

func repeatedKeyConfiguration(n int) []byte {
	return []byte(`{"schemaVersion":1,"stubs":[` + repeatedKeyObject(n) + `]}`)
}

func repeatedKeyYAML(n int, flow bool) []byte {
	separator := "\n"
	if flow {
		separator = ", "
	}
	entries := make([]string, n)
	for i := range entries {
		entries[i] = fmt.Sprintf("a: %d", i)
	}
	if flow {
		return []byte("{" + strings.Join(entries, separator) + "}")
	}
	return []byte(strings.Join(entries, separator) + separator)
}

// allocatedBytes reports how many bytes f allocated in total. It is used to
// measure the cost the duplicate-key diagnostic itself adds.
func allocatedBytes(f func()) uint64 {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// TestLoadBoundsRepeatedJSONKeyDiagnostic pins that the JSON duplicate-key
// diagnostic is one line naming the repeated key, that it is byte-identical for
// every repetition count, and that it stays inside maxDuplicateKeyDiagnosticBytes.
// The JSON scan reaches the repeated key in document order, so both the message
// and the work are bounded by the position of the first repeat and not by how
// often the key repeats further on.
func TestLoadBoundsRepeatedJSONKeyDiagnostic(t *testing.T) {
	const want = `decode JSON configuration: duplicate object key "id"`
	first := ""
	for _, count := range []int{2, 3, 16, 256, 1600, 3200} {
		document := repeatedKeyConfiguration(count)
		_, err := Load(document)
		if err == nil {
			t.Fatalf("%d repeated keys were accepted; a repeated object key must stay rejected", count)
		}
		message := err.Error()
		if message != want {
			t.Fatalf("diagnostic for %d repeated keys = %q, want %q", count, message, want)
		}
		if len(message) > maxDuplicateKeyDiagnosticBytes {
			t.Fatalf("diagnostic for %d repeated keys is %d bytes, above the %d-byte bound", count, len(message), maxDuplicateKeyDiagnosticBytes)
		}
		if first == "" {
			first = message
			continue
		}
		if message != first {
			t.Fatalf("diagnostic for %d repeated keys = %q, first diagnostic was %q", count, message, first)
		}
	}
}

// TestLoadBoundsRepeatedYAMLKeyDiagnostic pins the same bound for documents that
// only the YAML decoder can read: yaml.v3 reports one "already defined" line per
// repeated-key pair, so a bounded message must keep exactly one of them. The
// allocation logged here records the residual cost that stays inside yaml.v3.
func TestLoadBoundsRepeatedYAMLKeyDiagnostic(t *testing.T) {
	forms := []struct {
		name string
		flow bool
	}{
		{"block mapping", false},
		{"flow mapping", true},
	}
	for _, form := range forms {
		t.Run(form.name, func(t *testing.T) {
			first := ""
			for _, count := range []int{2, 3, 16, 256, 800} {
				document := repeatedKeyYAML(count, form.flow)
				allocated := allocatedBytes(func() {
					_, err := Load(document)
					if err == nil {
						t.Fatalf("%d repeated keys were accepted", count)
					}
					message := err.Error()
					if got := strings.Count(message, "already defined at line"); got != 1 {
						t.Fatalf("diagnostic for %d repeated keys contains %d duplicate-key lines, want 1", count, got)
					}
					if !strings.HasPrefix(message, "decode YAML configuration: yaml: unmarshal errors:\n  line ") {
						t.Fatalf("diagnostic for %d repeated keys = %q, want one yaml.v3 unmarshal error naming the key", count, message)
					}
					if len(message) > maxDuplicateKeyDiagnosticBytes {
						t.Fatalf("diagnostic for %d repeated keys is %d bytes, above the %d-byte bound", count, len(message), maxDuplicateKeyDiagnosticBytes)
					}
					if first == "" {
						first = message
					} else if message != first {
						t.Fatalf("diagnostic for %d repeated keys = %q, first diagnostic was %q", count, message, first)
					}
				})
				t.Logf("%s: repeated keys=%d document=%d bytes allocated=%d bytes", form.name, count, len(document), allocated)
			}
		})
	}
}

// loadYAMLDiagnostic returns the yaml.v3 diagnostic list of a document that
// only the YAML decoder can read, with the wrapper text removed.
func loadYAMLDiagnostic(t *testing.T, body string) string {
	t.Helper()
	_, err := Load([]byte(body))
	if err == nil {
		t.Fatalf("document with a repeated key was accepted: %s", body)
	}
	const prefix = "decode YAML configuration: yaml: unmarshal errors:\n  "
	message := err.Error()
	if !strings.HasPrefix(message, prefix) {
		t.Fatalf("diagnostic = %q, want the yaml.v3 unmarshal error list", message)
	}
	return strings.TrimPrefix(message, prefix)
}

// TestYAMLDuplicateKeyDiagnosticMarkerPinsDecoderWording pins the wording that
// yamlDuplicateKeyDiagnostic matches. yaml.v3 exposes duplicate-key diagnostics
// only as formatted strings (yaml.TypeError.Errors is []string), so the filter
// cannot recognise them structurally and has to match text; this test is what
// makes an upgrade that rewords the line fail loudly instead of quietly
// restoring the unbounded diagnostic.
func TestYAMLDuplicateKeyDiagnosticMarkerPinsDecoderWording(t *testing.T) {
	// The wording the pinned dependency produces: gopkg.in/yaml.v3 v3.0.1,
	// decode.go:776, "line %d: mapping key %#v already defined at line %d".
	tests := []struct {
		name string
		body string
		want string
	}{
		{"block mapping", "a: 1\na: 2\n", `line 2: mapping key "a" already defined at line 1`},
		{"flow mapping", "{a: 1, a: 2}", `line 1: mapping key "a" already defined at line 1`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostic := loadYAMLDiagnostic(t, test.body)
			if diagnostic != test.want {
				t.Fatalf("yaml.v3 diagnostic = %q, want %q: the dependency reworded its duplicate-key line, so yamlDuplicateKeyDiagnostic no longer recognises it and the bound stops applying", diagnostic, test.want)
			}
			if !yamlDuplicateKeyDiagnostic(diagnostic) {
				t.Fatalf("yamlDuplicateKeyDiagnostic(%q) = false, want true", diagnostic)
			}
		})
	}
	// Other diagnostics yaml.v3 can produce must not be mistaken for the
	// duplicate-key line, or a filter bug would drop them.
	for _, diagnostic := range []string{
		"line 1: cannot unmarshal !!str `x` into int",
		"already defined at line 1",
		"line 1: field a already set in type config.Config",
		`line 3: mapping key "a" must be unique`,
	} {
		if yamlDuplicateKeyDiagnostic(diagnostic) {
			t.Fatalf("yamlDuplicateKeyDiagnostic(%q) = true, want false", diagnostic)
		}
	}
}

// TestLoadFileBoundsRepeatedYAMLKeyDiagnostic is the end-to-end guard for that
// textual marker. It reads a real YAML configuration file through LoadFile, the
// entry point internal/cli's validate and serve --config use, and asserts only
// what the product promises: the document is rejected, the message names the
// repeated key, and the message stays inside maxDuplicateKeyDiagnosticBytes and
// does not change with the repetition count. It never repeats the library's
// phrasing, so an upgrade that rewords the duplicate-key diagnostic makes
// yamlDuplicateKeyDiagnostic stop matching, lets all C(count,2) diagnostics
// through, and fails this test on the byte bound instead of passing silently.
func TestLoadFileBoundsRepeatedYAMLKeyDiagnostic(t *testing.T) {
	// repeatedKeyYAML repeats the block mapping key "a".
	const key = "a"
	first := ""
	for _, count := range []int{64, 512} {
		path := filepath.Join(t.TempDir(), fmt.Sprintf("duplicate-keys-%03d.yaml", count))
		if err := os.WriteFile(path, repeatedKeyYAML(count, false), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		_, err := LoadFile(path)
		if err == nil {
			t.Fatalf("repeated keys=%d: a document with a repeated key was accepted", count)
		}
		message := err.Error()
		if len(message) > maxDuplicateKeyDiagnosticBytes {
			t.Fatalf("repeated keys=%d: diagnostic is %d bytes, above the %d-byte bound; the duplicate-key diagnostic is no longer bounded: %q",
				count, len(message), maxDuplicateKeyDiagnosticBytes, message[:maxDuplicateKeyDiagnosticBytes])
		}
		// One bounded diagnostic line names both occurrences of the key, so the
		// quoted name appears twice; an unclamped list repeats the line C(count,2)
		// times.
		if named := strings.Count(message, `"`+key+`"`); named > 2 {
			t.Fatalf("repeated keys=%d: diagnostic names the key %d times, want at most 2: %q", count, named, message)
		}
		if first == "" {
			first = message
		} else if message != first {
			t.Fatalf("repeated keys=%d: diagnostic = %q, the first diagnostic was %q", count, message, first)
		}
	}
}

// TestLoadRepeatedKeyDiagnosticAllocationStaysBounded measures the cost of the
// JSON duplicate-key diagnostic across doubling steps. Every step must stay
// inside maxDuplicateKeyDiagnosticAllocationFactor times its own document
// length, so the cost does not grow with the repetition count.
func TestLoadRepeatedKeyDiagnosticAllocationStaysBounded(t *testing.T) {
	counts := []int{200, 400, 800, 1600, 3200}
	var smallest uint64
	for i, count := range counts {
		document := repeatedKeyConfiguration(count)
		allocated := allocatedBytes(func() {
			if _, err := Load(document); err == nil {
				t.Fatalf("%d repeated keys were accepted", count)
			}
		})
		t.Logf("repeated JSON keys=%d document=%d bytes allocated=%d bytes", count, len(document), allocated)
		if bound := maxDuplicateKeyDiagnosticAllocationFactor * uint64(len(document)); allocated > bound {
			t.Fatalf("repeated keys=%d allocated %d bytes, above %d bytes for a %d-byte document", count, allocated, bound, len(document))
		}
		if i == 0 {
			smallest = allocated
			continue
		}
		// A cost that grows with the repetition count would multiply the
		// measured allocation by about four per doubling step. The slack
		// absorbs measurement noise on a small constant.
		if limit := 8*smallest + 1<<20; allocated > limit {
			t.Fatalf("repeated keys=%d allocated %d bytes, above %d bytes measured for %d keys", count, allocated, limit, counts[0])
		}
	}
}

// repeatedKeysThatExceedTheBound is a repetition count whose pre-FJ-065
// yaml.v3 diagnostic list is already larger than maxDuplicateKeyDiagnosticBytes,
// so a test that asserts both rejection and the bound cannot pass on the
// unbounded code.
const repeatedKeysThatExceedTheBound = 64

// TestLoadStillRejectsRepeatedKeys pins that bounding the diagnostic did not
// turn a repeated key into an accepted document on any parse path, including
// inside the intentionally open containers of §38.8.
func TestLoadStillRejectsRepeatedKeys(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"bare json stub object", repeatedKeyObject(repeatedKeysThatExceedTheBound)},
		{"json stub inside a configuration", string(repeatedKeyConfiguration(repeatedKeysThatExceedTheBound))},
		{"json duplicate inside an open raw body", `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{},"body":{"a":1,"a":2}}}}]}`},
		{"json duplicate inside raw header names", `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{"x":"1","x":"2"},"body":null}}}}]}`},
		{"block yaml mapping", string(repeatedKeyYAML(repeatedKeysThatExceedTheBound, false))},
		{"flow yaml mapping", string(repeatedKeyYAML(repeatedKeysThatExceedTheBound, true))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load([]byte(test.body))
			if err == nil {
				t.Fatalf("document with a repeated key was accepted: %s", test.body)
			}
			if len(err.Error()) > maxDuplicateKeyDiagnosticBytes {
				t.Fatalf("diagnostic is %d bytes, above the %d-byte bound: %q", len(err.Error()), maxDuplicateKeyDiagnosticBytes, err.Error())
			}
		})
	}
}

// TestLoadDuplicateKeyDiagnosticIsDeterministic runs the protocol of
// TestLoadDiagnosticsAreDeterministic over repeated-key documents and adds the
// bound, so an outcome that is identical only because it is unbounded does not
// satisfy this test.
func TestLoadDuplicateKeyDiagnosticIsDeterministic(t *testing.T) {
	const calls = 1000
	tests := []struct {
		name string
		body string
	}{
		{"json nested repeated keys", `{"schemaVersion":1,"a":{"x":1,"x":2},"b":{"y":1,"y":2}}`},
		{"json repeated id", `{"schemaVersion":1,"id":1,"id":2,"id":3}`},
		{"json sixteen repeated ids", `{"schemaVersion":1,` + strings.Trim(repeatedKeyObject(16), "{}") + `}`},
		{"yaml block repeated keys", "a: 1\na: 2\nb: 1\nb: 2\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			load := func() string {
				_, err := Load([]byte(test.body))
				if err == nil {
					return ""
				}
				return err.Error()
			}
			first := load()
			if first == "" {
				t.Fatalf("document with a repeated key was accepted: %s", test.body)
			}
			if len(first) > maxDuplicateKeyDiagnosticBytes {
				t.Fatalf("diagnostic is %d bytes, above the %d-byte bound: %q", len(first), maxDuplicateKeyDiagnosticBytes, first)
			}
			for call := 2; call <= calls; call++ {
				if got := load(); got != first {
					t.Fatalf("call %d returned %q, call 1 returned %q", call, got, first)
				}
			}
		})
	}
}

// TestLoadNamesFirstRepeatedKeyInDocumentOrder pins the rule for which key the
// bounded diagnostic names: the first key that is repeated when the document is
// walked in document order, which the JSON token scan produces directly and
// which never depends on Go's randomized map iteration order.
func TestLoadNamesFirstRepeatedKeyInDocumentOrder(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"first group repeated first", `{"schemaVersion":1,"a":{"x":1,"x":2},"y":9,"b":{"z":3,"z":4}}`, `duplicate object key "x"`},
		{"second group repeated first", `{"schemaVersion":1,"b":{"z":3,"z":4},"y":9,"a":{"x":1,"x":2}}`, `duplicate object key "z"`},
		{"outer key repeated", `{"schemaVersion":1,"k":{"v":1},"k":{"v":2}}`, `duplicate object key "k"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load([]byte(test.body))
			if err == nil {
				t.Fatalf("document with a repeated key was accepted: %s", test.body)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("diagnostic = %q, want it to contain %q", err.Error(), test.want)
			}
		})
	}
}

// TestLoadRepeatedKeyNamingIsDeterministicPerSyntax pins the two naming rules
// and shows that the difference between them is a difference between the two
// document syntaxes, not a difference between runs. The JSON scan reports the
// key whose second occurrence it reaches first, because it stops at the first
// repeat; yaml.v3 reports the pair with the earliest first occurrence. For the
// document below the two syntaxes therefore name different keys, and each
// syntax names its own key on every one of the calls.
func TestLoadRepeatedKeyNamingIsDeterministicPerSyntax(t *testing.T) {
	const calls = 200
	tests := []struct {
		name string
		body string
		want string
	}{
		{"json names the earliest second occurrence", `{"a":1,"b":1,"b":2,"a":2}`, `duplicate object key "b"`},
		{"yaml block names the earliest first occurrence", "a: 1\nb: 1\nb: 2\na: 2\n", `mapping key "a" already defined at line 1`},
		{"yaml flow names the earliest first occurrence", "{a: 1, b: 1, b: 2, a: 2}", `mapping key "a" already defined at line 1`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := ""
			for call := 1; call <= calls; call++ {
				_, err := Load([]byte(test.body))
				if err == nil {
					t.Fatalf("call %d accepted a document with a repeated key: %s", call, test.body)
				}
				message := err.Error()
				if !strings.Contains(message, test.want) {
					t.Fatalf("call %d: diagnostic = %q, want it to contain %q", call, message, test.want)
				}
				if first == "" {
					first = message
				} else if message != first {
					t.Fatalf("call %d returned %q, call 1 returned %q", call, message, first)
				}
			}
		})
	}
}

// TestLoadAcceptsDocumentsWithoutRepeatedKeys pins that the duplicate-key bound
// did not change the outcome for accepted documents or for other rejections.
func TestLoadAcceptsDocumentsWithoutRepeatedKeys(t *testing.T) {
	accepted := []struct {
		name  string
		body  string
		stubs int
	}{
		{"json defaults only", `{"schemaVersion":1}`, 0},
		{"yaml defaults only", "schemaVersion: 1\n", 0},
		{"yaml flow document", "{schemaVersion: 1}", 0},
		{"json stub", `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"answers":{}}}]}`, 1},
		{"yaml stub with open raw body", "schemaVersion: 1\nstubs:\n- id: x\n  profile: jev/v1\n  when: {}\n  then:\n    raw:\n      status: 200\n      headers: {}\n      body: {a: 1, b: 2}\n", 1},
	}
	for _, test := range accepted {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := Load([]byte(test.body))
			if err != nil {
				t.Fatalf("document without a repeated key rejected: %v", err)
			}
			if len(cfg.Stubs) != test.stubs {
				t.Fatalf("loaded %d stubs, want %d", len(cfg.Stubs), test.stubs)
			}
		})
	}
	rejected := []struct {
		name string
		body string
		want string
	}{
		{"unknown top level key", `{"schemaVersion":1,"extra":true}`, `json: unknown field "extra"`},
		{"missing schema version", "mode: strict\n", "schemaVersion is required"},
		{"trailing json document", `{"schemaVersion":1}{"schemaVersion":1}`, "decode YAML configuration"},
	}
	for _, test := range rejected {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load([]byte(test.body))
			if err == nil {
				t.Fatalf("document was accepted: %s", test.body)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("diagnostic = %q, want it to contain %q", err.Error(), test.want)
			}
		})
	}
}
