package apierror

import (
	"fmt"
	"net/http"
)

// ErrorCode identifies a specific error type for frontend consumption.
type ErrorCode string

const (
	ErrVMGuestAgentUnavailable ErrorCode = "VM_GUEST_AGENT_UNAVAILABLE"
	ErrVMNotRunning            ErrorCode = "VM_NOT_RUNNING"
	ErrNodeSSHFailed           ErrorCode = "NODE_SSH_FAILED"
	ErrNodeUnreachable         ErrorCode = "NODE_UNREACHABLE"
	ErrVMPermissionDenied      ErrorCode = "VM_PERMISSION_DENIED"
	ErrVMCommandTimeout        ErrorCode = "VM_COMMAND_TIMEOUT"
	ErrVMPathInvalid           ErrorCode = "VM_PATH_INVALID"
	ErrVMCommandFailed         ErrorCode = "VM_COMMAND_FAILED"
	ErrVMExecFailed            ErrorCode = "VM_EXEC_FAILED"
)

// APIError is a structured error that maps to HTTP responses with actionable info.
type APIError struct {
	Code       ErrorCode `json:"error_code"`
	Message    string    `json:"error"`
	Details    string    `json:"details,omitempty"`
	Hint       string    `json:"hint,omitempty"`
	HTTPStatus int       `json:"-"`
	Cause      error     `json:"-"`
}

func (e *APIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *APIError) Unwrap() error {
	return e.Cause
}

func GuestAgentUnavailable(cause error) *APIError {
	return &APIError{
		Code:       ErrVMGuestAgentUnavailable,
		Message:    "Guest Agent ist nicht verfuegbar",
		Details:    "Der QEMU Guest Agent muss in der VM installiert und gestartet sein.",
		Hint:       "apt install qemu-guest-agent && systemctl enable --now qemu-guest-agent",
		HTTPStatus: http.StatusPreconditionFailed,
		Cause:      cause,
	}
}

func VMNotRunning(vmid int) *APIError {
	return &APIError{
		Code:       ErrVMNotRunning,
		Message:    fmt.Sprintf("VM %d ist nicht gestartet", vmid),
		Details:    "Die VM muss laufen, bevor Cockpit-Operationen ausgefuehrt werden koennen.",
		Hint:       "Starten Sie die VM ueber das Power-Menue.",
		HTTPStatus: http.StatusConflict,
	}
}

func NodeSSHFailed(cause error) *APIError {
	return &APIError{
		Code:       ErrNodeSSHFailed,
		Message:    "SSH-Verbindung zum Node fehlgeschlagen",
		Details:    "Die SSH-Verbindung zum Proxmox-Node konnte nicht hergestellt werden.",
		Hint:       "Pruefen Sie die SSH-Konfiguration des Nodes und ob der Node erreichbar ist.",
		HTTPStatus: http.StatusServiceUnavailable,
		Cause:      cause,
	}
}

func NodeUnreachable(cause error) *APIError {
	return &APIError{
		Code:       ErrNodeUnreachable,
		Message:    "Node ist nicht erreichbar",
		Details:    "Der Proxmox-Node antwortet nicht auf API-Anfragen.",
		Hint:       "Pruefen Sie die Netzwerkverbindung und ob der Node online ist.",
		HTTPStatus: http.StatusServiceUnavailable,
		Cause:      cause,
	}
}

func VMCommandTimeout(cause error) *APIError {
	return &APIError{
		Code:       ErrVMCommandTimeout,
		Message:    "Befehl hat das Zeitlimit ueberschritten",
		Details:    "Der Befehl wurde nach 30 Sekunden abgebrochen.",
		Hint:       "Der Befehl dauert zu lange. Versuchen Sie einen einfacheren Befehl oder pruefen Sie die VM-Auslastung.",
		HTTPStatus: http.StatusGatewayTimeout,
		Cause:      cause,
	}
}

func VMPathInvalid(path string) *APIError {
	return &APIError{
		Code:       ErrVMPathInvalid,
		Message:    fmt.Sprintf("Ungueltiger Dateipfad: %s", path),
		Details:    "Der Pfad enthaelt ungueltige Zeichen oder eine unerlaubte Traversierung.",
		HTTPStatus: http.StatusBadRequest,
	}
}

func VMCommandFailed(exitCode int, stderr string) *APIError {
	return &APIError{
		Code:       ErrVMCommandFailed,
		Message:    fmt.Sprintf("Befehl fehlgeschlagen (Exit-Code %d)", exitCode),
		Details:    stderr,
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

func VMExecFailed(cause error) *APIError {
	return &APIError{
		Code:       ErrVMExecFailed,
		Message:    "Befehlsausfuehrung fehlgeschlagen",
		Details:    cause.Error(),
		HTTPStatus: http.StatusInternalServerError,
		Cause:      cause,
	}
}
