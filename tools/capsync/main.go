// Command capsync analyzes the capability gap between the Eon Go SDK and this
// Terraform provider.
//
// It extracts every operation exposed by a given SDK release (from the
// api/openapi.yaml shipped inside the SDK module), extracts the provider's
// current SDK coverage by static analysis of internal/client and
// internal/provider, and diffs the two against the committed capability
// manifest (capabilities/manifest.yaml).
//
// Classification decisions (resource / data_source / skip and their reasons)
// live in the manifest and are owned by humans: capsync proposes entries for
// operations it has never seen, but never rewrites an existing entry's
// classification or reason. Coverage status and covered_by are facts and are
// recomputed on every run.
//
// Usage:
//
//	go run ./tools/capsync                      # report against go.mod SDK version
//	go run ./tools/capsync -sdk-version latest  # report against newest SDK release
//	go run ./tools/capsync -update-manifest     # also append newly seen operations
//	go run ./tools/capsync -check               # exit 1 if there are unclassified ops
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	providerDir := flag.String("provider-dir", ".", "Path to the provider repository root.")
	sdkVersion := flag.String("sdk-version", "", "SDK version to analyze (e.g. v1.173.0 or 'latest'). Defaults to the version in go.mod.")
	specPath := flag.String("spec", "", "Path to a local openapi.yaml. Overrides -sdk-version resolution.")
	manifestPath := flag.String("manifest", "capabilities/manifest.yaml", "Path to the capability manifest, relative to -provider-dir.")
	jsonOut := flag.String("json-out", "capabilities/gap-report.json", "Where to write the machine-readable gap report, relative to -provider-dir. Empty to skip.")
	mdOut := flag.String("md-out", "capabilities/gap-report.md", "Where to write the human-readable gap report, relative to -provider-dir. Empty to skip.")
	updateManifest := flag.Bool("update-manifest", false, "Write newly discovered operations and recomputed coverage status back to the manifest.")
	check := flag.Bool("check", false, "Exit non-zero if any operation is missing from the manifest or needs review.")
	flag.Parse()

	if err := run(*providerDir, *sdkVersion, *specPath, *manifestPath, *jsonOut, *mdOut, *updateManifest, *check); err != nil {
		fmt.Fprintf(os.Stderr, "capsync: %v\n", err)
		os.Exit(1)
	}
}

func run(providerDir, sdkVersion, specPath, manifestPath, jsonOut, mdOut string, updateManifest, check bool) error {
	resolvedVersion := sdkVersion
	var ops []SpecOperation
	var err error
	if specPath != "" {
		ops, err = parseSpecFile(specPath)
		if resolvedVersion == "" {
			resolvedVersion = "local"
		}
	} else {
		resolvedVersion, ops, err = loadSDKSpec(providerDir, sdkVersion)
	}
	if err != nil {
		return fmt.Errorf("loading SDK spec: %w", err)
	}

	coverage, err := extractCoverage(providerDir, ops)
	if err != nil {
		return fmt.Errorf("extracting provider coverage: %w", err)
	}

	manifest, err := loadManifest(join(providerDir, manifestPath))
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	report := buildReport(resolvedVersion, ops, coverage, manifest)

	if updateManifest {
		if err := saveManifest(join(providerDir, manifestPath), manifest, report); err != nil {
			return fmt.Errorf("saving manifest: %w", err)
		}
	}
	if jsonOut != "" {
		if err := writeJSONReport(join(providerDir, jsonOut), report); err != nil {
			return fmt.Errorf("writing JSON report: %w", err)
		}
	}
	if mdOut != "" {
		if err := writeMarkdownReport(join(providerDir, mdOut), report); err != nil {
			return fmt.Errorf("writing markdown report: %w", err)
		}
	}

	fmt.Printf("SDK %s: %d operations — %d covered, %d covered (internal), %d gaps, %d skipped, %d need review\n",
		report.SDKVersion, len(report.Operations),
		report.Stats.Covered, report.Stats.CoveredInternal, report.Stats.Gaps, report.Stats.Skipped, report.Stats.NeedsReview)

	if check && (report.Stats.NeedsReview > 0 || report.Stats.NewOperations > 0) {
		return fmt.Errorf("%d operation(s) are new or need review; run 'make gap-report' and classify them in the manifest", report.Stats.NeedsReview+report.Stats.NewOperations)
	}
	return nil
}

// join joins a repo-root-relative path onto the provider dir, leaving
// absolute paths untouched.
func join(providerDir, p string) string {
	if p == "" || p[0] == '/' {
		return p
	}
	return providerDir + "/" + p
}
