package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// protoV6ProviderFactories serves the real provider (real client, real HTTP)
// in-process for acceptance tests driven by the terraform CLI.
var protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"eon": providerserver.NewProtocol6WithError(New("test", client.NewClient)()),
}

// fakeEonServer is an in-memory stand-in for the Eon API, speaking the exact
// JSON the SDK models marshal to. Acceptance tests point the provider at it
// via accProviderConfig, so terraform plan/apply exercise the full stack —
// schema, CRUD logic, and the real internal/client HTTP layer — with no
// credentials and no live backend.
type fakeEonServer struct {
	*httptest.Server

	mu              sync.Mutex
	nextID          int
	postureControls map[string]externalEonSdkAPI.BackupPostureControl
}

func newFakeEonServer(t *testing.T) *fakeEonServer {
	t.Helper()

	f := &fakeEonServer{postureControls: map[string]externalEonSdkAPI.BackupPostureControl{}}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/projects/{projectId}/backup-posture-controls", func(w http.ResponseWriter, r *http.Request) {
		var req externalEonSdkAPI.CreateBackupPostureControlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.nextID++
		control := externalEonSdkAPI.BackupPostureControl{
			Id:               fmt.Sprintf("pc-%d", f.nextID),
			Name:             req.Name,
			Severity:         req.Severity,
			ResourceSelector: req.ResourceSelector,
			Rules:            req.Rules,
		}
		f.postureControls[control.Id] = control
		writeJSON(w, http.StatusCreated, control)
	})

	mux.HandleFunc("GET /v1/projects/{projectId}/backup-posture-controls/{controlId}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		control, ok := f.postureControls[r.PathValue("controlId")]
		if !ok {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, control)
	})

	mux.HandleFunc("PUT /v1/projects/{projectId}/backup-posture-controls/{controlId}", func(w http.ResponseWriter, r *http.Request) {
		var req externalEonSdkAPI.UpdateBackupPostureControlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		control, ok := f.postureControls[r.PathValue("controlId")]
		if !ok {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		control.Name = req.Name
		control.Severity = req.Severity
		control.ResourceSelector = req.ResourceSelector
		control.Rules = req.Rules
		f.postureControls[control.Id] = control
		writeJSON(w, http.StatusOK, control)
	})

	mux.HandleFunc("DELETE /v1/projects/{projectId}/backup-posture-controls/{controlId}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if _, ok := f.postureControls[r.PathValue("controlId")]; !ok {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		delete(f.postureControls, r.PathValue("controlId"))
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("POST /v1/projects/{projectId}/backup-posture-controls/list", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		controls := make([]externalEonSdkAPI.BackupPostureControl, 0, len(f.postureControls))
		for _, c := range f.postureControls {
			controls = append(controls, c)
		}
		writeJSON(w, http.StatusOK, externalEonSdkAPI.ListBackupPostureControlsResponse{BackupPostureControls: controls})
	})

	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Server.Close)
	return f
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// accProviderConfig returns a provider block wired to the fake server. The
// Authorization default header short-circuits OAuth in the real client, and
// EON_USE_EXACT_ENDPOINT (set by the caller via t.Setenv) stops the provider
// from appending /api to the endpoint.
func accProviderConfig(t *testing.T, f *fakeEonServer) string {
	t.Helper()
	t.Setenv("EON_USE_EXACT_ENDPOINT", "true")
	t.Setenv("EON_DEFAULT_HEADERS", "Authorization:Bearer acceptance-test-token")
	return fmt.Sprintf(`
provider "eon" {
  endpoint      = %q
  client_id     = "acceptance-test"
  client_secret = "acceptance-test"
  project_id    = "proj-acceptance"
}
`, f.URL)
}
