package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	accTestProjectID    = "acc-project-id"
	accTestClientID     = "acc-client-id"
	accTestClientSecret = "acc-client-secret"
)

// fakeEonServer is an in-memory Eon API used by acceptance tests.
type fakeEonServer struct {
	mu                    sync.Mutex
	server                *httptest.Server
	backupPostureControls map[string]*externalEonSdkAPI.BackupPostureControl
	idps                  map[string]*externalEonSdkAPI.Idp
	nextControlID         int
}

func newFakeEonServer() *fakeEonServer {
	f := &fakeEonServer{
		backupPostureControls: make(map[string]*externalEonSdkAPI.BackupPostureControl),
		idps:                  make(map[string]*externalEonSdkAPI.Idp),
		nextControlID:         1,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", f.handle)
	f.server = httptest.NewServer(mux)
	return f
}

func (f *fakeEonServer) URL() string {
	return f.server.URL
}

func (f *fakeEonServer) Close() {
	f.server.Close()
}

func (f *fakeEonServer) DeleteBackupPostureControl(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.backupPostureControls, id)
}

func (f *fakeEonServer) BackupPostureControlIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]string, 0, len(f.backupPostureControls))
	for id := range f.backupPostureControls {
		ids = append(ids, id)
	}
	return ids
}

func (f *fakeEonServer) SeedIdp(idp externalEonSdkAPI.Idp) {
	f.mu.Lock()
	defer f.mu.Unlock()
	copied := idp
	f.idps[idp.Id] = &copied
}

func (f *fakeEonServer) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case r.Method == http.MethodPost && path == "/v1/token":
		f.writeJSON(w, http.StatusOK, map[string]interface{}{
			"accessToken":       "acc-test-token",
			"expirationSeconds": 3600,
		})
		return

	case r.Method == http.MethodPost && path == fmt.Sprintf("/v1/projects/%s/backup-posture-controls", accTestProjectID):
		f.createBackupPostureControl(w, r)
		return

	case r.Method == http.MethodPost && path == fmt.Sprintf("/v1/projects/%s/backup-posture-controls/list", accTestProjectID):
		f.listBackupPostureControls(w, r)
		return

	case strings.HasPrefix(path, fmt.Sprintf("/v1/projects/%s/backup-posture-controls/", accTestProjectID)):
		id := strings.TrimPrefix(path, fmt.Sprintf("/v1/projects/%s/backup-posture-controls/", accTestProjectID))
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			f.getBackupPostureControl(w, id)
		case http.MethodPut:
			f.updateBackupPostureControl(w, r, id)
		case http.MethodDelete:
			f.deleteBackupPostureControl(w, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return

	case r.Method == http.MethodPost && path == "/v1/idps/list":
		f.listIdps(w, r)
		return
	}

	http.Error(w, fmt.Sprintf("no route for %s %s", r.Method, path), http.StatusNotFound)
}

func (f *fakeEonServer) createBackupPostureControl(w http.ResponseWriter, r *http.Request) {
	var req externalEonSdkAPI.CreateBackupPostureControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	id := fmt.Sprintf("bpc-%d", f.nextControlID)
	f.nextControlID++
	control := &externalEonSdkAPI.BackupPostureControl{
		Id:               id,
		Name:             req.Name,
		Severity:         req.Severity,
		ResourceSelector: req.ResourceSelector,
		Rules:            req.Rules,
	}
	f.backupPostureControls[id] = control
	f.writeJSON(w, http.StatusOK, control)
}

func (f *fakeEonServer) getBackupPostureControl(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	control, ok := f.backupPostureControls[id]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	f.writeJSON(w, http.StatusOK, control)
}

func (f *fakeEonServer) updateBackupPostureControl(w http.ResponseWriter, r *http.Request, id string) {
	var req externalEonSdkAPI.UpdateBackupPostureControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	control, ok := f.backupPostureControls[id]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	control.Name = req.Name
	control.Severity = req.Severity
	control.ResourceSelector = req.ResourceSelector
	control.Rules = req.Rules
	f.backupPostureControls[id] = control
	f.writeJSON(w, http.StatusOK, control)
}

func (f *fakeEonServer) deleteBackupPostureControl(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.backupPostureControls[id]; !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	delete(f.backupPostureControls, id)
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeEonServer) listBackupPostureControls(w http.ResponseWriter, r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()
	controls := make([]externalEonSdkAPI.BackupPostureControl, 0, len(f.backupPostureControls))
	for _, c := range f.backupPostureControls {
		controls = append(controls, *c)
	}
	sort.Slice(controls, func(i, j int) bool { return controls[i].Id < controls[j].Id })
	total := int32(len(controls))
	resp := externalEonSdkAPI.ListBackupPostureControlsResponse{
		BackupPostureControls: controls,
		TotalCount:            &total,
	}
	f.writeJSON(w, http.StatusOK, resp)
}

func (f *fakeEonServer) listIdps(w http.ResponseWriter, r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()
	idps := make([]externalEonSdkAPI.Idp, 0, len(f.idps))
	for _, idp := range f.idps {
		idps = append(idps, *idp)
	}
	sort.Slice(idps, func(i, j int) bool { return idps[i].Id < idps[j].Id })
	resp := externalEonSdkAPI.NewListIdpsResponse(idps, int32(len(idps)))
	f.writeJSON(w, http.StatusOK, resp)
}

func (f *fakeEonServer) writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func protoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"eon": providerserver.NewProtocol6WithError(New("test", client.NewClient)()),
	}
}

func setupAccTestEnv(t *testing.T, fake *fakeEonServer) {
	t.Helper()
	t.Setenv("EON_ENDPOINT", fake.URL())
	t.Setenv("EON_CLIENT_ID", accTestClientID)
	t.Setenv("EON_CLIENT_SECRET", accTestClientSecret)
	t.Setenv("EON_PROJECT_ID", accTestProjectID)
	t.Setenv("EON_USE_EXACT_ENDPOINT", "true")
	t.Setenv("EON_DEFAULT_HEADERS", "Authorization:Bearer acc-test-token")
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
}
