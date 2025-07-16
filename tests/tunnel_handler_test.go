package tests

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/MmadF14/vwireguard/handler"
	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
)

type memStore struct{ tunnels map[string]model.Tunnel }

func (m *memStore) Init() error                              { return nil }
func (m *memStore) GetUsers() ([]model.User, error)          { return nil, nil }
func (m *memStore) GetUserByName(string) (model.User, error) { return model.User{}, nil }
func (m *memStore) SaveUser(model.User) error                { return nil }
func (m *memStore) DeleteUser(string) error                  { return nil }
func (m *memStore) GetGlobalSettings() (model.GlobalSetting, error) {
	return model.GlobalSetting{}, nil
}
func (m *memStore) GetServer() (model.Server, error)            { return model.Server{}, nil }
func (m *memStore) GetClients(bool) ([]model.ClientData, error) { return nil, nil }
func (m *memStore) GetClientByID(string, model.QRCodeSettings) (model.ClientData, error) {
	return model.ClientData{}, nil
}
func (m *memStore) SaveClient(model.Client) error                         { return nil }
func (m *memStore) DeleteClient(string) error                             { return nil }
func (m *memStore) SaveServerInterface(model.ServerInterface) error       { return nil }
func (m *memStore) SaveServerKeyPair(model.ServerKeypair) error           { return nil }
func (m *memStore) SaveGlobalSettings(model.GlobalSetting) error          { return nil }
func (m *memStore) GetWakeOnLanHosts() ([]model.WakeOnLanHost, error)     { return nil, nil }
func (m *memStore) GetWakeOnLanHost(string) (*model.WakeOnLanHost, error) { return nil, nil }
func (m *memStore) DeleteWakeOnHostLanHost(string) error                  { return nil }
func (m *memStore) SaveWakeOnLanHost(model.WakeOnLanHost) error           { return nil }
func (m *memStore) DeleteWakeOnHost(model.WakeOnLanHost) error            { return nil }
func (m *memStore) GetPath() string                                       { return "" }
func (m *memStore) SaveHashes(model.ClientServerHashes) error             { return nil }
func (m *memStore) GetHashes() (model.ClientServerHashes, error) {
	return model.ClientServerHashes{}, nil
}
func (m *memStore) GetTunnels() ([]model.Tunnel, error) {
	ts := []model.Tunnel{}
	for _, t := range m.tunnels {
		ts = append(ts, t)
	}
	return ts, nil
}
func (m *memStore) GetTunnelByID(id string) (model.Tunnel, error) { return m.tunnels[id], nil }
func (m *memStore) SaveTunnel(t model.Tunnel) error               { m.tunnels[t.ID] = t; return nil }
func (m *memStore) DeleteTunnel(string) error                     { return nil }
func (m *memStore) UpdateTunnelStatus(id string, s model.TunnelStatus) error {
	t := m.tunnels[id]
	t.Status = s
	m.tunnels[id] = t
	return nil
}
func (m *memStore) UpdateTunnelStats(string, int64, int64) error { return nil }

func TestRouteAllConflict(t *testing.T) {
	os.Setenv("VWIREGUARD_TEST", "1")
	db := &memStore{tunnels: map[string]model.Tunnel{
		"a": {ID: "a", RouteAll: true, Status: model.TunnelStatusInactive, Enabled: true},
		"b": {ID: "b", RouteAll: true, Status: model.TunnelStatusInactive, Enabled: true},
	}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:id/start")
	c.SetParamNames("id")
	c.SetParamValues("a")
	if err := handler.StartTunnel(db)(c); err != nil {
		t.Fatal(err)
	}
	if db.tunnels["a"].Status != model.TunnelStatusActive {
		t.Fatalf("first tunnel not active")
	}
	req2 := httptest.NewRequest(http.MethodPut, "/", nil)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.SetPath("/:id/start")
	c2.SetParamNames("id")
	c2.SetParamValues("b")
	if err := handler.StartTunnel(db)(c2); err != nil {
		t.Fatal(err)
	}
	if db.tunnels["a"].Status != model.TunnelStatusInactive || db.tunnels["b"].Status != model.TunnelStatusActive {
		t.Fatalf("route-all not switched")
	}
}
