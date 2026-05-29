package vm

import (
	"context"
	"testing"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
)

type fakePower struct{ calls []string }

func (f *fakePower) StartVM(_ context.Context, _ uuid.UUID, _ int, _ string) (string, error) {
	f.calls = append(f.calls, "start")
	return "upid", nil
}
func (f *fakePower) StopVM(_ context.Context, _ uuid.UUID, _ int, _ string) (string, error) {
	f.calls = append(f.calls, "stop")
	return "upid", nil
}
func (f *fakePower) ShutdownVM(_ context.Context, _ uuid.UUID, _ int, _ string) (string, error) {
	f.calls = append(f.calls, "shutdown")
	return "upid", nil
}
func (f *fakePower) RebootVM(_ context.Context, _ uuid.UUID, _ int, _ string) (string, error) {
	f.calls = append(f.calls, "reboot")
	return "upid", nil
}

func TestScheduledActionExecuteDispatch(t *testing.T) {
	vmid := 100
	for _, tc := range []struct{ action, want string }{
		{"start", "start"},
		{"stop", "stop"},
		{"shutdown", "shutdown"},
		{"restart", "reboot"}, // UI "restart" maps to the Proxmox reboot endpoint
	} {
		fp := &fakePower{}
		svc := NewScheduledActionService(nil, fp)
		err := svc.Execute(context.Background(), model.ScheduledAction{
			NodeID: uuid.New(), VMID: &vmid, VMType: "qemu", Action: tc.action,
		})
		if err != nil {
			t.Fatalf("action %s: unexpected error: %v", tc.action, err)
		}
		if len(fp.calls) != 1 || fp.calls[0] != tc.want {
			t.Fatalf("action %q dispatched %v, want [%s]", tc.action, fp.calls, tc.want)
		}
	}
}

func TestScheduledActionExecuteRejectsUnknownAction(t *testing.T) {
	vmid := 100
	svc := NewScheduledActionService(nil, &fakePower{})
	err := svc.Execute(context.Background(), model.ScheduledAction{VMID: &vmid, Action: "frobnicate"})
	if err == nil {
		t.Fatalf("expected error for unknown action")
	}
}

func TestScheduledActionExecuteRequiresVMID(t *testing.T) {
	svc := NewScheduledActionService(nil, &fakePower{})
	err := svc.Execute(context.Background(), model.ScheduledAction{Action: "start"})
	if err == nil {
		t.Fatalf("expected error when vmid is nil")
	}
}
