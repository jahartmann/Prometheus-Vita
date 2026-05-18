package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/llm"
)

// SystemSpec describes the hardware Prometheus is currently running on.
// It is used to recommend an appropriate Ollama model for the agent.
type SystemSpec struct {
	OS         string    `json:"os"`
	Arch       string    `json:"arch"`
	CPUCores   int       `json:"cpu_cores"`
	TotalRAMGB float64   `json:"total_ram_gb"`
	GPUs       []GPUInfo `json:"gpus"`
	Tier       string    `json:"tier"`
	TierLabel  string    `json:"tier_label"`
	Notes      []string  `json:"notes,omitempty"`
}

// GPUInfo describes a single detected GPU.
type GPUInfo struct {
	Name   string  `json:"name"`
	VRAMGB float64 `json:"vram_gb"`
	Driver string  `json:"driver,omitempty"`
	Vendor string  `json:"vendor,omitempty"`
}

// ModelRecommendation describes a curated Ollama model suggestion.
type ModelRecommendation struct {
	Name        string   `json:"name"`         // Ollama tag, e.g. "qwen2.5:14b"
	Tier        string   `json:"tier"`         // minimal | light | standard | pro | enterprise
	SizeGB      float64  `json:"size_gb"`      // approximate disk/VRAM requirement
	ToolCalling bool     `json:"tool_calling"` // true if model has stable tool/function calling
	Reasoning   bool     `json:"reasoning"`    // chain-of-thought capable
	Description string   `json:"description"`
	BestFor     []string `json:"best_for"`    // ["agent","chat","coding","reasoning"]
	Pulled      bool     `json:"pulled"`      // true if currently installed in Ollama
	Recommended bool     `json:"recommended"` // true for the primary suggestion
	Default     bool     `json:"default"`     // true for the auto-selected default
}

// SystemRecommendation bundles the detected spec, a sorted list of model
// suggestions, and the model that the backend would activate by default.
type SystemRecommendation struct {
	System       SystemSpec            `json:"system"`
	Models       []ModelRecommendation `json:"models"`
	DefaultModel string                `json:"default_model"`
	OllamaReady  bool                  `json:"ollama_ready"`
	OllamaURL    string                `json:"ollama_url"`
}

// catalog is the curated list of Ollama models we know about. The order is
// deliberate — within a tier we prefer the model with the strongest tool-calling.
var modelCatalog = []ModelRecommendation{
	// --- enterprise tier (>= 64 GB available memory) ---
	{
		Name:        "llama3.3:70b",
		Tier:        "enterprise",
		SizeGB:      43,
		ToolCalling: true,
		Description: "Meta Llama 3.3 70B — top-tier tool calling, exzellent für komplexe Agent-Workflows.",
		BestFor:     []string{"agent", "chat", "reasoning"},
	},
	{
		Name:        "qwen2.5:72b",
		Tier:        "enterprise",
		SizeGB:      47,
		ToolCalling: true,
		Description: "Alibaba Qwen 2.5 72B — sehr starkes Tool-Calling, mehrsprachig (inkl. Deutsch).",
		BestFor:     []string{"agent", "chat", "coding"},
	},
	{
		Name:        "qwq:32b",
		Tier:        "enterprise",
		SizeGB:      20,
		ToolCalling: true,
		Reasoning:   true,
		Description: "Qwen QwQ 32B — starkes Reasoning-Modell, gut für tiefere Diagnosen.",
		BestFor:     []string{"agent", "reasoning"},
	},

	// --- pro tier (>= 24 GB available memory) ---
	{
		Name:        "qwen2.5:32b",
		Tier:        "pro",
		SizeGB:      20,
		ToolCalling: true,
		Description: "Qwen 2.5 32B — stabiles Tool-Calling, sehr gute Default-Wahl auf großer Hardware.",
		BestFor:     []string{"agent", "chat", "coding"},
	},
	{
		Name:        "llama3.3:70b-instruct-q4_K_M",
		Tier:        "pro",
		SizeGB:      43,
		ToolCalling: true,
		Description: "Llama 3.3 70B 4-Bit-quantisiert — Top-Modell mit kleinerem Speicherbedarf.",
		BestFor:     []string{"agent", "reasoning"},
	},
	{
		Name:        "qwen2.5-coder:32b",
		Tier:        "pro",
		SizeGB:      20,
		ToolCalling: true,
		Description: "Qwen 2.5 Coder 32B — Tool-Calling mit Coding-Fokus, ideal für SSH-/Skript-Aktionen.",
		BestFor:     []string{"agent", "coding"},
	},

	// --- standard tier (>= 12 GB available memory) ---
	{
		Name:        "qwen2.5:14b",
		Tier:        "standard",
		SizeGB:      9,
		ToolCalling: true,
		Description: "Qwen 2.5 14B — klare Empfehlung für mittlere Hardware. Solides Tool-Calling.",
		BestFor:     []string{"agent", "chat"},
	},
	{
		Name:        "mistral-small:24b",
		Tier:        "standard",
		SizeGB:      14,
		ToolCalling: true,
		Description: "Mistral Small 24B — starkes Tool-Calling, gut auf europäischen Sprachen.",
		BestFor:     []string{"agent", "chat"},
	},

	// --- light tier (>= 6 GB available memory) ---
	{
		Name:        "llama3.1:8b",
		Tier:        "light",
		SizeGB:      4.7,
		ToolCalling: true,
		Description: "Meta Llama 3.1 8B — Mindeststandard für agentisches Verhalten. Klein, ausreichend, läuft auf den meisten Maschinen.",
		BestFor:     []string{"agent", "chat"},
	},
	{
		Name:        "qwen2.5:7b",
		Tier:        "light",
		SizeGB:      4.7,
		ToolCalling: true,
		Description: "Qwen 2.5 7B — solides Tool-Calling auf kleiner Hardware.",
		BestFor:     []string{"agent", "chat"},
	},

	// --- minimal tier (< 6 GB) — Tool-Calling unzuverlässig ---
	{
		Name:        "llama3.2:3b",
		Tier:        "minimal",
		SizeGB:      2,
		ToolCalling: true,
		Description: "Llama 3.2 3B — nur für sehr kleine Geräte. Tool-Calling funktioniert, aber Antworten sind oft simpel.",
		BestFor:     []string{"chat"},
	},
	{
		Name:        "qwen2.5:3b",
		Tier:        "minimal",
		SizeGB:      2,
		ToolCalling: true,
		Description: "Qwen 2.5 3B — leichtgewichtige Alternative für Edge-Hardware.",
		BestFor:     []string{"chat"},
	},
}

