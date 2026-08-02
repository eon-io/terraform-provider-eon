package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manifest is the committed record of every SDK operation, its triage
// classification, and its coverage status. It lives at
// capabilities/manifest.yaml and is the pipeline's decision store:
//
//   - classification, reason, terraform_name, notes and needs_review are
//     HUMAN-OWNED. capsync writes them once, when it first sees an operation,
//     and never again. Reviewers override a decision by editing these fields;
//     subsequent runs respect the edit, so skipped items are never re-proposed.
//   - method, path, status, covered_by and first_seen are FACTS. capsync
//     recomputes them on every -update-manifest run.
type Manifest struct {
	SDKModule  string                    `yaml:"sdk_module"`
	SDKVersion string                    `yaml:"sdk_version"`
	Operations map[string]*ManifestEntry `yaml:"operations"`
}

// ManifestEntry is the manifest record for one SDK operation.
type ManifestEntry struct {
	Method         string   `yaml:"method"`
	Path           string   `yaml:"path"`
	Classification string   `yaml:"classification"`
	Reason         string   `yaml:"reason"`
	TerraformName  string   `yaml:"terraform_name,omitempty"`
	Notes          string   `yaml:"notes,omitempty"`
	NeedsReview    bool     `yaml:"needs_review,omitempty"`
	Status         string   `yaml:"status"`
	CoveredBy      []string `yaml:"covered_by,omitempty"`
	FirstSeen      string   `yaml:"first_seen,omitempty"`
}

func loadManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- path comes from a CLI flag of this dev tool
	if os.IsNotExist(err) {
		return &Manifest{SDKModule: sdkModule, Operations: map[string]*ManifestEntry{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if m.Operations == nil {
		m.Operations = map[string]*ManifestEntry{}
	}
	if m.SDKModule == "" {
		m.SDKModule = sdkModule
	}
	return &m, nil
}

const manifestHeader = `# Capability manifest: every Eon SDK operation, its Terraform triage
# classification, and its provider coverage status.
#
# Generated and updated by 'make gap-report' (tools/capsync). Field ownership:
#
#   classification, reason, terraform_name, notes, needs_review
#       Human-owned triage decisions. capsync seeds them for newly discovered
#       operations and NEVER overwrites them afterwards. To override a
#       decision, edit the field and commit; later runs respect the override,
#       and operations classified 'skip' are never re-proposed.
#
#   method, path, status, covered_by, first_seen
#       Facts recomputed by capsync on every run. Do not edit by hand.
#
# classification: resource | data_source | skip
# status:         covered | covered_internal | gap | skipped
`

// saveManifest writes the manifest back, folding in what the report learned:
// new operations are appended with their proposed triage, and the factual
// fields of existing entries are refreshed. Human-owned fields of existing
// entries are never touched.
func saveManifest(path string, m *Manifest, r *Report) error {
	m.SDKVersion = r.SDKVersion
	for i := range r.Operations {
		op := &r.Operations[i]
		entry, exists := m.Operations[op.ID]
		if !exists {
			entry = &ManifestEntry{
				Classification: op.Classification,
				Reason:         op.Reason,
				TerraformName:  op.TerraformName,
				NeedsReview:    op.NeedsReview,
				FirstSeen:      r.SDKVersion,
			}
			m.Operations[op.ID] = entry
		}
		entry.Method = op.Method
		entry.Path = op.Path
		entry.Status = op.Status
		entry.CoveredBy = op.CoveredBy
		if entry.FirstSeen == "" {
			entry.FirstSeen = r.SDKVersion
		}
	}

	out, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, append([]byte(manifestHeader), out...), 0o600)
}
