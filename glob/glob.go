package glob

import (
	"fmt"
	"io/fs"
	"path/filepath"

	gitignore "github.com/sabhiram/go-gitignore"
)

func GetPaths(basePath string, ignorePatterns []string) ([]string, error) {
	ignore := gitignore.CompileIgnoreLines(ignorePatterns...)

	var paths []string
	err := filepath.WalkDir(
		basePath,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			ignored := ignore.MatchesPath(path)

			if d.IsDir() && ignored {
				return filepath.SkipDir
			}
			if ignored || d.IsDir() {
				return nil
			}

			paths = append(paths, path)
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error finding paths: %w", err)
	}

	return paths, nil
}
