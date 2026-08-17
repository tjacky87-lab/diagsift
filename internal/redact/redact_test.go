package redact_test

import (
	"bytes"
	"testing"

	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/redact"
)

func TestSyntheticCanaryClasses(t *testing.T) {
	config := manifest.Redactions{Builtins: []string{
		"private-keys", "bearer-tokens", "credentials", "url-credentials", "connection-strings", "paths",
	}}
	r, err := redact.New(config)
	if err != nil {
		t.Fatal(err)
	}
	canaries := []string{
		"-----BEGIN PRIVATE KEY-----\nINVALID-SYNTHETIC-KEY-DATA\n-----END PRIVATE KEY-----",
		"Bearer invalid.synthetic.token.value",
		"password=INVALID_SYNTHETIC_PASSWORD",
		"https://synthetic-user:invalid-password@example.invalid/path",
		"Server=example.invalid;User Id=synthetic;Password=invalid-password;",
		`C:\Users\SyntheticPerson\project`,
		"/home/synthetic-person/project",
	}
	for _, canary := range canaries {
		output, counts, err := r.Sanitize([]byte(canary))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(output, []byte(canary)) || len(counts) == 0 {
			t.Fatalf("canary survived or was not counted: output=%q counts=%v", output, counts)
		}
	}
}

func TestChunkBoundaryRedaction(t *testing.T) {
	r, err := redact.New(manifest.Redactions{Builtins: []string{"bearer-tokens"}})
	if err != nil {
		t.Fatal(err)
	}
	stream := r.Stream(1024)
	for _, chunk := range []string{"prefix Bear", "er invalid.", "synthetic.", "token.value suffix"} {
		if _, err := stream.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	output, counts, truncated, err := stream.Finish()
	if err != nil || truncated {
		t.Fatalf("err=%v truncated=%v", err, truncated)
	}
	if bytes.Contains(output, []byte("invalid.synthetic.token.value")) || counts["bearer-tokens"] != 1 {
		t.Fatalf("output=%q counts=%v", output, counts)
	}
}

func TestCustomRuleUsesOnlyIDInReplacement(t *testing.T) {
	r, err := redact.New(manifest.Redactions{Custom: []manifest.CustomRule{{ID: "ticket", Pattern: `DS-[0-9]{4}`}}})
	if err != nil {
		t.Fatal(err)
	}
	output, counts, err := r.Sanitize([]byte("ticket DS-0000"))
	if err != nil || string(output) != "ticket [REDACTED:ticket]" || counts["ticket"] != 1 {
		t.Fatalf("output=%q counts=%v err=%v", output, counts, err)
	}
}
