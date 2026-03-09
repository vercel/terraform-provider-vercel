package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

type discoveredTest struct {
	Package string
	Name    string
}

type shard struct {
	Name      string `json:"name"`
	Package   string `json:"package"`
	Pattern   string `json:"pattern"`
	TestCount int    `json:"test_count"`
}

type matrix struct {
	Include []shard `json:"include"`
}

func main() {
	var (
		format        = flag.String("format", "matrix", "output format: matrix or summary")
		generalShards = flag.Int("general-shards", 4, "maximum number of general-purpose shards per package")
	)
	flag.Parse()

	if *generalShards < 1 {
		fmt.Fprintln(os.Stderr, "general-shards must be at least 1")
		os.Exit(1)
	}

	tests, err := discoverAcceptanceTests()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	shards, err := planShards(tests, *generalShards)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	switch *format {
	case "matrix":
		if err := json.NewEncoder(os.Stdout).Encode(matrix{Include: shards}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "summary":
		for _, shard := range shards {
			fmt.Printf("%s\t%s\t%d tests\n", shard.Name, shard.Package, shard.TestCount)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown format %q\n", *format)
		os.Exit(1)
	}
}

func discoverAcceptanceTests() ([]discoveredTest, error) {
	packages, err := listPackages()
	if err != nil {
		return nil, err
	}

	var tests []discoveredTest
	for _, pkg := range packages {
		pkgTests, err := listPackageAcceptanceTests(pkg)
		if err != nil {
			return nil, err
		}
		tests = append(tests, pkgTests...)
	}

	if len(tests) == 0 {
		return nil, fmt.Errorf("no acceptance tests discovered")
	}

	sort.Slice(tests, func(i, j int) bool {
		if tests[i].Package == tests[j].Package {
			return tests[i].Name < tests[j].Name
		}
		return tests[i].Package < tests[j].Package
	})

	return tests, nil
}

func listPackages() ([]string, error) {
	cmd := exec.Command("go", "list", "./...")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}

	var packages []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		packages = append(packages, line)
	}
	return packages, nil
}

func listPackageAcceptanceTests(pkg string) ([]discoveredTest, error) {
	cmd := exec.Command("go", "test", pkg, "-list", "^TestAcc_")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("list acceptance tests for %s: %w\n%s", pkg, err, stderr.String())
	}

	var tests []discoveredTest
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "TestAcc_") {
			continue
		}
		tests = append(tests, discoveredTest{
			Package: pkg,
			Name:    line,
		})
	}

	return tests, nil
}

func planShards(tests []discoveredTest, maxGeneralShards int) ([]shard, error) {
	testsByPackage := make(map[string][]string)
	for _, test := range tests {
		testsByPackage[test.Package] = append(testsByPackage[test.Package], test.Name)
	}

	var packages []string
	for pkg := range testsByPackage {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	var planned []shard
	assigned := make(map[string]int, len(tests))
	for _, pkg := range packages {
		pkgTests := append([]string(nil), testsByPackage[pkg]...)
		sort.Strings(pkgTests)

		var sensitive []string
		var dedicated []string
		var general []string
		for _, test := range pkgTests {
			if isSensitiveTest(pkg, test) {
				sensitive = append(sensitive, test)
				continue
			}
			if isDedicatedSlowTest(pkg, test) {
				dedicated = append(dedicated, test)
				continue
			}
			general = append(general, test)
		}

		slug := packageSlug(pkg)
		if len(sensitive) > 0 {
			planned = append(planned, shard{
				Name:      slug + "-sensitive",
				Package:   pkg,
				Pattern:   exactMatchPattern(sensitive),
				TestCount: len(sensitive),
			})
			for _, test := range sensitive {
				assigned[testKey(pkg, test)]++
			}
		}

		for _, test := range dedicated {
			planned = append(planned, shard{
				Name:      dedicatedShardName(slug, test),
				Package:   pkg,
				Pattern:   exactMatchPattern([]string{test}),
				TestCount: 1,
			})
			assigned[testKey(pkg, test)]++
		}

		generalBucketCount := maxGeneralShards
		if len(general) < generalBucketCount {
			generalBucketCount = len(general)
		}
		if generalBucketCount == 0 {
			continue
		}

		buckets := make([][]string, generalBucketCount)
		for i, test := range general {
			buckets[i%generalBucketCount] = append(buckets[i%generalBucketCount], test)
			assigned[testKey(pkg, test)]++
		}

		for i, bucket := range buckets {
			if len(bucket) == 0 {
				continue
			}
			planned = append(planned, shard{
				Name:      fmt.Sprintf("%s-%d", slug, i+1),
				Package:   pkg,
				Pattern:   exactMatchPattern(bucket),
				TestCount: len(bucket),
			})
		}
	}

	if err := verifyAssignments(tests, assigned); err != nil {
		return nil, err
	}

	return planned, nil
}

func verifyAssignments(tests []discoveredTest, assigned map[string]int) error {
	for _, test := range tests {
		key := testKey(test.Package, test.Name)
		switch assigned[key] {
		case 1:
		case 0:
			return fmt.Errorf("acceptance test %s in %s was not assigned to any shard", test.Name, test.Package)
		default:
			return fmt.Errorf("acceptance test %s in %s was assigned to %d shards", test.Name, test.Package, assigned[key])
		}
	}
	return nil
}

func isSensitiveTest(pkg, test string) bool {
	if !strings.HasSuffix(pkg, "/vercel") {
		return false
	}

	switch test {
	case "TestAcc_SharedEnvironmentVariableProjectLink",
		"TestAcc_TeamConfig",
		"TestAcc_TeamConfigDataSource",
		"TestAcc_TeamMemberDataSource",
		"TestAcc_TeamMemberResource":
		return true
	default:
		return false
	}
}

func isDedicatedSlowTest(pkg, test string) bool {
	if !strings.HasSuffix(pkg, "/vercel") {
		return false
	}

	switch test {
	case "TestAcc_NetworkResource":
		return true
	default:
		return false
	}
}

func dedicatedShardName(pkgSlug, test string) string {
	switch test {
	case "TestAcc_NetworkResource":
		return pkgSlug + "-network"
	default:
		return pkgSlug + "-slow"
	}
}

func packageSlug(pkg string) string {
	slug := pkg[strings.LastIndex(pkg, "/")+1:]
	if slug == "" {
		slug = "root"
	}
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug = re.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}

func exactMatchPattern(tests []string) string {
	escaped := make([]string, 0, len(tests))
	for _, test := range tests {
		escaped = append(escaped, regexp.QuoteMeta(test))
	}
	return fmt.Sprintf("^(%s)$", strings.Join(escaped, "|"))
}

func testKey(pkg, test string) string {
	return pkg + "\x00" + test
}
