package backup

import (
	"fmt"
	"strings"
)

// FileDiff represents the comparison result of a single file between two
// backup snapshots.
type FileDiff struct {
	FilePath string `json:"file_path"`
	Status   string `json:"status"` // "added", "removed", "modified", "unchanged"
	Diff     string `json:"diff,omitempty"`
}

// DiffFiles compares two sets of collected files and returns a diff for each
// file. Files present only in newFiles are marked "added", files only in
// oldFiles are "removed", and files in both are compared by hash to determine
// "unchanged" or "modified".
func DiffFiles(oldFiles, newFiles []CollectedFile) []FileDiff {
	oldMap := make(map[string]CollectedFile, len(oldFiles))
	for _, f := range oldFiles {
		oldMap[f.Path] = f
	}

	newMap := make(map[string]CollectedFile, len(newFiles))
	for _, f := range newFiles {
		newMap[f.Path] = f
	}

	var diffs []FileDiff

	// Check files in new set
	for path, newFile := range newMap {
		oldFile, exists := oldMap[path]
		if !exists {
			diffs = append(diffs, FileDiff{
				FilePath: path,
				Status:   "added",
			})
			continue
		}

		if oldFile.Hash == newFile.Hash {
			diffs = append(diffs, FileDiff{
				FilePath: path,
				Status:   "unchanged",
			})
		} else {
			diff := generateUnifiedDiff(
				string(oldFile.Content),
				string(newFile.Content),
			)
			diffs = append(diffs, FileDiff{
				FilePath: path,
				Status:   "modified",
				Diff:     diff,
			})
		}
	}

	// Check for removed files (in old but not in new)
	for path := range oldMap {
		if _, exists := newMap[path]; !exists {
			diffs = append(diffs, FileDiff{
				FilePath: path,
				Status:   "removed",
			})
		}
	}

	return diffs
}

// generateUnifiedDiff produces a simple unified-diff-style output showing
// lines removed (-) and lines added (+) between two text contents.
func generateUnifiedDiff(oldContent, newContent string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Build a simple LCS-based diff
	oldLen := len(oldLines)
	newLen := len(newLines)

	// Create a matrix for longest common subsequence
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

	// Backtrack to produce the diff
	var diffLines []string
	i, j := oldLen, newLen
	var stack []string

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

	// Reverse the stack to get correct order
	for k := len(stack) - 1; k >= 0; k-- {
		diffLines = append(diffLines, stack[k])
	}

	if len(diffLines) == 0 {
		return ""
	}

	return fmt.Sprintf("--- a\n+++ b\n%s", strings.Join(diffLines, "\n"))
}
