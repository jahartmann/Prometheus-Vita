package drift

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/llm"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/backup"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

const aiAnalysisModel = "claude-sonnet-4-20250514"

type Service struct {
	driftRepo   repository.DriftRepository
	backupRepo  repository.BackupRepository
	fileRepo    repository.BackupFileRepository
	nodeRepo    repository.NodeRepository
	encryptor   *crypto.Encryptor
	sshPool     *ssh.Pool
	collector   *backup.FileCollector
	llmRegistry *llm.Registry
}

func NewService(
	driftRepo repository.DriftRepository,
	backupRepo repository.BackupRepository,
	fileRepo repository.BackupFileRepository,
	nodeRepo repository.NodeRepository,
	encryptor *crypto.Encryptor,
	sshPool *ssh.Pool,
	llmRegistry *llm.Registry,
) *Service {
	return &Service{
		driftRepo:   driftRepo,
		backupRepo:  backupRepo,
		fileRepo:    fileRepo,
		nodeRepo:    nodeRepo,
		encryptor:   encryptor,
		sshPool:     sshPool,
		collector:   backup.NewFileCollector(),
		llmRegistry: llmRegistry,
	}
}

func (s *Service) CheckDrift(ctx context.Context, nodeID uuid.UUID) (*model.DriftCheck, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	check := &model.DriftCheck{
		NodeID:    nodeID,
		Status:    model.DriftStatusRunning,
		CheckedAt: time.Now().UTC(),
	}
	if err := s.driftRepo.Create(ctx, check); err != nil {
		return nil, fmt.Errorf("create drift check: %w", err)
	}

	// Get latest backup for comparison
	latestBackup, err := s.backupRepo.GetLatestByNode(ctx, nodeID)
	if err != nil {
		check.Status = model.DriftStatusFailed
		check.ErrorMessage = "no backup found for comparison"
		if err := s.driftRepo.Update(ctx, check); err != nil {
			slog.Error("failed to update drift check status",
				slog.String("check_id", check.ID.String()),
				slog.String("status", check.Status),
				slog.Any("error", err),
			)
		}
		return check, nil
	}

	// Get backup files
	backupFiles, err := s.fileRepo.GetByBackupID(ctx, latestBackup.ID)
	if err != nil {
		check.Status = model.DriftStatusFailed
		check.ErrorMessage = fmt.Sprintf("failed to get backup files: %v", err)
		if err := s.driftRepo.Update(ctx, check); err != nil {
			slog.Error("failed to update drift check status",
				slog.String("check_id", check.ID.String()),
				slog.String("status", check.Status),
				slog.Any("error", err),
			)
		}
		return check, nil
	}

	// Decrypt SSH credentials and connect
	privateKey, err := s.encryptor.Decrypt(node.SSHPrivateKey)
	if err != nil {
		check.Status = model.DriftStatusFailed
		check.ErrorMessage = "failed to decrypt SSH key"
		if err := s.driftRepo.Update(ctx, check); err != nil {
			slog.Error("failed to update drift check status",
				slog.String("check_id", check.ID.String()),
				slog.String("status", check.Status),
				slog.Any("error", err),
			)
		}
		return check, nil
	}

	sshClient, err := s.sshPool.Get(nodeID.String(), ssh.SSHConfig{
		Host:       node.Hostname,
		Port:       node.SSHPort,
		User:       node.SSHUser,
		PrivateKey: privateKey,
		HostKey:    node.SSHHostKey,
	})
	if err != nil {
		check.Status = model.DriftStatusFailed
		check.ErrorMessage = fmt.Sprintf("SSH connection failed: %v", err)
		if err := s.driftRepo.Update(ctx, check); err != nil {
			slog.Error("failed to update drift check status",
				slog.String("check_id", check.ID.String()),
				slog.String("status", check.Status),
				slog.Any("error", err),
			)
		}
		return check, nil
	}
	defer s.sshPool.Return(nodeID.String(), sshClient)

	// Collect current files from node
	currentFiles, err := s.collector.CollectFiles(ctx, sshClient, backup.DefaultPaths)
	if err != nil {
		check.Status = model.DriftStatusFailed
		check.ErrorMessage = fmt.Sprintf("failed to collect files: %v", err)
		if err := s.driftRepo.Update(ctx, check); err != nil {
			slog.Error("failed to update drift check status",
				slog.String("check_id", check.ID.String()),
				slog.String("status", check.Status),
				slog.Any("error", err),
			)
		}
		return check, nil
	}

	// Convert backup files to CollectedFile format for comparison
	var oldFiles []backup.CollectedFile
	for _, bf := range backupFiles {
		oldFiles = append(oldFiles, backup.CollectedFile{
			Path:    bf.FilePath,
			Hash:    bf.FileHash,
			Content: bf.Content,
			Size:    bf.FileSize,
		})
	}

	// Diff
	diffs := backup.DiffFiles(oldFiles, currentFiles)

	// Count changes
	var changed, added, removed int
	var details []model.DriftFileDetail
	for _, d := range diffs {
		switch d.Status {
		case "modified":
			changed++
			details = append(details, model.DriftFileDetail{FilePath: d.FilePath, Status: d.Status, Diff: d.Diff})
		case "added":
			added++
			details = append(details, model.DriftFileDetail{FilePath: d.FilePath, Status: d.Status})
		case "removed":
			removed++
			details = append(details, model.DriftFileDetail{FilePath: d.FilePath, Status: d.Status})
		}
	}

	detailsJSON, err := json.Marshal(details)
	if err != nil {
		slog.Error("failed to marshal drift details",
			slog.String("check_id", check.ID.String()),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("marshal drift details: %w", err)
	}

	check.Status = model.DriftStatusCompleted
	check.TotalFiles = len(currentFiles)
	check.ChangedFiles = changed
	check.AddedFiles = added
	check.RemovedFiles = removed
	check.Details = detailsJSON
	check.CheckedAt = time.Now().UTC()

	if err := s.driftRepo.Update(ctx, check); err != nil {
		return nil, fmt.Errorf("update drift check: %w", err)
	}

	slog.Info("drift check completed",
		slog.String("node_id", nodeID.String()),
		slog.Int("changed", changed),
		slog.Int("added", added),
		slog.Int("removed", removed),
	)

	// Run AI analysis asynchronously if there are changes
	if (changed + added + removed) > 0 {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("AI drift analysis goroutine panicked",
						slog.String("check_id", check.ID.String()),
						slog.Any("panic", r),
						slog.String("stack", string(debug.Stack())),
					)
				}
			}()
			bgCtx := context.Background()
			if err := s.analyzeDriftWithAI(bgCtx, check); err != nil {
				slog.Warn("AI drift analysis failed",
					slog.String("check_id", check.ID.String()),
					slog.Any("error", err),
				)
			}
		}()
	}

	return check, nil
}

