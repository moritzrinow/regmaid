package regmaid

import (
	"bufio"
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/regclient/regclient/types/ref"
	str2duration "github.com/xhit/go-str2duration/v2"
)

var (
	Yes    bool
	DryRun bool
)

type PolicyResult struct {
	Error      error
	Policy     *Policy
	Registry   *Registry
	Manifests  []Manifest
	Repository string
	TotalTags  int
}

func ExecuteClean(ctx context.Context) error {
	cfg, err := LoadConfig(ConfigPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	maid, err := New(cfg)
	if err != nil {
		return err
	}

	var lock sync.Mutex
	var wg sync.WaitGroup
	var results []*PolicyResult

	// Process policies
	for _, policy := range cfg.Policies {

		reg := getRegistry(cfg, &policy)

		var retention time.Duration

		if policy.Retention != "" {
			duration, err := str2duration.ParseDuration(policy.Retention)
			if err == nil {
				retention = duration
			}
		}

		fmt.Printf("Processing policy %q...\n", policy.Name)

		repos, err := maid.GetRepositories(ctx, reg.Host, policy.Repository)
		if err != nil {
			fmt.Printf("Error getting repositories for policy %q: %v\n", policy.Name, err)
			return err
		}

		for _, repoName := range repos {
			wg.Add(1)

			go func(repoName string) {
				defer wg.Done()

				fmt.Printf("Processing repository %q\n", repoName)

				repo, _ := ref.New(fmt.Sprintf("%s/%s", reg.Host, repoName))

				total, manifests, err := maid.ScanRepository(ctx, repo.CommonName(), policy.Match, policy.Regex)

				result := &PolicyResult{
					TotalTags:  total,
					Policy:     &policy,
					Registry:   reg,
					Manifests:  manifests,
					Repository: repoName,
				}

				if err != nil {
					result.Error = err
				}

				// Sort tags by age (ascending)
				slices.SortFunc(result.Manifests, func(a, b Manifest) int {
					return cmp.Compare(a.Age, b.Age)
				})

				// Always keep N newest tags
				keep := min(result.Policy.Keep, len(result.Manifests))
				result.Manifests = result.Manifests[keep:]

				filtered := []Manifest{}

				// Filter for tags after retention period
				for _, m := range result.Manifests {
					if m.Age >= retention {
						filtered = append(filtered, m)
					}
				}

				result.Manifests = filtered

				lock.Lock()

				results = append(results, result)

				lock.Unlock()
			}(repoName)
		}

		fmt.Printf("Finished processing policy %q\n", policy.Name)
	}

	// Wait for all policies to finish processing
	wg.Wait()

	fail := false

	// Check for errors
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("Error processing policy %q: %v\n", result.Policy.Name, result.Error)

			fail = true

			continue
		}
	}

	if fail {
		return fmt.Errorf("One or more policies finished with an error, therefore exiting.")
	}

	total := 0

	for _, result := range results {
		if len(result.Manifests) > 0 {
			total += len(result.Manifests)

			fmt.Printf("Policy %q found %d/%d tags eligible for deletion:\n", result.Policy.Name, len(result.Manifests), result.TotalTags)

			for _, m := range result.Manifests {
				fmt.Printf("%s:%s (%s) (%dd)\n", result.Repository, m.Tag, m.Digest, int(m.Age.Hours()/24))
			}
		} else {

			fmt.Printf("Policy %q found 0/%d tags eligible for deletion.\n", result.Policy.Name, result.TotalTags)
		}
	}

	fmt.Printf("Total tags to be deleted: %d\n", total)

	if DryRun {
		return nil
	}

	if total == 0 {
		fmt.Printf("No policies found any tags eligible for deletion, therefore exiting.\n")

		return nil
	}

	if !Yes {
		scanner := bufio.NewScanner(os.Stdin)

		fmt.Printf("Enter 'yes' to confirm deletion: ")

		scanner.Scan()

		if err := scanner.Err(); err != nil {
			return err
		}

		input := scanner.Text()

		if !strings.EqualFold(input, "yes") {
			fmt.Println("Cancelled deletion.")
			return nil
		}
	}

	for _, result := range results {
		wg.Add(1)

		go func() {
			defer wg.Done()

			reg := getRegistry(cfg, result.Policy)

			repo, _ := ref.New(fmt.Sprintf("%s/%s", reg.Host, result.Repository))

			digestsMap := make(map[string]any)

			for _, m := range result.Manifests {
				if _, ok := digestsMap[m.Digest]; !ok {
					digestsMap[m.Digest] = nil
				}
			}

			digests := make([]string, 0)

			for d := range digestsMap {
				digests = append(digests, d)
			}

			err := maid.DeleteManifests(ctx, repo.CommonName(), digests)
			if err != nil {
				fmt.Printf("Error deleting manifests: %v\n", err)
			}
		}()

	}

	wg.Wait()

	fmt.Printf("Finished.\n")

	return nil
}

// We assume there are no errors since the config was validated before
func getRegistry(cfg *Config, policy *Policy) *Registry {
	regIdx := slices.IndexFunc(cfg.Registries, func(reg Registry) bool {
		return reg.Name == policy.Registry
	})

	if regIdx < 0 {
		return nil
	}

	return &cfg.Registries[regIdx]
}
