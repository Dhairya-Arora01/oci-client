package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

const (
	theRegistry = "ghcr.io"
	theRepo     = "ghcr.io/abc/my-repo"
)

func main() {
	client, err := newOCIClient(theRegistry, theRepo)
	if err != nil {
		panic(fmt.Errorf("failed to create new OCI client: %w", err))
	}

	ctx := context.Background()

	tags, err := client.listReleases(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to list releases: %w", err))
	}

	for _, tag := range tags {
		fmt.Println(tag)
	}

	releaseManifest, err := client.getReleaseByTag(ctx, "v13")
	if err != nil {
		panic(fmt.Errorf("failed to get release by tag: %w", err))
	}

	fmt.Println("descriptor", releaseManifest)

	if err := client.downloadReleaseAssets(ctx, "v13"); err != nil {
		panic(fmt.Errorf("failed to download release assets: %w", err))
	}
}

type ociClient struct {
	client     *auth.Client
	repository *remote.Repository
}

func newOCIClient(registry, repository string) (*ociClient, error) {
	accessToken := os.Getenv("MY_PAT")
	if accessToken == "" {
		return nil, errors.New("no access token provided")
	}

	client := auth.Client{
		Credential: auth.StaticCredential(registry, auth.Credential{
			AccessToken: accessToken,
		}),
	}

	repo, err := remote.NewRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for the repository: %w", err)
	}

	repo.Client = &client

	return &ociClient{
		client:     &client,
		repository: repo,
	}, nil
}

func (c *ociClient) listReleases(ctx context.Context) ([]string, error) {
	tags, err := registry.Tags(ctx, c.repository)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, nil
}

func (c *ociClient) getReleaseByTag(ctx context.Context, tag string) (string, error) {
	descriptor, err := c.repository.Resolve(ctx, tag)
	if err != nil {
		return "", fmt.Errorf("failed to resolve tag: %w", err)
	}

	manifest, err := content.FetchAll(ctx, c.repository, descriptor)
	if err != nil {
		return "", fmt.Errorf("failed to pull the manifest: %w", err)
	}

	return string(manifest), err
}

func (c *ociClient) downloadReleaseAssets(ctx context.Context, tag string) error {
	dest, err := file.New("/home/me/oci-oras")
	if err != nil {
		return fmt.Errorf("failed to create file store: %w", err)
	}

	defer dest.Close()

	_, err = oras.Copy(ctx, c.repository, tag, dest, tag, oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("failed to copy assets from the registry: %w", err)
	}

	return nil
}