func (s *Service) analyzeDriftWithAI(ctx context.Context, check *model.DriftCheck) error {
	if s.llmRegistry == nil {
		return fmt.Errorf("LLM registry not configured")
	}

	provider, err := s.llmRegistry.GetForModel(aiAnalysisModel)
	if err != nil {
		return fmt.Errorf("get LLM provider: %w", err)
	}

	// Parse details
	var details []model.DriftFileDetail
	if err := json.Unmarshal(check.Details, &details); err != nil {
		return fmt.Errorf("parse drift details: %w", err)
	}

	if len(details) == 0 {
		return nil
	}

	// Build the prompt with all file changes
	var fileDescriptions []string
	for _, d := range details {
		desc := fmt.Sprintf("Datei: %s\nAenderungstyp: %s", d.FilePath, d.Status)
		if d.Diff != "" {
			// Truncate very long diffs to avoid token limits
			diff := d.Diff
			if len(diff) > 3000 {
				diff = diff[:3000] + "\n... (gekuerzt)"
			}
			desc += fmt.Sprintf("\nDiff:\n```\n%s\n```", diff)
		}
		fileDescriptions = append(fileDescriptions, desc)
	}

	prompt := fmt.Sprintf(`Du bist ein Proxmox-Infrastruktur-Sicherheitsexperte. Analysiere die folgenden Konfigurationsaenderungen, die durch einen Drift-Check erkannt wurden.

Fuer JEDE geaenderte Datei, gib eine Analyse im folgenden JSON-Format:

{
  "file_analyses": [
    {
      "file_path": "/pfad/zur/datei",
      "severity": 5,
      "severity_reason": "Grund fuer die Schwere",
      "category": "Security|Performance|Network|Configuration|Cosmetic",
      "risk_assessment": "Was koennte schiefgehen",
      "recommendation": "fix|accept|monitor",
      "summary": "Kurze deutsche Zusammenfassung"
    }
  ],
  "overall_severity": 5,
  "overall_summary": "Gesamtbewertung auf Deutsch"
}

Severity-Skala:
- 1-3: Niedrig (kosmetisch, Kommentare, unwichtige Aenderungen)
- 4-6: Mittel (Konfigurationsaenderungen die beobachtet werden sollten)
- 7-10: Hoch/Kritisch (Sicherheitsrelevant, Netzwerk-/Firewall-Aenderungen, SSH-Konfiguration)

Kategorien:
- Security: SSH-Keys, Firewall, Authentifizierung, Zertifikate
- Performance: CPU/RAM-Limits, Swap, Scheduler-Einstellungen
- Network: Netzwerkkonfiguration, Bridges, VLANs, DNS
- Configuration: Proxmox-Cluster, Storage, allgemeine Konfiguration
- Cosmetic: Kommentare, Formatierung, unwichtige Aenderungen

Empfehlungen:
- fix: Aenderung sollte rueckgaengig gemacht werden
- accept: Aenderung ist akzeptabel und kann als neue Baseline uebernommen werden
- monitor: Aenderung beobachten, noch nicht handeln

Antworte NUR mit dem JSON-Objekt, ohne zusaetzlichen Text.

Geaenderte Dateien:

%s`, strings.Join(fileDescriptions, "\n\n---\n\n"))

	req := llm.CompletionRequest{
		Model: aiAnalysisModel,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 4096,
	}

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		return fmt.Errorf("LLM completion: %w", err)
	}

	// Parse the AI response
	content := strings.TrimSpace(resp.Content)
	// Remove markdown code block markers if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var aiResult struct {
		FileAnalyses    []model.AIFileAnalysis `json:"file_analyses"`
		OverallSeverity int                    `json:"overall_severity"`
		OverallSummary  string                 `json:"overall_summary"`
	}
	if err := json.Unmarshal([]byte(content), &aiResult); err != nil {
		slog.Warn("failed to parse AI analysis response",
			slog.String("content", content),
			slog.Any("error", err),
		)
		return fmt.Errorf("parse AI response: %w", err)
	}

	// Build the analysis result
	analysisResult := model.AIAnalysisResult{
		AnalyzedAt:      time.Now().UTC(),
		Model:           aiAnalysisModel,
		FileAnalyses:    aiResult.FileAnalyses,
		OverallSeverity: aiResult.OverallSeverity,
		OverallSummary:  aiResult.OverallSummary,
	}

	analysisJSON, err := json.Marshal(analysisResult)
	if err != nil {
		return fmt.Errorf("marshal AI analysis: %w", err)
	}

	// Update the file details with per-file AI analysis
	fileAnalysisMap := make(map[string]*model.AIFileAnalysis)
	for i := range aiResult.FileAnalyses {
		fa := &aiResult.FileAnalyses[i]
		fileAnalysisMap[fa.FilePath] = fa
	}

	// Enrich details with AI analysis
	for i := range details {
		if fa, ok := fileAnalysisMap[details[i].FilePath]; ok {
			details[i].AIFileAnalysis = fa
		}
	}

	enrichedDetailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal enriched drift details: %w", err)
	}

	// Update the drift check with AI analysis
	check.AIAnalysis = analysisJSON
	check.Details = enrichedDetailsJSON
	if err := s.driftRepo.Update(ctx, check); err != nil {
		return fmt.Errorf("update drift check with AI analysis: %w", err)
	}

	slog.Info("AI drift analysis completed",
		slog.String("check_id", check.ID.String()),
		slog.Int("overall_severity", aiResult.OverallSeverity),
	)

	return nil
}

