package ssh

import "time"

// CommandResult holds the output of an SSH command execution.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// SSHConfig holds the configuration for an SSH connection.
type SSHConfig struct {
	Host       string
	Port       int
	User       string
	PrivateKey string
	Password   string
	Timeout    time.Duration
}

// PoolConfig holds the configuration for the SSH connection pool.
type PoolConfig struct {
	MaxConnections int
	IdleTimeout    time.Duration
}
