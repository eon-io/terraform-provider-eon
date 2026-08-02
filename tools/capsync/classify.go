package main

import (
	"regexp"
	"strings"
)

// Classification values used in the manifest.
const (
	ClassResource   = "resource"
	ClassDataSource = "data_source"
	ClassSkip       = "skip"
)

// Proposal is capsync's suggested triage for an operation it has not seen
// before. Proposals seed new manifest entries; humans own them from there.
type Proposal struct {
	Classification string
	Reason         string
	TerraformName  string
	NeedsReview    bool
}

var agenticPattern = regexp.MustCompile(`(?i)\b(agents?|assistants?|chats?|conversations?|mcp|copilot|llm)\b|/(agent|assistant|chat|conversation|mcp)`)

// classify applies the triage rules to one operation. The guiding heuristic:
// if a practitioner cannot express the operation as desired state that
// Terraform can reconcile and drift-detect, it does not belong in the
// provider.
//
// allOps is the full operation set for the same SDK release, used to detect
// CRUD shapes (does this path also support GET? is there a matching
// remove/cancel counterpart?).
func classify(op SpecOperation, allOps []SpecOperation) Proposal {
	pathLower := strings.ToLower(op.Path)
	last := lastSegment(op.Path)

	// The agentic surface (agents, chat, assistants, MCP and similar) is
	// excluded by policy regardless of shape.
	if agenticPattern.MatchString(op.Path) || agenticPattern.MatchString(op.ID) || agenticPattern.MatchString(op.Tag) {
		return Proposal{ClassSkip, "Agentic surface (agents/chat/assistants/MCP) is excluded from Terraform by policy.", "", false}
	}

	// Whole categories with no reconcilable desired state.
	switch op.Tag {
	case "auth":
		return Proposal{ClassSkip, "Authentication, token issuance, and credential rotation are session/security actions, not declarative infrastructure.", "", false}
	case "jobs":
		return Proposal{ClassSkip, "Job listing/status polling is imperative monitoring of runtime activity; there is no desired state to reconcile.", "", false}
	case "billing":
		return Proposal{ClassSkip, "Cost and billing queries are pure reporting.", "", false}
	case "dashboard":
		return Proposal{ClassSkip, "Dashboard/metrics endpoints are pure reporting.", "", false}
	case "databaseSnapshots":
		return Proposal{ClassSkip, "Ad-hoc query execution against snapshots (run/poll/fetch results) is an imperative job with no durable state.", "", false}
	case "actionApprovals":
		return Proposal{ClassSkip, "Approval requests are one-shot workflow actions tied to a session, not durable configuration.", "", false}
	}

	// Restore triggers: imperative by nature, but the provider already chose
	// to model restores as the eon_restore_job resource, so new restore types
	// extend that resource instead of becoming new surface area.
	if strings.Contains(op.Path, "/snapshots/{snapshotId}/") &&
		(strings.HasPrefix(last, "restore-") || strings.HasPrefix(last, "convert-")) {
		return Proposal{ClassResource, "One-shot restore trigger; restores are modeled by the existing eon_restore_job resource — extend it with this restore type rather than adding new surface.", "eon_restore_job", true}
	}

	// One-shot lifecycle actions on objects that are otherwise managed
	// declaratively. Where a transition matters for reconciliation (e.g.
	// reconnecting a disconnected account) the owning resource handles it
	// internally via plan modifiers.
	switch last {
	case "disconnect", "reconnect", "cancel", "submit", "rotate", "take-snapshot", "retry":
		return Proposal{ClassSkip, "Imperative one-shot action with no reconcilable state; lifecycle transitions belong inside the owning resource, not as standalone surface.", "", false}
	}

	// Toggle pairs (exclude/include, hold/remove-hold) describe a durable,
	// drift-detectable setting: presence of the Terraform resource means the
	// setting is applied, destroy reverts it. eon_volume_backup_exclusion is
	// the in-repo precedent.
	switch last {
	case "exclude", "include":
		return Proposal{ClassResource, "Exclude/include pair describes a durable per-object setting; model like the existing eon_volume_backup_exclusion (create = exclude, delete = include).", parentName(op) + "_backup_exclusion", false}
	case "hold", "remove-hold":
		return Proposal{ClassResource, "Hold/remove-hold pair describes a durable per-snapshot setting; model as a toggle resource (create = hold, delete = remove hold).", parentName(op) + "_hold", false}
	}

	// List operations in this API are usually POST (to carry a filter body),
	// either on ".../list" or named "list*"; they are lookups, not creates.
	isList := strings.HasSuffix(pathLower, "/list") || strings.HasPrefix(strings.ToLower(op.ID), "list")

	verbsOnPath := map[string]bool{}
	for _, other := range allOps {
		if other.Path == op.Path {
			verbsOnPath[other.Method] = true
		}
	}

	// Same-path read+write (+optional delete): a singleton config or item
	// object with full reconcile support (metrics config, connectivity
	// config, override settings).
	if !isList && verbsOnPath["GET"] && (verbsOnPath["PUT"] || verbsOnPath["PATCH"]) {
		return Proposal{ClassResource, "Read+write (+delete) on a stable path: a durable configuration object Terraform can reconcile and drift-detect.", proposeName(op, true), false}
	}

	// Write+delete without read (e.g. override endpoints): still declarative
	// state, but drift detection must come from the parent object's read.
	if !isList && !verbsOnPath["GET"] && (verbsOnPath["PUT"] || verbsOnPath["PATCH"]) && verbsOnPath["DELETE"] {
		return Proposal{ClassResource, "Set/remove pair describes a durable override setting; model as a resource whose read comes from the parent object.", proposeName(op, true) + "_override", true}
	}

	// Collection create + item get: classic CRUD resource.
	if op.Method == "POST" && !isList {
		itemPrefix := op.Path + "/"
		for _, other := range allOps {
			if other.Method == "GET" && strings.HasPrefix(other.Path, itemPrefix) && len(pathParams(other.Path)) == len(op.PathParams)+1 {
				return Proposal{ClassResource, "Create with matching item read: a durable object with stable identity and a real lifecycle.", proposeName(op, true), false}
			}
		}
	}

	// Read-only lookups: useful as data sources when their output feeds other
	// resources.
	if op.Method == "GET" || isList {
		return Proposal{ClassDataSource, "Read-only lookup whose output is useful as input to other resources.", proposeName(op, false), true}
	}

	return Proposal{ClassSkip, "NEEDS REVIEW: no triage rule matched; classify manually.", "", true}
}

