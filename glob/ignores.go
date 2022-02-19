package glob

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var defaultIgnores = []string{
	".hg",
	".git",
	".gitmodules",
	".svn",
	".cache",
	".next",
	".now",
	".vercel",
	".npmignore",
	".dockerignore",
	".gitignore",
	".*.swp",
	".DS_Store",
	".wafpicke-*",
	".lock-wscript",
	".env.local",
	".env.*.local",
	".venv",
	"npm-debug.log",
	"config.gypi",
	"node_modules",
	"__pycache__",
	"venv",
	"CVS",
	".vercel_build_output",
	".terraform*",
}

// GetIgnores is used to parse a .vercelignore file from a given directory, and
// combine the expected results with a default set of ignored files.
func GetIgnores(path string) ([]string, error) {
	ignoreFilePath := filepath.Join(path, ".vercelignore")
	ignoreFile, err := os.ReadFile(ignoreFilePath)
	if errors.Is(err, fs.ErrNotExist) {
		return defaultIgnores, nil
	}
	if err != nil {
		return nil, fmt.Errorf("unable to read .vercelignore file: %w", err)
	}

	var ignores []string
	sc := bufio.NewScanner(strings.NewReader(string(ignoreFile)))
	for sc.Scan() {
		ignores = append(ignores, sc.Text())
	}

	ignores = append(ignores, defaultIgnores...)
	return ignores, nil
}
