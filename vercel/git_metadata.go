package vercel

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// findRepoRoot traverses upward from startDir until it finds either
// - .vercel/repo.json (treat as repo root), OR
// - .git/config (or a .git file for worktrees/submodules)
// Returns the directory path or empty string if not found.
func findRepoRoot(startDir string) string {
	curr := startDir
	for {
		if curr == "/" || curr == "." || len(curr) == 0 {
			return ""
		}

		// .vercel/repo.json
		if fileExists(filepath.Join(curr, ".vercel", "repo.json")) || fileExists(filepath.Join(curr, ".now", "repo.json")) {
			return curr
		}
		// .git/config directory or .git file
		if fileExists(filepath.Join(curr, ".git", "config")) || fileExists(filepath.Join(curr, ".git")) {
			return curr
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			return ""
		}
		curr = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0 || info.IsDir() || info.Mode().IsRegular()
}

// runGit executes a git command in cwd with the given timeout.
func runGit(ctx context.Context, cwd string, timeout time.Duration, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, "git", args...)
	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if cctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("git %v timed out", args)
	}
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("git %v failed: %s", args, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

// collectGitMetadata collects Git metadata from repoRoot, preferring a remote url that contains
// linkRepo (e.g. "org/repo") when provided.
func collectGitMetadata(ctx context.Context, repoRoot string, linkRepo string) (*client.GitMetadata, error) {
	if repoRoot == "" {
		return nil, nil
	}
	// Determine remote URL(s)
	remoteURL := ""
	remotesOut, err := runGit(ctx, repoRoot, 2*time.Second, "remote", "-v")
	if err == nil && remotesOut != "" {
		lines := strings.Split(remotesOut, "\n")
		remoteMap := map[string]string{}
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[0]
				url := parts[1]
				// Only set if not set yet to prefer first occurrence
				if _, ok := remoteMap[name]; !ok {
					remoteMap[name] = url
				}
			}
		}
		if linkRepo != "" {
			for _, url := range remoteMap {
				if strings.Contains(url, linkRepo) {
					remoteURL = url
					break
				}
			}
		}
		if remoteURL == "" {
			if u, ok := remoteMap["origin"]; ok {
				remoteURL = u
			} else {
				// pick any deterministic remote (alphabetical)
				keys := make([]string, 0, len(remoteMap))
				for k := range remoteMap {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				if len(keys) > 0 {
					remoteURL = remoteMap[keys[0]]
				}
			}
		}
	}
	if remoteURL == "" {
		// Fallback to git config get origin
		if u, err := runGit(ctx, repoRoot, 2*time.Second, "config", "--get", "remote.origin.url"); err == nil {
			remoteURL = u
		}
	}

	// Get last commit info
	an, err1 := runGit(ctx, repoRoot, 2*time.Second, "log", "-1", "--pretty=%an")
	ae, err2 := runGit(ctx, repoRoot, 2*time.Second, "log", "-1", "--pretty=%ae")
	sub, err3 := runGit(ctx, repoRoot, 2*time.Second, "log", "-1", "--pretty=%s")
	branch, err4 := runGit(ctx, repoRoot, 2*time.Second, "rev-parse", "--abbrev-ref", "HEAD")
	sha, err5 := runGit(ctx, repoRoot, 2*time.Second, "rev-parse", "HEAD")

	// Dirty state
	dirtyOut, err6 := runGit(ctx, repoRoot, 2*time.Second, "--no-optional-locks", "status", "-s")
	if err6 != nil {
		// fallback without flag
		dirtyOut, _ = runGit(ctx, repoRoot, 2*time.Second, "status", "-s")
	}

	// If commit info or dirty check failed entirely, skip metadata (mirror CLI best-effort)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
		return nil, nil
	}

	gm := &client.GitMetadata{
		CommitAuthorName:  an,
		CommitAuthorEmail: ae,
		CommitMessage:     sub,
		CommitRef:         branch,
		CommitSha:         sha,
		Dirty:             strings.TrimSpace(dirtyOut) != "",
	}
	if remoteURL != "" {
		gm.RemoteUrl = remoteURL
	}
	return gm, nil
}

// startingDirsFromFiles returns candidate starting directories ordered by shallowness
// (fewest path separators). It resolves absolute path and symlinks.
func startingDirsFromFiles(files []client.DeploymentFile) []string {
	type cand struct {
		dir   string
		depth int
	}
	seen := map[string]struct{}{}
	cands := []cand{}
	for _, f := range files {
		if f.File == "" { // skip
			continue
		}
		abs, err := filepath.Abs(f.File)
		if err != nil {
			continue
		}
		abs, _ = filepath.EvalSymlinks(abs)
		dir := filepath.Dir(abs)
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		depth := strings.Count(filepath.ToSlash(dir), "/")
		cands = append(cands, cand{dir: dir, depth: depth})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].depth < cands[j].depth })
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		out = append(out, c.dir)
	}
	return out
}

// detectRepoRootFromFiles picks a small number of shallowest candidates and tries to
// discover a repo root by traversing upwards from each.
func detectRepoRootFromFiles(files []client.DeploymentFile) string {
	cands := startingDirsFromFiles(files)
	limit := min(len(cands), 5)
	for i := range limit {
		root := findRepoRoot(cands[i])
		if root != "" {
			return root
		}
	}
	return ""
}

// getLinkRepo returns the "org/repo" string for a linked project when available.
func getLinkRepo(pr client.ProjectResponse) string {
	if pr.Link == nil {
		return ""
	}
	switch pr.Link.Type {
	case "github":
		return fmt.Sprintf("%s/%s", pr.Link.Org, pr.Link.Repo)
	case "gitlab":
		if pr.Link.ProjectNamespace != "" && pr.Link.ProjectURL != "" {
			// For GitLab the CLI matches by org/repo; construct a best-effort "namespace/repo"
			// repo name inferred from URL is handled elsewhere in client.Repository(), but here
			// a simple contains match on namespace is still useful, so return namespace.
			return pr.Link.ProjectNamespace
		}
	case "bitbucket":
		if pr.Link.Owner != "" && pr.Link.Slug != "" {
			return fmt.Sprintf("%s/%s", pr.Link.Owner, pr.Link.Slug)
		}
	}
	return ""
}

// prepareGitMetadata best-effort detects repo root and collects git metadata.
func prepareGitMetadata(ctx context.Context, files []client.DeploymentFile, _ string, project client.ProjectResponse) *client.GitMetadata {
	var startRoot string
	if len(files) > 0 {
		startRoot = detectRepoRootFromFiles(files)
	}
	if startRoot == "" {
		// Ref-only deployments or failed detection: try from current working dir
		cwd, _ := os.Getwd()
		startRoot = findRepoRoot(cwd)
	}
	if startRoot == "" {
		return nil
	}
	linkRepo := getLinkRepo(project)
	gm, err := collectGitMetadata(ctx, startRoot, linkRepo)
	if err != nil {
		// Log and proceed without metadata
		tflog.Debug(ctx, fmt.Sprintf("skipping git metadata: %v", err))
		return nil
	}
	if gm != nil {
		short := gm.CommitSha
		if len(short) > 7 {
			short = short[:7]
		}
		tflog.Debug(ctx, fmt.Sprintf("attached git metadata (sha=%s, ref=%s, dirty=%t)", short, gm.CommitRef, gm.Dirty))
	}
	return gm
}