// tierThresholds maps the lower bound (GB) for each tier.
var tierThresholds = []struct {
	Min   float64
	Tier  string
	Label string
}{
	{Min: 64, Tier: "enterprise", Label: "Enterprise (≥ 64 GB)"},
	{Min: 24, Tier: "pro", Label: "Pro (≥ 24 GB)"},
	{Min: 12, Tier: "standard", Label: "Standard (≥ 12 GB)"},
	{Min: 6, Tier: "light", Label: "Light (≥ 6 GB)"},
	{Min: 0, Tier: "minimal", Label: "Minimal (< 6 GB)"},
}

// tierRank lets us include all lower tiers in the recommendation list.
var tierRank = map[string]int{
	"minimal":    0,
	"light":      1,
	"standard":   2,
	"pro":        3,
	"enterprise": 4,
}

// RecommendationService produces hardware-aware model suggestions.
type RecommendationService struct {
	ollama *llm.OllamaProvider
}

// NewRecommendationService builds a service against the active Ollama provider.
// The provider may be nil — the service still produces recommendations, just
// without "pulled" badges.
func NewRecommendationService(ollama *llm.OllamaProvider) *RecommendationService {
	return &RecommendationService{ollama: ollama}
}

// Build returns the full system + recommendation snapshot.
func (s *RecommendationService) Build(ctx context.Context) SystemRecommendation {
	spec := DetectSystem(ctx)
	pulled := s.installedModels(ctx)

	models := recommendForTier(spec.Tier, pulled)

	defaultModel := pickDefault(models)

	rec := SystemRecommendation{
		System:       spec,
		Models:       models,
		DefaultModel: defaultModel,
	}
	if s.ollama != nil {
		rec.OllamaURL = s.ollama.BaseURL()
		rec.OllamaReady = len(pulled) > 0
	}
	return rec
}

// DetectSystem reads /proc/meminfo, /proc/cpuinfo and probes nvidia-smi to
// build a SystemSpec. All sources are best-effort: missing data leaves a
// field at its zero value plus a note explaining why.
func DetectSystem(ctx context.Context) SystemSpec {
	spec := SystemSpec{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}

	if ramGB, err := readTotalRAMGB(); err == nil {
		spec.TotalRAMGB = ramGB
	} else {
		spec.Notes = append(spec.Notes, fmt.Sprintf("RAM-Erkennung: %v", err))
	}

	gpus, gpuNote := detectGPUs(ctx)
	spec.GPUs = gpus
	if gpuNote != "" {
		spec.Notes = append(spec.Notes, gpuNote)
	}

	tier, label := classifyTier(spec)
	spec.Tier = tier
	spec.TierLabel = label
	return spec
}

