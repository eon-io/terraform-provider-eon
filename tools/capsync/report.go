package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Operation status values.
const (
	StatusCovered         = "covered"          // consumed by at least one resource or data source
	StatusCoveredInternal = "covered_internal" // consumed only by internal plumbing (auth, plan modifiers, shared helpers)
	StatusGap             = "gap"              // classified resource/data_source but not consumed
	StatusSkipped         = "skipped"          // classified skip and not consumed
)

// Report is the full gap analysis for one SDK release.
type Report struct {
	SDKModule  string            `json:"sdk_module"`
	SDKVersion string            `json:"sdk_version"`
	Stats      Stats             `json:"stats"`
	Operations []ReportOperation `json:"operations"`
	Gaps       []GapGroup        `json:"gaps"`
	Removed    []string          `json:"removed_operations,omitempty"`
}

// Stats summarizes the report.
type Stats struct {
	Total           int `json:"total"`
	Covered         int `json:"covered"`
	CoveredInternal int `json:"covered_internal"`
	Gaps            int `json:"gaps"`
	Skipped         int `json:"skipped"`
	NeedsReview     int `json:"needs_review"`
	NewOperations   int `json:"new_operations"`
}

// ReportOperation is one operation with its spec shape, triage decision, and
// coverage status.
type ReportOperation struct {
	SpecOperation
	Classification string   `json:"classification"`
	Reason         string   `json:"reason"`
	TerraformName  string   `json:"terraform_name,omitempty"`
	Notes          string   `json:"notes,omitempty"`
	Status         string   `json:"status"`
	CoveredBy      []string `json:"covered_by,omitempty"`
	New            bool     `json:"new,omitempty"`
	NeedsReview    bool     `json:"needs_review,omitempty"`
}

// GapGroup bundles the gap operations that belong to one proposed capability
// (one future resource or data source, i.e. one future PR).
type GapGroup struct {
	TerraformName  string   `json:"terraform_name"`
	Classification string   `json:"classification"`
	OperationIDs   []string `json:"operation_ids"`
}

// buildReport joins the spec, the coverage scan, and the manifest into the
// gap report. Manifest entries win over fresh classification proposals; the
// manifest is not modified here (see saveManifest).
func buildReport(sdkVersion string, ops []SpecOperation, coverage Coverage, manifest *Manifest) *Report {
	r := &Report{SDKModule: sdkModule, SDKVersion: sdkVersion}

	seen := map[string]bool{}
	for _, op := range ops {
		seen[op.ID] = true
		ro := ReportOperation{SpecOperation: op, CoveredBy: coverage.Consumers[op.ID]}

		if entry, ok := manifest.Operations[op.ID]; ok {
			ro.Classification = entry.Classification
			ro.Reason = entry.Reason
			ro.TerraformName = entry.TerraformName
			ro.Notes = entry.Notes
			ro.NeedsReview = entry.NeedsReview
		} else {
			proposal := classify(op, ops)
			ro.Classification = proposal.Classification
			ro.Reason = proposal.Reason
			ro.TerraformName = proposal.TerraformName
			ro.NeedsReview = proposal.NeedsReview
			ro.New = true
			r.Stats.NewOperations++
		}

		switch {
		case len(coverage.TerraformConsumers(op.ID)) > 0:
			ro.Status = StatusCovered
			r.Stats.Covered++
		case len(ro.CoveredBy) > 0:
			ro.Status = StatusCoveredInternal
			r.Stats.CoveredInternal++
		case ro.Classification == ClassSkip:
			ro.Status = StatusSkipped
			r.Stats.Skipped++
		default:
			ro.Status = StatusGap
			r.Stats.Gaps++
		}
		if ro.NeedsReview {
			r.Stats.NeedsReview++
		}
		r.Operations = append(r.Operations, ro)
	}
	r.Stats.Total = len(r.Operations)

	// Manifest entries whose operation no longer exists in this SDK release.
	for id := range manifest.Operations {
		if !seen[id] {
			r.Removed = append(r.Removed, id)
		}
	}
	sort.Strings(r.Removed)

	// Group gaps by proposed capability: one group = one future PR.
	groups := map[string]*GapGroup{}
	for _, op := range r.Operations {
		if op.Status != StatusGap {
			continue
		}
		name := op.TerraformName
		if name == "" {
			name = op.ID
		}
		g, ok := groups[name]
		if !ok {
			g = &GapGroup{TerraformName: name, Classification: op.Classification}
			groups[name] = g
		}
		g.OperationIDs = append(g.OperationIDs, op.ID)
	}
	for _, g := range groups {
		sort.Strings(g.OperationIDs)
		r.Gaps = append(r.Gaps, *g)
	}
	sort.Slice(r.Gaps, func(i, j int) bool { return r.Gaps[i].TerraformName < r.Gaps[j].TerraformName })

	return r
}

