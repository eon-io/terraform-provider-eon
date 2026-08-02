package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const sdkModule = "github.com/eon-io/eon-sdk-go"

// SpecOperation is one operation extracted from the SDK's OpenAPI spec.
type SpecOperation struct {
	ID            string   `json:"operation_id"`
	Method        string   `json:"method"`
	Path          string   `json:"path"`
	Tag           string   `json:"tag"`
	Summary       string   `json:"summary"`
	RequestModel  string   `json:"request_model,omitempty"`
	ResponseModel string   `json:"response_model,omitempty"`
	PathParams    []string `json:"path_params,omitempty"`
}

// SDKService returns the generated Go API service name for the operation's
// tag, e.g. tag "backupPolicies" -> "BackupPoliciesAPI".
func (o SpecOperation) SDKService() string {
	if o.Tag == "" {
		return ""
	}
	return strings.ToUpper(o.Tag[:1]) + o.Tag[1:] + "API"
}

// SDKMethod returns the generated Go method name for the operation, which is
// the operationId with the first letter upper-cased.
func (o SpecOperation) SDKMethod() string {
	if o.ID == "" {
		return ""
	}
	return strings.ToUpper(o.ID[:1]) + o.ID[1:]
}

// loadSDKSpec resolves an SDK version (go.mod version when empty, or a query
// like "latest") to a concrete release, downloads the module through the
// standard module cache, and parses the api/openapi.yaml it ships.
func loadSDKSpec(providerDir, version string) (string, []SpecOperation, error) {
	if version == "" {
		out, err := goCmd(providerDir, "list", "-m", "-f", "{{.Version}}", sdkModule)
		if err != nil {
			return "", nil, fmt.Errorf("resolving SDK version from go.mod: %w", err)
		}
		version = strings.TrimSpace(out)
	}

	out, err := goCmd(providerDir, "mod", "download", "-json", sdkModule+"@"+version)
	if err != nil {
		return "", nil, fmt.Errorf("downloading %s@%s: %w", sdkModule, version, err)
	}
	var dl struct {
		Version string
		Dir     string
	}
	if err := json.Unmarshal([]byte(out), &dl); err != nil {
		return "", nil, fmt.Errorf("parsing go mod download output: %w", err)
	}

	ops, err := parseSpecFile(dl.Dir + "/api/openapi.yaml")
	if err != nil {
		return "", nil, err
	}
	return dl.Version, ops, nil
}

func goCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// parseSpecFile extracts all operations from an OpenAPI 3 YAML document.
func parseSpecFile(path string) ([]SpecOperation, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var spec struct {
		Paths map[string]map[string]struct {
			OperationID string   `yaml:"operationId"`
			Tags        []string `yaml:"tags"`
			Summary     string   `yaml:"summary"`
			RequestBody struct {
				Content map[string]struct {
					Schema struct {
						Ref string `yaml:"$ref"`
					} `yaml:"schema"`
				} `yaml:"content"`
			} `yaml:"requestBody"`
			Responses map[string]struct {
				Content map[string]struct {
					Schema struct {
						Ref string `yaml:"$ref"`
					} `yaml:"schema"`
				} `yaml:"content"`
			} `yaml:"responses"`
		} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	httpVerbs := map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}
	var ops []SpecOperation
	for p, item := range spec.Paths {
		for verb, op := range item {
			if !httpVerbs[verb] || op.OperationID == "" {
				continue
			}
			o := SpecOperation{
				ID:         op.OperationID,
				Method:     strings.ToUpper(verb),
				Path:       p,
				Summary:    op.Summary,
				PathParams: pathParams(p),
			}
			if len(op.Tags) > 0 {
				o.Tag = op.Tags[0]
			}
			for _, c := range op.RequestBody.Content {
				o.RequestModel = refName(c.Schema.Ref)
				break
			}
			for _, code := range []string{"200", "201", "202"} {
				if r, ok := op.Responses[code]; ok {
					for _, c := range r.Content {
						o.ResponseModel = refName(c.Schema.Ref)
						break
					}
					break
				}
			}
			ops = append(ops, o)
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path != ops[j].Path {
			return ops[i].Path < ops[j].Path
		}
		return ops[i].Method < ops[j].Method
	})
	return ops, nil
}

func refName(ref string) string {
	if ref == "" {
		return ""
	}
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

func pathParams(p string) []string {
	var params []string
	for _, seg := range strings.Split(p, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			params = append(params, strings.Trim(seg, "{}"))
		}
	}
	return params
}
