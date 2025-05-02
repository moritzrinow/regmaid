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
	Tag    string
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

// Scans all tagged manifests of the repository and returns the ones matching the specified filter.
func (r *RegMaid) ScanRepository(ctx context.Context, repo string, match string) (int, []Manifest, error) {
	repoRef, err := ref.New(repo)
	if err != nil {

		return 0, nil, err
	}

	regex, err := getRegex(match)
	if err != nil {
		return 0, nil, err
	}

	// Cancels pending manifest retrievals running in the background
	backgroundCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	tags := make([]Manifest, 0)

	tc := make(chan Manifest)

	var wg sync.WaitGroup

	opts := []scheme.TagOpts{}

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
						fmt.Printf("Unable to determine age of manifest %s: %v\n", m.GetRef().CommonName(), err)

						return
					}

					created := ociConfig.GetConfig().Created

					age := time.Now().Sub(*created)

					tc <- Manifest{
						Digest: m.GetDescriptor().Digest.String(),
						Tag:    m.GetRef().Tag,
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

	return totalTags, tags, nil
}

func getRegex(match string) (*regexp.Regexp, error) {
	if match == "" {
		match = "*"
	}

	regexStr := "^" + regexp.QuoteMeta(match) + "$"
	regexStr = strings.ReplaceAll(regexStr, "\\*", ".*")
	regexStr = strings.ReplaceAll(regexStr, "\\?", ".")

	return regexp.Compile(regexStr)
}
