package file

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestGetPaths(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files          []string
		ignorePatterns []string
		expected       []string
	}{
		"allowlist patterns": {
			files: []string{
				".vercelignore",
				"api/example.js",
				"example",
				"index.html",
				"main.tf",
				"providers.tf",
				"vercel.json",
				"versions.tf",
			},
			ignorePatterns: []string{"/*", "!api", "!vercel.json", "!*.html"},
			expected:       []string{"api/example.js", "index.html", "vercel.json"},
		},
		"root anchored pattern": {
			files:          []string{"root.txt", "nested/root.txt", "nested/other.txt"},
			ignorePatterns: []string{"/root.txt"},
			expected:       []string{"nested/other.txt", "nested/root.txt"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			workingDirectory, err := filepath.Abs(".")
			if err != nil {
				t.Fatal(err)
			}

			for _, name := range test.files {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			relativeRoot, err := filepath.Rel(workingDirectory, root)
			if err != nil {
				t.Fatal(err)
			}

			for _, basePath := range []string{root, relativeRoot} {
				paths, err := GetPaths(basePath, test.ignorePatterns)
				if err != nil {
					t.Fatal(err)
				}

				actual := make([]string, 0, len(paths))
				for _, path := range paths {
					relativePath, err := filepath.Rel(basePath, path)
					if err != nil {
						t.Fatal(err)
					}
					actual = append(actual, filepath.ToSlash(relativePath))
				}
				slices.Sort(actual)

				if !slices.Equal(actual, test.expected) {
					t.Fatalf("GetPaths(%q) = %q, want %q", basePath, actual, test.expected)
				}
			}
		})
	}
}