// AcceptBaseline accepts the current state as the new baseline for a drift check.
func (s *Service) AcceptBaseline(ctx context.Context, checkID uuid.UUID) error {
	check, err := s.driftRepo.GetByID(ctx, checkID)
	if err != nil {
		return fmt.Errorf("get drift check: %w", err)
	}

	now := time.Now().UTC()
	check.BaselineUpdatedAt = &now

	// Clear the drift details since we accepted them
	emptyDetails, err := json.Marshal([]model.DriftFileDetail{})
	if err != nil {
		return fmt.Errorf("marshal empty details: %w", err)
	}
	check.Details = emptyDetails
	check.ChangedFiles = 0
	check.AddedFiles = 0
	check.RemovedFiles = 0

	if err := s.driftRepo.Update(ctx, check); err != nil {
		return fmt.Errorf("update drift check baseline: %w", err)
	}

	slog.Info("baseline accepted",
		slog.String("check_id", checkID.String()),
	)

	return nil
}

// IgnoreDrift marks a specific file change as acknowledged in a drift check.
func (s *Service) IgnoreDrift(ctx context.Context, checkID uuid.UUID, filePath string) error {
	check, err := s.driftRepo.GetByID(ctx, checkID)
	if err != nil {
		return fmt.Errorf("get drift check: %w", err)
	}

	var details []model.DriftFileDetail
	if err := json.Unmarshal(check.Details, &details); err != nil {
		return fmt.Errorf("parse details: %w", err)
	}

	found := false
	for i := range details {
		if details[i].FilePath == filePath {
			details[i].Acknowledged = true
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("file not found in drift details: %s", filePath)
	}

	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal drift details: %w", err)
	}
	check.Details = detailsJSON

	if err := s.driftRepo.Update(ctx, check); err != nil {
		return fmt.Errorf("update drift check: %w", err)
	}

	return nil
}

