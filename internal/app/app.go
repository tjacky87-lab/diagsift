// Package app implements the DiagSift command-line interface.
package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/tjacky87-lab/diagsift/internal/bundle"
	inspectbundle "github.com/tjacky87-lab/diagsift/internal/inspect"
	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/plan"
	"github.com/tjacky87-lab/diagsift/internal/redact"
)

var Version = "0.1.0-dev"

const (
	ExitOK         = 0
	ExitUsage      = 2
	ExitValidation = 3
	ExitConsent    = 4
	ExitCollect    = 5
	ExitInspect    = 6
)

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return ExitUsage
	}
	switch args[0] {
	case "version":
		if len(args) != 1 {
			return usageError(stderr, "version accepts no arguments")
		}
		_, _ = fmt.Fprintf(stdout, "diagsift %s\n", Version)
		return ExitOK
	case "validate", "plan":
		if len(args) > 2 {
			return usageError(stderr, "%s accepts at most one manifest path", args[0])
		}
		path := "diagsift.yaml"
		if len(args) == 2 {
			path = args[1]
		}
		loaded, err := manifest.Load(path)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "validation failed: %v\n", err)
			return ExitValidation
		}
		preview, err := plan.Build(loaded)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "validation failed: %v\n", err)
			return ExitValidation
		}
		if args[0] == "validate" {
			_, _ = fmt.Fprintf(stdout, "valid: %s (%s)\n", preview.Name, filepath.Base(preview.ManifestPath))
			return ExitOK
		}
		data, err := preview.JSON()
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "plan rendering failed: %v\n", err)
			return ExitValidation
		}
		_, _ = stdout.Write(data)
		return ExitOK
	case "collect":
		return runCollect(args[1:], stdin, stdout, stderr)
	case "inspect":
		if len(args) != 2 {
			return usageError(stderr, "inspect requires exactly one bundle path")
		}
		summary, err := inspectbundle.Open(args[1])
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "inspection rejected bundle: %v\n", err)
			return ExitInspect
		}
		data, err := summary.JSON()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "inspection failed: summary could not be rendered")
			return ExitInspect
		}
		_, _ = stdout.Write(data)
		return ExitOK
	case "help", "-h", "--help":
		usage(stdout)
		return ExitOK
	default:
		return usageError(stderr, "unknown command %q", args[0])
	}
}

func runCollect(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	manifestPath, output, yes, err := parseCollectArgs(args)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	loaded, err := manifest.Load(manifestPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "validation failed: %v\n", err)
		return ExitValidation
	}
	preview, err := plan.Build(loaded)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "validation failed: %v\n", err)
		return ExitValidation
	}
	redactor, err := redact.New(loaded.Manifest.Redactions)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "validation failed: redaction configuration is invalid")
		return ExitValidation
	}
	if !yes {
		data, err := preview.JSON()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "collection plan could not be rendered")
			return ExitCollect
		}
		_, _ = stdout.Write(data)
		_, _ = fmt.Fprint(stdout, "Type YES to collect this plan locally: ")
		scanner := bufio.NewScanner(stdin)
		if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "YES" {
			_, _ = fmt.Fprintln(stderr, "collection cancelled: explicit consent was not received")
			return ExitConsent
		}
	}
	result, err := bundle.Create(context.Background(), loaded, preview, output, Version, redactor)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "collection failed: %v\n", err)
		return ExitCollect
	}
	_, _ = fmt.Fprintf(stdout, "created %s: %d collector entries, %d errors, partial=%t\n",
		filepath.Base(output), result.Entries, result.Errors, result.Partial)
	return ExitOK
}

func parseCollectArgs(args []string) (manifestPath, output string, yes bool, err error) {
	manifestPath = "diagsift.yaml"
	positionals := []string{}
	for index := 0; index < len(args); index++ {
		switch {
		case args[index] == "--yes":
			yes = true
		case args[index] == "--output":
			index++
			if index >= len(args) || args[index] == "" {
				return "", "", false, fmt.Errorf("--output requires a value")
			}
			output = args[index]
		case strings.HasPrefix(args[index], "--output="):
			output = strings.TrimPrefix(args[index], "--output=")
			if output == "" {
				return "", "", false, fmt.Errorf("--output requires a value")
			}
		case strings.HasPrefix(args[index], "-"):
			return "", "", false, fmt.Errorf("unknown collect option %q", args[index])
		default:
			positionals = append(positionals, args[index])
		}
	}
	if len(positionals) > 1 {
		return "", "", false, fmt.Errorf("collect accepts at most one manifest path")
	}
	if len(positionals) == 1 {
		manifestPath = positionals[0]
	}
	if output == "" {
		return "", "", false, fmt.Errorf("collect requires --output")
	}
	return manifestPath, output, yes, nil
}

func usageError(stderr io.Writer, format string, args ...any) int {
	_, _ = fmt.Fprintf(stderr, "error: "+format+"\n\n", args...)
	usage(stderr)
	return ExitUsage
}

func usage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `DiagSift collects bounded, local diagnostic bundles for review.

Usage:
  diagsift validate [manifest]
  diagsift plan [manifest]
  diagsift collect [manifest] --output <bundle.zip> [--yes]
  diagsift inspect <bundle.zip>
  diagsift version`)
}
