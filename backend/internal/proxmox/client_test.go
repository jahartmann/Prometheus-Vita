package proxmox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testClient builds a Client pointed at an httptest server. baseURL is set
// directly (white-box) so no TLS/auth handshake is needed; paths are matched
// verbatim by the test mux.
func testClient(srv *httptest.Server) *Client {
	return &Client{baseURL: srv.URL, httpClient: srv.Client()}
}

func TestGetNodeStatusParsing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/nodes/pve1/status", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{
			"cpu":0.5,
			"memory":{"total":1000,"used":400,"free":600},
			"rootfs":{"total":2000,"used":500},
			"cpuinfo":{"cpus":8,"model":"Test CPU"},
			"uptime":1234,
			"loadavg":["0.15","0.20","0.10"]
		}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	st, err := testClient(srv).GetNodeStatus(context.Background(), "pve1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.CPUUsage != 50 {
		t.Errorf("CPUUsage = %v, want 50", st.CPUUsage)
	}
	if st.MemTotal != 1000 || st.MemUsed != 400 {
		t.Errorf("mem = %d/%d, want 400/1000", st.MemUsed, st.MemTotal)
	}
	if st.CPUCores != 8 {
		t.Errorf("CPUCores = %d, want 8", st.CPUCores)
	}
	if st.Uptime != 1234 {
		t.Errorf("Uptime = %d, want 1234", st.Uptime)
	}
	if len(st.LoadAvg) != 3 || st.LoadAvg[0] != 0.15 {
		t.Errorf("LoadAvg = %v, want [0.15 0.20 0.10]", st.LoadAvg)
	}
}

func TestGetNodeStatusClampsOvercommitCPU(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/nodes/pve1/status", func(w http.ResponseWriter, _ *http.Request) {
		// cpu 1.5 → 150%, must be clamped to 100.
		_, _ = w.Write([]byte(`{"data":{"cpu":1.5,"memory":{"total":1,"used":1},"cpuinfo":{"cpus":1}}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	st, err := testClient(srv).GetNodeStatus(context.Background(), "pve1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.CPUUsage != 100 {
		t.Errorf("CPUUsage = %v, want clamped 100", st.CPUUsage)
	}
}

func TestGetVMsMergesQemuAndLxc(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/nodes/pve1/qemu", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"vmid":100,"name":"vm1","status":"running"}]}`))
	})
	mux.HandleFunc("/nodes/pve1/lxc", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"vmid":200,"name":"ct1","status":"stopped"}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	vms, err := testClient(srv).GetVMs(context.Background(), "pve1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vms) != 2 {
		t.Fatalf("got %d vms, want 2", len(vms))
	}
	byID := map[int]VMInfo{}
	for _, v := range vms {
		byID[v.VMID] = v
	}
	if byID[100].Type != "qemu" || byID[100].Name != "vm1" {
		t.Errorf("qemu vm mismatch: %+v", byID[100])
	}
	if byID[200].Type != "lxc" || byID[200].Status != "stopped" {
		t.Errorf("lxc ct mismatch: %+v", byID[200])
	}
}

func TestGetVMsErrorsWhenBothEndpointsFail(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errors":"boom"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if _, err := testClient(srv).GetVMs(context.Background(), "pve1"); err == nil {
		t.Fatalf("expected error when both qemu and lxc endpoints fail")
	}
}

func TestDoRequestErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"auth", http.StatusUnauthorized, `{}`, "authentication failed"},
		{"forbidden", http.StatusForbidden, `{}`, "authentication failed"},
		{"notfound", http.StatusNotFound, `{}`, "API error 404"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/x", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			_, err := testClient(srv).doRequest(context.Background(), http.MethodGet, "/x")
			if err == nil {
				t.Fatalf("expected error for status %d", tc.status)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}