func readTotalRAMGB() (float64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, fmt.Errorf("/proc/meminfo nicht lesbar (kein Linux?)")
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, fmt.Errorf("MemTotal-Zeile unvollständig")
		}
		kb, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return 0, fmt.Errorf("MemTotal nicht parsbar: %w", err)
		}
		return kb / 1024.0 / 1024.0, nil
	}
	return 0, fmt.Errorf("MemTotal in /proc/meminfo nicht gefunden")
}

func detectGPUs(ctx context.Context) ([]GPUInfo, string) {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return nil, "nvidia-smi nicht gefunden — GPU-Erkennung übersprungen."
	}

	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "nvidia-smi",
		"--query-gpu=name,memory.total,driver_version",
		"--format=csv,noheader,nounits",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Sprintf("nvidia-smi schlug fehl: %v", err)
	}

	var gpus []GPUInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Split(line, ", ")
		if len(parts) < 2 {
			continue
		}
		mib, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		gpu := GPUInfo{
			Name:   strings.TrimSpace(parts[0]),
			VRAMGB: mib / 1024.0,
			Vendor: "nvidia",
		}
		if len(parts) >= 3 {
			gpu.Driver = strings.TrimSpace(parts[2])
		}
		gpus = append(gpus, gpu)
	}
	if len(gpus) == 0 {
		return nil, "nvidia-smi lieferte keine GPU zurück."
	}
	return gpus, ""
}

func classifyTier(spec SystemSpec) (string, string) {
	memGB := spec.TotalRAMGB
	for _, gpu := range spec.GPUs {
		if gpu.VRAMGB > memGB {
			memGB = gpu.VRAMGB
		}
	}
	for _, t := range tierThresholds {
		if memGB >= t.Min {
			return t.Tier, t.Label
		}
	}
	return "minimal", "Minimal (< 6 GB)"
}

func recommendForTier(tier string, pulled map[string]bool) []ModelRecommendation {
	rank, ok := tierRank[tier]
	if !ok {
		rank = tierRank["light"]
	}
	out := make([]ModelRecommendation, 0, len(modelCatalog))
	for _, m := range modelCatalog {
		modelRank, ok := tierRank[m.Tier]
		if !ok {
			continue
		}
		if modelRank > rank {
			continue
		}
		entry := m
		entry.Pulled = pulled[stripTagSuffix(entry.Name)] || pulled[entry.Name]
		out = append(out, entry)
	}

	// Sort: same-tier first (descending capability), then lower tiers.
	sort.SliceStable(out, func(i, j int) bool {
		ri := tierRank[out[i].Tier]
		rj := tierRank[out[j].Tier]
		if ri != rj {
			return ri > rj
		}
		// Within a tier: pulled models first (instant use), then by size desc.
		if out[i].Pulled != out[j].Pulled {
			return out[i].Pulled
		}
		return out[i].SizeGB > out[j].SizeGB
	})

	// Mark recommended (first match for current tier) and default.
	for i := range out {
		if out[i].Tier == tier {
			out[i].Recommended = true
			break
		}
	}
	if len(out) > 0 {
		out[0].Default = true
	}
	return out
}

func pickDefault(models []ModelRecommendation) string {
	for _, m := range models {
		if m.Default {
			return m.Name
		}
	}
	if len(models) > 0 {
		return models[0].Name
	}
	return "llama3.1:8b"
}

// installedModels asks Ollama for /api/tags and returns a set of installed
// model names (both with and without the size tag suffix).
func (s *RecommendationService) installedModels(ctx context.Context) map[string]bool {
	if s.ollama == nil {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	models, err := s.ollama.DiscoverModels(cctx)
	if err != nil {
		return nil
	}
	out := make(map[string]bool, len(models)*2)
	for _, m := range models {
		out[m.Name] = true
		out[stripTagSuffix(m.Name)] = true
	}
	return out
}

func stripTagSuffix(name string) string {
	if idx := strings.IndexByte(name, ':'); idx > 0 {
		return name[:idx]
	}
	return name
}

// PullModel asks the Ollama server to pull the given model. The request blocks
// until completion; for large models this may take several minutes — callers
// should run it in a goroutine and surface progress separately.
func (s *RecommendationService) PullModel(ctx context.Context, name string) error {
	if s.ollama == nil {
		return fmt.Errorf("Ollama-Provider nicht verfügbar")
	}
	body, _ := json.Marshal(map[string]any{
		"name":   name,
		"stream": false,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(s.ollama.BaseURL(), "/")+"/api/pull",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama-Pull fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("Ollama-Pull lieferte Status %d", resp.StatusCode)
	}
	return nil
}
