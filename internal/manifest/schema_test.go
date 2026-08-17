package manifest_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"gopkg.in/yaml.v3"
)

const fullSchemaManifest = `apiVersion: diagsift/v1alpha1
kind: DiagnosticBundle
metadata: {name: schema-test}
limits:
  maxFiles: 20
  maxTotalBytes: 1048576
  maxDuration: 30s
roots:
  - id: project
    path: .
collectors:
  - id: logs
    type: file
    root: project
    paths: [fixtures/app.log]
    archivePrefix: logs
    maxBytes: 4096
  - id: platform
    type: system
    fields: [os, arch]
  - id: version
    type: command
    executable: example-app
    args: [version]
    timeout: 2s
    maxOutputBytes: 4096
redactions:
  builtins: [credentials, private-keys, bearer-tokens, url-credentials, connection-strings, paths]
  custom:
    - id: ticket
      pattern: 'DS-[0-9]{4}'
`

func TestSchemaAcceptsRepresentativeGoValidManifests(t *testing.T) {
	schema := compileSchema(t)
	for name, input := range map[string]string{
		"minimal": valid,
		"full":    fullSchemaManifest,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := manifest.Load(writeManifest(t, input)); err != nil {
				t.Fatalf("Go validation rejected representative valid manifest: %v", err)
			}
			if err := schema.Validate(unmarshalYAML(t, input)); err != nil {
				t.Fatalf("schema rejected representative Go-valid manifest: %v", err)
			}
		})
	}
}

func TestSchemaRejectsRepresentativeGoInvalidManifests(t *testing.T) {
	schema := compileSchema(t)
	cases := map[string]string{
		"wrong api version":           strings.Replace(fullSchemaManifest, manifest.APIVersion, "diagsift/v9", 1),
		"root extra field":            strings.Replace(fullSchemaManifest, "    path: .", "    path: .\n    extra: true", 1),
		"file mixed field":            strings.Replace(fullSchemaManifest, "    maxBytes: 4096", "    maxBytes: 4096\n    fields: [os]", 1),
		"hostname field":              strings.Replace(fullSchemaManifest, "fields: [os, arch]", "fields: [os, hostname]", 1),
		"shell executable":            strings.Replace(fullSchemaManifest, "executable: example-app", "executable: C:\\\\Windows\\\\System32\\\\CMD.EXE", 1),
		"batch executable":            strings.Replace(fullSchemaManifest, "executable: example-app", "executable: C:\\\\tools\\\\collect.bat", 1),
		"command script executable":   strings.Replace(fullSchemaManifest, "executable: example-app", "executable: diag.CMD", 1),
		"limit above ceiling":         strings.Replace(fullSchemaManifest, "maxFiles: 20", "maxFiles: 1001", 1),
		"file bytes above ceiling":    strings.Replace(fullSchemaManifest, "maxBytes: 4096", "maxBytes: 8388609", 1),
		"command bytes above ceiling": strings.Replace(fullSchemaManifest, "maxOutputBytes: 4096", "maxOutputBytes: 4194305", 1),
		"unknown redaction":           strings.Replace(fullSchemaManifest, "credentials,", "unknown,", 1),
		"custom regex too long":       strings.Replace(fullSchemaManifest, "'DS-[0-9]{4}'", "'"+strings.Repeat("x", 513)+"'", 1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := manifest.Load(writeManifest(t, input)); err == nil {
				t.Fatal("Go validation unexpectedly accepted representative invalid manifest")
			}
			if err := schema.Validate(unmarshalYAML(t, input)); err == nil {
				t.Fatal("schema unexpectedly accepted representative Go-invalid manifest")
			}
		})
	}
}

func compileSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate schema test source")
	}
	path := filepath.Join(filepath.Dir(source), "..", "..", "schemas", "diagsift-v1alpha1.schema.json")
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	schema, err := compiler.Compile(path)
	if err != nil {
		t.Fatalf("compile JSON Schema: %v", err)
	}
	return schema
}

func unmarshalYAML(t *testing.T, input string) any {
	t.Helper()
	var value any
	if err := yaml.Unmarshal([]byte(input), &value); err != nil {
		t.Fatalf("decode YAML for schema validation: %v", err)
	}
	return value
}

func TestSchemaFileHasExpectedIdentity(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(source), "..", "..", "schemas", "diagsift-v1alpha1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"$id": "urn:diagsift:schema:v1alpha1"`) {
		t.Fatal("schema identity is not the public URN")
	}
}