func writeJSONReport(path string, r *Report) error {
	out, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

func writeMarkdownReport(path string, r *Report) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Eon SDK / Terraform provider capability gap report\n\n")
	fmt.Fprintf(&b, "SDK release: `%s@%s`\n\n", r.SDKModule, r.SDKVersion)
	fmt.Fprintf(&b, "| Total | Covered | Covered (internal) | Gaps | Skipped | Needs review |\n")
	fmt.Fprintf(&b, "|---|---|---|---|---|---|\n")
	fmt.Fprintf(&b, "| %d | %d | %d | %d | %d | %d |\n\n",
		r.Stats.Total, r.Stats.Covered, r.Stats.CoveredInternal, r.Stats.Gaps, r.Stats.Skipped, r.Stats.NeedsReview)

	byStatus := func(status string) []ReportOperation {
		var out []ReportOperation
		for _, op := range r.Operations {
			if op.Status == status {
				out = append(out, op)
			}
		}
		return out
	}

	if len(r.Gaps) > 0 {
		fmt.Fprintf(&b, "## Gaps: proposed new provider surface\n\n")
		fmt.Fprintf(&b, "One group per capability; each group is expected to become one PR.\n\n")
		opByID := map[string]ReportOperation{}
		for _, op := range r.Operations {
			opByID[op.ID] = op
		}
		for _, g := range r.Gaps {
			fmt.Fprintf(&b, "### `%s` (%s)\n\n", g.TerraformName, g.Classification)
			fmt.Fprintf(&b, "| Operation | Endpoint | Reasoning |\n|---|---|---|\n")
			for _, id := range g.OperationIDs {
				op := opByID[id]
				flag := ""
				if op.NeedsReview {
					flag = " ⚠️"
				}
				fmt.Fprintf(&b, "| `%s`%s | `%s %s` | %s |\n", op.ID, flag, op.Method, op.Path, cell(op.Reason))
			}
			b.WriteString("\n")
		}
	}

	if ops := byStatus(StatusSkipped); len(ops) > 0 {
		fmt.Fprintf(&b, "## Skipped operations\n\n")
		fmt.Fprintf(&b, "Not exposed in Terraform; edit `capabilities/manifest.yaml` to override.\n\n")
		fmt.Fprintf(&b, "| Operation | Endpoint | Reason |\n|---|---|---|\n")
		for _, op := range ops {
			fmt.Fprintf(&b, "| `%s` | `%s %s` | %s |\n", op.ID, op.Method, op.Path, cell(op.Reason))
		}
		b.WriteString("\n")
	}

	if ops := byStatus(StatusCovered); len(ops) > 0 {
		fmt.Fprintf(&b, "## Covered operations\n\n")
		fmt.Fprintf(&b, "| Operation | Endpoint | Consumed by |\n|---|---|---|\n")
		for _, op := range ops {
			fmt.Fprintf(&b, "| `%s` | `%s %s` | %s |\n", op.ID, op.Method, op.Path, cell(strings.Join(op.CoveredBy, ", ")))
		}
		b.WriteString("\n")
	}

	if ops := byStatus(StatusCoveredInternal); len(ops) > 0 {
		fmt.Fprintf(&b, "## Covered internally\n\n")
		fmt.Fprintf(&b, "Consumed by provider plumbing rather than a specific resource or data source.\n\n")
		fmt.Fprintf(&b, "| Operation | Endpoint | Consumed by | Reason |\n|---|---|---|---|\n")
		for _, op := range ops {
			fmt.Fprintf(&b, "| `%s` | `%s %s` | %s | %s |\n", op.ID, op.Method, op.Path, cell(strings.Join(op.CoveredBy, ", ")), cell(op.Reason))
		}
		b.WriteString("\n")
	}

	if len(r.Removed) > 0 {
		fmt.Fprintf(&b, "## Removed from SDK\n\n")
		fmt.Fprintf(&b, "Present in the manifest but absent from this SDK release:\n\n")
		for _, id := range r.Removed {
			fmt.Fprintf(&b, "- `%s`\n", id)
		}
		b.WriteString("\n")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// cell escapes a string for use inside a markdown table cell.
func cell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " ")
}