// CompareNodes compares specified files across multiple nodes.
func (s *Service) CompareNodes(ctx context.Context, req model.CompareNodesRequest) (*model.CompareNodesResponse, error) {
	if len(req.NodeIDs) < 2 {
		return nil, fmt.Errorf("at least 2 nodes required for comparison")
	}
	if len(req.FilePaths) == 0 {
		return nil, fmt.Errorf("at least 1 file path required")
	}

	// Resolve nodes
	type nodeInfo struct {
		id   uuid.UUID
		name string
		client *ssh.Client
	}

	var nodes []nodeInfo
	for _, nid := range req.NodeIDs {
		nodeID, err := uuid.Parse(nid)
		if err != nil {
			return nil, fmt.Errorf("invalid node ID %s: %w", nid, err)
		}

		node, err := s.nodeRepo.GetByID(ctx, nodeID)
		if err != nil {
			return nil, fmt.Errorf("get node %s: %w", nid, err)
		}

		privateKey, err := s.encryptor.Decrypt(node.SSHPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt SSH key for node %s: %w", node.Name, err)
		}

		sshClient, err := s.sshPool.Get(nodeID.String(), ssh.SSHConfig{
			Host:       node.Hostname,
			Port:       node.SSHPort,
			User:       node.SSHUser,
			PrivateKey: privateKey,
			HostKey:    node.SSHHostKey,
		})
		if err != nil {
			return nil, fmt.Errorf("SSH to node %s: %w", node.Name, err)
		}

		nodes = append(nodes, nodeInfo{id: nodeID, name: node.Name, client: sshClient})
	}

	var comparisons []model.NodeComparisonEntry

	for _, filePath := range req.FilePaths {
		entry := model.NodeComparisonEntry{
			FilePath: filePath,
		}

		// Collect file from each node
		for _, n := range nodes {
			nfc := model.NodeFileContent{
				NodeID:   n.id.String(),
				NodeName: n.name,
			}

			content, err := n.client.CopyFrom(ctx, filePath)
			if err != nil {
				nfc.Error = fmt.Sprintf("Datei konnte nicht gelesen werden: %v", err)
			} else {
				nfc.Content = string(content)
			}

			entry.NodeFiles = append(entry.NodeFiles, nfc)
		}

		// Generate pairwise diffs
		for i := 0; i < len(entry.NodeFiles); i++ {
			for j := i + 1; j < len(entry.NodeFiles); j++ {
				a := entry.NodeFiles[i]
				b := entry.NodeFiles[j]

				diff := model.NodeDifference{
					NodeA:     a.NodeID,
					NodeAName: a.NodeName,
					NodeB:     b.NodeID,
					NodeBName: b.NodeName,
				}

				if a.Error != "" || b.Error != "" {
					diff.Identical = false
					if a.Error != "" {
						diff.Diff = fmt.Sprintf("Node %s: %s", a.NodeName, a.Error)
					} else {
						diff.Diff = fmt.Sprintf("Node %s: %s", b.NodeName, b.Error)
					}
				} else {
					hashA := fmt.Sprintf("%x", sha256.Sum256([]byte(a.Content)))
					hashB := fmt.Sprintf("%x", sha256.Sum256([]byte(b.Content)))

					if hashA == hashB {
						diff.Identical = true
					} else {
						diff.Identical = false
						diff.Diff = generateSimpleDiff(a.Content, b.Content)
					}
				}

				entry.Differences = append(entry.Differences, diff)
			}
		}

		comparisons = append(comparisons, entry)
	}

	return &model.CompareNodesResponse{
		Comparisons: comparisons,
	}, nil
}

