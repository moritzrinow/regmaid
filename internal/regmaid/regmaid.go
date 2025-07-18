package regmaid

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/scheme"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
)

type RegMaid struct {
	client *regclient.RegClient
}

type Manifest struct {
	Digest string
	Tags   []string
	Age    time.Duration
}

type TagFilter struct {
	Match  string
	MinAge time.Duration
}

func New(c *Config) (*RegMaid, error) {
	client := NewRegistryClient(c)

	return &RegMaid{
		client: client,
	}, nil
}

func (r *RegMaid) DeleteManifests(ctx context.Context, repo string, digests []string) error {
	repoRef, err := ref.New(repo)
	if err != nil {

		return err
	}

	var wg sync.WaitGroup

	for _, d := range digests {
		wg.Add(1)

		go func() {
			defer wg.Done()
			manifestRef := repoRef.SetDigest(d)

			err := r.client.ManifestDelete(ctx, manifestRef)
			if err != nil {
				fmt.Printf("Error deleting manifest %s: %v\n", manifestRef.CommonName(), err)

				return
			}

			fmt.Printf("Deleted manifest %s\n", manifestRef.CommonName())
		}()
	}

	wg.Wait()

	return nil
}

// Get a list of all repositories matching with the specified match.
func (r *RegMaid) GetRepositories(ctx context.Context, host string, match string) ([]string, error) {
	// If the repository name is an exact match and not a wildcard expression, we short circuit to avoid calling the _catalog API
	if !strings.Contains(match, "*") {
		return []string{match}, nil
	}

	rl, err := r.client.RepoList(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("error listing repositories for host %s: %v", host, err)
	}

	repos, err := rl.GetRepos()
	if err != nil {
		return nil, fmt.Errorf("error extracting repositories for host %s: %v", host, err)
	}

	regex, err := getRegex(match, false)
	if err != nil {
		return nil, err
	}

	filteredRepos := make([]string, 0)
	for _, repo := range repos {
		if regex.MatchString(repo) {
			filteredRepos = append(filteredRepos, repo)
		}
	}

	return filteredRepos, nil
}

// Scans all tagged manifests of the repository and returns the ones matching the specified filter.
func (r *RegMaid) ScanRepository(ctx context.Context, repo string, match string, isRegex bool) (int, []Manifest, error) {
	repoRef, err := ref.New(repo)
	if err != nil {

		return 0, nil, err
	}

	regex, err := getRegex(match, isRegex)
	if err != nil {
		return 0, nil, err
	}

	// Cancels pending manifest retrievals running in the background
	backgroundCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	tags := make([]Manifest, 0)

	tc := make(chan Manifest)

	var wg sync.WaitGroup

	var opts []scheme.TagOpts

	tagList, err := r.client.TagList(ctx, repoRef, opts...)

	if err != nil {
		return 0, nil, err
	}

	totalTags := len(tagList.Tags)

	for _, t := range tagList.Tags {
		if regex.MatchString(t) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tagRef := repoRef.SetTag(t)

				m, err := r.client.ManifestGet(backgroundCtx, tagRef)

				if err != nil {
					fmt.Printf("Error retrieving manifest for tag %s: %v\n", tagRef.CommonName(), err)

					return
				}

				if m.IsList() {
					// Not supported
				} else {
					imager := m.(manifest.Imager)

					imgConf, _ := imager.GetConfig()

					ociConfig, err := r.client.BlobGetOCIConfig(ctx, repoRef, imgConf)

					if err != nil {
						fmt.Printf("Unable to read OCI config blob for %s: %v\n", m.GetRef().CommonName(), err)

						return
					}

					created := ociConfig.GetConfig().Created
					if created == nil {
						fmt.Printf("Unable to determine age of manifest %s due to missing 'created' property\n", m.GetRef().CommonName())

						return
					}

					age := time.Now().Sub(*created)

					tag := m.GetRef().Tag

					tc <- Manifest{
						Digest: m.GetDescriptor().Digest.String(),
						Tags:   []string{tag},
						Age:    age,
					}
				}
			}()
		}
	}

	go func() {
		wg.Wait()

		close(tc)
	}()

	for t := range tc {
		tags = append(tags, t)
	}

	manifests := make(map[string]*Manifest)

	// Build distinct list of manifests and N tags
	for _, t := range tags {
		tag := t.Tags[0]
		if m, ok := manifests[t.Digest]; ok {
			m.Tags = append(m.Tags, tag)
		} else {
			m := Manifest{
				Digest: t.Digest,
				Age:    t.Age,
				Tags:   []string{tag},
			}

			manifests[t.Digest] = &m
		}
	}

	res := make([]Manifest, 0)

	for _, m := range manifests {
		res = append(res, *m)
	}

	return totalTags, res, nil
}

func getRegex(match string, isRegex bool) (*regexp.Regexp, error) {
	if match == "" {
		isRegex = false
		match = "*"
	}

	if isRegex {
		return regexp.Compile(match)
	} else {
		regexStr := "^" + regexp.QuoteMeta(match) + "$"
		regexStr = strings.ReplaceAll(regexStr, "\\*", ".*")
		regexStr = strings.ReplaceAll(regexStr, "\\?", ".")
		return regexp.Compile(regexStr)
	}
}
