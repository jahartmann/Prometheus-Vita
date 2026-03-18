package logscan

import (
	"context"
	"log/slog"
	"strings"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

// DiscoveryService seeds builtin log sources and discovers additional log files
// on a node via SSH.
type DiscoveryService struct {
	sourceRepo repository.LogSourceRepository
	sshPool    *ssh.Pool
	nodeRepo   repository.NodeRepository
}

// NewDiscoveryService creates a new DiscoveryService.
func NewDiscoveryService(
	sourceRepo repository.LogSourceRepository,
	sshPool *ssh.Pool,
	nodeRepo repository.NodeRepository,
) *DiscoveryService {
	return &DiscoveryService{
		sourceRepo: sourceRepo,
		sshPool:    sshPool,
		nodeRepo:   nodeRepo,
	}
}

// SeedBuiltinSources ensures that all entries from BuiltinSources exist in the
// database for the given node.  Existing rows are updated via upsert so the
// enabled / parser_type values are preserved.
func (s *DiscoveryService) SeedBuiltinSources(ctx context.Context, nodeID uuid.UUID) error {
	for _, b := range BuiltinSources {
		src := &model.LogSource{
			NodeID:     nodeID,
			Path:       b.Path,
			IsBuiltin:  true,
			Enabled:    true,
			ParserType: b.ParserType,
		}
		if err := s.sourceRepo.Upsert(ctx, src); err != nil {
			slog.Error("logscan: failed to upsert builtin source",
				slog.String("node_id", nodeID.String()),
				slog.String("path", b.Path),
				slog.Any("error", err),
			)
			return err
		}
	}
	slog.Debug("logscan: builtin sources seeded",
		slog.String("node_id", nodeID.String()),
		slog.Int("count", len(BuiltinSources)),
	)
	return nil
}

// DiscoverSources seeds builtin sources, then runs a remote `find` command on
// the node to locate recently-modified log files.  Any newly found paths that
// are not already a builtin are upserted as disabled custom sources.  The full
// list of sources for the node is returned.
func (s *DiscoveryService) DiscoverSources(ctx context.Context, nodeID uuid.UUID) ([]model.LogSource, error) {
	// Always ensure builtins are present first.
	if err := s.SeedBuiltinSources(ctx, nodeID); err != nil {
		return nil, err
	}

	// Build the set of builtin paths for fast lookup.
	builtinPaths := make(map[string]struct{}, len(BuiltinSources))
	for _, b := range BuiltinSources {
		builtinPaths[b.Path] = struct{}{}
	}

	// Look up the node to build the SSH config.
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	sshCfg := ssh.SSHConfig{
		Host:       node.Hostname,
		Port:       node.SSHPort,
		User:       node.SSHUser,
		PrivateKey: node.SSHPrivateKey,
	}
	if sshCfg.Port == 0 {
		sshCfg.Port = 22
	}
	if sshCfg.User == "" {
		sshCfg.User = "root"
	}

	// Obtain a pooled SSH client for this node.
	client, err := s.sshPool.Get(nodeID.String(), sshCfg)
	if err != nil {
		slog.Warn("logscan: cannot connect to node for discovery",
			slog.String("node_id", nodeID.String()),
			slog.Any("error", err),
		)
		// Return whatever is already in the DB rather than a hard error.
		return s.sourceRepo.ListByNode(ctx, nodeID)
	}
	defer s.sshPool.Return(nodeID.String(), client)

	// Discover log files modified within the last 60 minutes.
	const findCmd = `find /var/log -type f \( -name "*.log" -o -name "syslog*" -o -name "auth.log*" \) -mmin -60 2>/dev/null`
	result, err := client.RunCommand(ctx, findCmd)
	if err != nil {
		slog.Warn("logscan: find command failed",
			slog.String("node_id", nodeID.String()),
			slog.Any("error", err),
		)
		// Non-fatal — return existing DB sources.
		return s.sourceRepo.ListByNode(ctx, nodeID)
	}

	// Parse one path per line; upsert non-builtin paths as disabled sources.
	for _, line := range strings.Split(result.Stdout, "\n") {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		if _, isBuiltin := builtinPaths[path]; isBuiltin {
			continue
		}

		src := &model.LogSource{
			NodeID:     nodeID,
			Path:       path,
			IsBuiltin:  false,
			Enabled:    false,
			ParserType: "syslog",
		}
		if err := s.sourceRepo.Upsert(ctx, src); err != nil {
			slog.Warn("logscan: failed to upsert discovered source",
				slog.String("node_id", nodeID.String()),
				slog.String("path", path),
				slog.Any("error", err),
			)
		}
	}

	return s.sourceRepo.ListByNode(ctx, nodeID)
}
