package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/logsquaredn/blobproxy/.dagger/internal/dagger"
)

type BlobproxyDev struct {}

const (
	gid   = "1001"
	uid   = gid
	group = "blobproxy"
	user  = group
	home  = "/home/" + user
)

func (m *BlobproxyDev) Version(ctx context.Context, workspace *dagger.Workspace) string {
	version := "v0.0.0-unknown"

	gitRepo := workspace.Directory(".").AsGit()
	gitRef := gitRepo.LatestVersion()

	if ref, err := gitRef.Ref(ctx); err == nil {
		version = strings.TrimPrefix(ref, "refs/tags/")
	}

	if latestVersionCommit, err := gitRef.Commit(ctx); err == nil {
		if headCommit, err := gitRepo.Head().Commit(ctx); err == nil {
			if headCommit != latestVersionCommit {
				if len(headCommit) > 7 {
					headCommit = headCommit[:7]
				}
				version += "-" + headCommit
			}
		}
	}

	if empty, _ := gitRepo.Uncommitted().IsEmpty(ctx); !empty {
		version += "+dirty"
	}

	return version
}

func (m *BlobproxyDev) Tag(ctx context.Context, workspace *dagger.Workspace) string {
	before, _, _ := strings.Cut(strings.TrimPrefix(m.Version(ctx, workspace), "v"), "+")
	return before
}

func (m *BlobproxyDev) Binary(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +default=v0.0.0-unknown
	version,
	// +optional
	goarch,
	// +optional
	goos string,
) *dagger.File {
	return dag.Go(dagger.GoOpts{
		Workspace: workspace,
	}).
		Build(dagger.GoBuildOpts{
			Pkg:     "./cmd/blobproxy",
			Ldflags: "-s -w -X main.version=" + version,
			Goarch: goarch,
		})
}

// +check
func (m *BlobproxyDev) Container(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	arch string,
) *dagger.Container {
	return dag.Wolfi().
		Container(dagger.WolfiContainerOpts{
			Arch: arch,
		}).
		WithExec([]string{"adduser", "-D", "-h", home, "-u", uid, "-g", group, user}).
		WithFile("/usr/local/bin/blobproxy", m.Binary(ctx, workspace, m.Version(ctx, workspace), arch, "")).
		WithEntrypoint([]string{"blobproxy"})
}

func (m *BlobproxyDev) Ship(
	ctx context.Context,
	workspace *dagger.Workspace,
	githubRepo string,
	githubToken *dagger.Secret,
	// +optional
	// +default="latest"
	tag string,
) error {
	registry := "ghcr.io"
	container := m.Container(ctx, workspace, "amd64").WithRegistryAuth(registry, "x-access-token", githubToken)
	if _, err := container.Publish(ctx, fmt.Sprintf("%s/%s:%s", registry, githubRepo, tag), dagger.ContainerPublishOpts{
		PlatformVariants: []*dagger.Container{m.Container(ctx, workspace, "arm64")},
	}); err != nil {
		return err
	}

	return nil
}

func (m *BlobproxyDev) Release(
	ctx context.Context,
	workspace *dagger.Workspace,
	githubRepo string,
	githubToken *dagger.Secret,
) error {
	gitRepo := workspace.Directory(".").AsGit()
	latestVersion := gitRepo.LatestVersion()
	rawLatestVersionRef, err := latestVersion.Ref(ctx)
	if err != nil {
		return err
	}

	trimmedRawLatestVersionRef := strings.TrimPrefix(rawLatestVersionRef, "refs/tags/")
	latestVersionRef, err := semver.NewVersion(trimmedRawLatestVersionRef)
	if err != nil {
		return fmt.Errorf("%w: %s", err, trimmedRawLatestVersionRef)
	}

	tags := []string{"latest", latestVersionRef.String()}
	if latestVersionRef.Prerelease() == "" {
		tags = append(tags,
			fmt.Sprintf("%d.%d", latestVersionRef.Major(), latestVersionRef.Minor()),
			fmt.Sprintf("%d", latestVersionRef.Major()),
		)
	}
	for _, tag := range tags {
		if err := m.Ship(ctx, workspace, githubRepo, githubToken, tag); err != nil {
			return err
		}
	}

	return dag.Release(latestVersion).
		Create(ctx, githubToken, githubRepo, "blobproxy")
}
