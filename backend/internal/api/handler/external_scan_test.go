package handler

import "testing"

func TestExtractVMMAC(t *testing.T) {
	cfg := map[string]interface{}{
		"name": "web",
		"net0": "virtio=BC:24:11:AA:BB:CC,bridge=vmbr0,firewall=1",
	}
	if got := extractVMMAC(cfg); got != "BC:24:11:AA:BB:CC" {
		t.Fatalf("got %q, want BC:24:11:AA:BB:CC", got)
	}

	// Non-contiguous interface index is still found.
	cfg2 := map[string]interface{}{"net3": "e1000=00:11:22:33:44:55,bridge=vmbr1"}
	if got := extractVMMAC(cfg2); got != "00:11:22:33:44:55" {
		t.Fatalf("got %q, want 00:11:22:33:44:55", got)
	}

	// No network device → empty.
	if got := extractVMMAC(map[string]interface{}{"name": "x"}); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestParseNmapPorts(t *testing.T) {
	out := "Host: 192.168.1.50 ()\tPorts: 22/open/tcp//ssh//, 80/open/tcp//http//nginx 1.24/, 3306/closed/tcp//mysql//\tIgnored State: closed (10)"
	ports := parseNmapPorts(out)
	if len(ports) != 2 {
		t.Fatalf("got %d open ports, want 2: %+v", len(ports), ports)
	}
	if ports[0].LocalPort != 22 || ports[0].Protocol != "tcp" || ports[0].Process != "ssh" {
		t.Fatalf("port[0] = %+v", ports[0])
	}
	if ports[1].LocalPort != 80 || ports[1].Process != "http" {
		t.Fatalf("port[1] = %+v", ports[1])
	}
	// Closed ports are excluded.
	for _, p := range ports {
		if p.LocalPort == 3306 {
			t.Fatalf("closed port 3306 must not appear")
		}
	}
}