// generateSimpleDiff produces a unified diff between two strings.
func generateSimpleDiff(oldContent, newContent string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	oldLen := len(oldLines)
	newLen := len(newLines)

	lcs := make([][]int, oldLen+1)
	for i := range lcs {
		lcs[i] = make([]int, newLen+1)
	}

	for i := 1; i <= oldLen; i++ {
		for j := 1; j <= newLen; j++ {
			if oldLines[i-1] == newLines[j-1] {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else if lcs[i-1][j] >= lcs[i][j-1] {
				lcs[i][j] = lcs[i-1][j]
			} else {
				lcs[i][j] = lcs[i][j-1]
			}
		}
	}

	var stack []string
	i, j := oldLen, newLen
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			stack = append(stack, " "+oldLines[i-1])
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			stack = append(stack, "+"+newLines[j-1])
			j--
		} else if i > 0 {
			stack = append(stack, "-"+oldLines[i-1])
			i--
		}
	}

	var diffLines []string
	for k := len(stack) - 1; k >= 0; k-- {
		diffLines = append(diffLines, stack[k])
	}

	if len(diffLines) == 0 {
		return ""
	}

	return fmt.Sprintf("--- a\n+++ b\n%s", strings.Join(diffLines, "\n"))
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*model.DriftCheck, error) {
	return s.driftRepo.GetByID(ctx, id)
}

func (s *Service) GetLatestByNode(ctx context.Context, nodeID uuid.UUID) (*model.DriftCheck, error) {
	return s.driftRepo.GetLatestByNode(ctx, nodeID)
}

func (s *Service) ListByNode(ctx context.Context, nodeID uuid.UUID, limit int) ([]model.DriftCheck, error) {
	return s.driftRepo.ListByNode(ctx, nodeID, limit)
}

func (s *Service) ListAll(ctx context.Context, limit int) ([]model.DriftCheck, error) {
	return s.driftRepo.ListAll(ctx, limit)
}
