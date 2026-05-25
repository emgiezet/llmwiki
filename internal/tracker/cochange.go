package tracker

import (
	"sort"
	"strings"
)

const coChangeThreshold = 0.3
const minCommitsForClustering = 20

// Clusterer groups files into Areas based on git co-change history.
type Clusterer struct {
	runner    GitRunner
	threshold float64
}

// NewClusterer returns a Clusterer with the default co-change threshold.
func NewClusterer(runner GitRunner) *Clusterer {
	return &Clusterer{runner: runner, threshold: coChangeThreshold}
}

// Cluster returns Areas for projectRoot.
// When len(commits) < minCommitsForClustering, uses fallbackAreas(fallbackFiles).
func (c *Clusterer) Cluster(projectRoot string, fallbackFiles []string) ([]Area, error) {
	commits, err := c.runner.LogFiles(projectRoot)
	if err != nil {
		return nil, err
	}
	if len(commits) < minCommitsForClustering {
		return fallbackAreas(fallbackFiles), nil
	}
	return c.buildAreas(commits), nil
}

// buildAreas uses union-find on co-occurrence data to group files into Areas.
func (c *Clusterer) buildAreas(commits []CommitFiles) []Area {
	// Step 1: build file counts and pair counts.
	fileCounts := make(map[string]int)
	pairCounts := make(map[[2]string]int)

	for _, commit := range commits {
		for _, f := range commit.Files {
			fileCounts[f]++
		}
		// All pairs in this commit.
		for i := 0; i < len(commit.Files); i++ {
			for j := i + 1; j < len(commit.Files); j++ {
				a, b := commit.Files[i], commit.Files[j]
				if a > b {
					a, b = b, a
				}
				pairCounts[[2]string{a, b}]++
			}
		}
	}

	// Step 2: union-find with path compression.
	parent := make(map[string]string)

	var find func(x string) string
	find = func(x string) string {
		if _, ok := parent[x]; !ok {
			parent[x] = x
		}
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	union := func(x, y string) {
		rx, ry := find(x), find(y)
		if rx != ry {
			parent[rx] = ry
		}
	}

	// Ensure all files appear in union-find.
	for f := range fileCounts {
		find(f)
	}

	// Union files whose co-change ratio >= threshold.
	for pair, count := range pairCounts {
		a, b := pair[0], pair[1]
		countA, countB := fileCounts[a], fileCounts[b]
		minCount := countA
		if countB < minCount {
			minCount = countB
		}
		if minCount > 0 && float64(count)/float64(minCount) >= c.threshold {
			union(a, b)
		}
	}

	// Step 3: group files by their find-root.
	groups := make(map[string][]string)
	for f := range fileCounts {
		root := find(f)
		groups[root] = append(groups[root], f)
	}

	// Step 4: build Areas.
	areas := make([]Area, 0, len(groups))
	for _, files := range groups {
		sort.Strings(files)
		areas = append(areas, Area{
			Name:          areaName(files),
			Files:         files,
			ClusterMethod: "git-cochange",
		})
	}

	// Step 5: sort areas by Name.
	sort.Slice(areas, func(i, j int) bool {
		return areas[i].Name < areas[j].Name
	})

	return areas
}

// fallbackAreas groups files by their top-level directory component.
func fallbackAreas(files []string) []Area {
	groups := make(map[string][]string)
	for _, f := range files {
		topDir := strings.SplitN(f, "/", 2)[0]
		groups[topDir] = append(groups[topDir], f)
	}

	areas := make([]Area, 0, len(groups))
	for dir, groupFiles := range groups {
		sort.Strings(groupFiles)
		areas = append(areas, Area{
			Name:          dir,
			Files:         groupFiles,
			ClusterMethod: "scanner-heuristic",
		})
	}

	sort.Slice(areas, func(i, j int) bool {
		return areas[i].Name < areas[j].Name
	})

	return areas
}
