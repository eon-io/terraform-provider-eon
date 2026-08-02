package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	testAccProjectID    = "test-project"
	testAccClientID     = "test-client-id"
	testAccClientSecret = "test-client-secret"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server that Terraform can connect to.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"eon": providerserver.NewProtocol6WithError(New("test", client.NewClient)()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
}

// fakeEonServer is an in-memory Eon API used by acceptance tests.
type fakeEonServer struct {
	server    *httptest.Server
	mu        sync.Mutex
	projectID string

	controls map[string]*externalEonSdkAPI.BackupPostureControl
	idps     map[string]*externalEonSdkAPI.Idp
	nextID   int
}

func newFakeEonServer(t *testing.T) *fakeEonServer {
	t.Helper()
	f := &fakeEonServer{
		projectID: testAccProjectID,
		controls:  make(map[string]*externalEonSdkAPI.BackupPostureControl),
		idps:      make(map[string]*externalEonSdkAPI.Idp),
		nextID:    1,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", f.handle)
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeEonServer) URL() string {
	return f.server.URL
}

func (f *fakeEonServer) providerConfig() string {
	return fmt.Sprintf(`
provider "eon" {
  endpoint      = %q
  client_id     = %q
  client_secret = %q
  project_id    = %q
}
`, f.URL(), testAccClientID, testAccClientSecret, f.projectID)
}

func (f *fakeEonServer) DeleteControl(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.controls, id)
}

func (f *fakeEonServer) AddIdp(idp *externalEonSdkAPI.Idp) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.idps[idp.Id] = idp
}

func (f *fakeEonServer) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case r.Method == http.MethodPost && path == "/v1/token":
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"accessToken":       "test-token",
			"expirationSeconds": 43200,
		})
		return
	case r.Method == http.MethodPost && path == "/v1/oauth2/token":
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"access_token": "test-token",
			"expires_in":   43200,
			"token_type":   "Bearer",
		})
		return
	}

	prefix := fmt.Sprintf("/v1/projects/%s/backup-posture-controls", f.projectID)
	switch {
	case r.Method == http.MethodPost && path == prefix:
		f.handleCreateControl(w, r)
		return
	case r.Method == http.MethodPost && path == prefix+"/list":
		f.handleListControls(w, r)
		return
	case strings.HasPrefix(path, prefix+"/"):
		id := strings.TrimPrefix(path, prefix+"/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			f.handleGetControl(w, id)
		case http.MethodPut:
			f.handleUpdateControl(w, r, id)
		case http.MethodDelete:
			f.handleDeleteControl(w, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	case r.Method == http.MethodPost && path == "/v1/idps/list":
		f.handleListIdps(w, r)
		return
	}

	http.NotFound(w, r)
}

func (f *fakeEonServer) handleCreateControl(w http.ResponseWriter, r *http.Request) {
	var req externalEonSdkAPI.CreateBackupPostureControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	id := fmt.Sprintf("bpc-%d", f.nextID)
	f.nextID++
	control := &externalEonSdkAPI.BackupPostureControl{
		Id:               id,
		Name:             req.GetName(),
		Severity:         req.GetSeverity(),
		ResourceSelector: req.ResourceSelector,
		Rules:            req.Rules,
	}
	f.controls[id] = control
	writeJSON(w, http.StatusCreated, control)
}

func (f *fakeEonServer) handleGetControl(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	control, ok := f.controls[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, control)
}

func (f *fakeEonServer) handleUpdateControl(w http.ResponseWriter, r *http.Request, id string) {
	var req externalEonSdkAPI.UpdateBackupPostureControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	control, ok := f.controls[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	control.Name = req.GetName()
	control.Severity = req.GetSeverity()
	control.ResourceSelector = req.ResourceSelector
	control.Rules = req.Rules
	f.controls[id] = control
	writeJSON(w, http.StatusOK, control)
}

func (f *fakeEonServer) handleDeleteControl(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.controls[id]; !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	delete(f.controls, id)
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeEonServer) handleListControls(w http.ResponseWriter, r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]externalEonSdkAPI.BackupPostureControl, 0, len(f.controls))
	for _, c := range f.controls {
		items = append(items, *c)
	}
	writeJSON(w, http.StatusOK, externalEonSdkAPI.ListBackupPostureControlsResponse{
		BackupPostureControls: items,
	})
}

func (f *fakeEonServer) handleListIdps(w http.ResponseWriter, r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]externalEonSdkAPI.Idp, 0, len(f.idps))
	for _, idp := range f.idps {
		items = append(items, *idp)
	}
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewListIdpsResponse(items, int32(len(items))))
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newTestIdp(id, name string) *externalEonSdkAPI.Idp {
	return externalEonSdkAPI.NewIdp(id, name)
}