// lastSegment returns the final non-parameter segment of a path.
func lastSegment(p string) string {
	segs := strings.Split(strings.Trim(p, "/"), "/")
	for i := len(segs) - 1; i >= 0; i-- {
		if !strings.HasPrefix(segs[i], "{") {
			return segs[i]
		}
	}
	return ""
}

// proposeName derives a suggested Terraform type name from the operation
// path, e.g. /v1/projects/{projectId}/source-accounts/{accountId}/metrics-config
// -> eon_source_account_metrics_config. Resources are fully singular; data
// sources keep their last word plural (eon_resource_snapshots). It is only a
// suggestion for the human reviewer.
func proposeName(op SpecOperation, singular bool) string {
	words := pathWords(op.Path)
	if len(words) == 0 {
		return ""
	}
	for i, w := range words {
		if singular || i < len(words)-1 {
			words[i] = singularize(w)
		}
	}
	return "eon_" + strings.Join(words, "_")
}

// parentName is proposeName for the path with its final action segment
// (exclude, hold, ...) removed, so both halves of a toggle pair land on the
// same proposed capability.
func parentName(op SpecOperation) string {
	words := pathWords(op.Path)
	if len(words) > 0 {
		words = words[:len(words)-1]
	}
	for i, w := range words {
		words[i] = singularize(w)
	}
	return "eon_" + strings.Join(words, "_")
}

func pathWords(path string) []string {
	var words []string
	for _, seg := range strings.Split(strings.Trim(path, "/"), "/") {
		if strings.HasPrefix(seg, "{") || seg == "v1" || seg == "projects" || seg == "list" {
			continue
		}
		words = append(words, strings.ReplaceAll(seg, "-", "_"))
	}
	return words
}

func singularize(w string) string {
	switch {
	case strings.HasSuffix(w, "ies"):
		return strings.TrimSuffix(w, "ies") + "y"
	case strings.HasSuffix(w, "ses"):
		return strings.TrimSuffix(w, "es")
	case strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss"):
		return strings.TrimSuffix(w, "s")
	}
	return w
}
