package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v66/github"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

const (
	labelKey   = "windkube.com/infra-ready"
	labelValue = "true"
)

func main() {
	ctx := context.Background()

	repoFull := mustEnv("GITHUB_REPOSITORY")
	filePath := mustEnv("CLUSTER_FILE_PATH")
	token := mustEnv("GITHUB_TOKEN")

	owner, repo, ok := strings.Cut(repoFull, "/")
	if !ok {
		log.Fatalf("GITHUB_REPOSITORY must be in owner/repo form, got %q", repoFull)
	}

	gh := github.NewClient(nil).WithAuthToken(token)

	repoInfo, _, err := gh.Repositories.Get(ctx, owner, repo)
	if err != nil {
		log.Fatalf("get repo: %v", err)
	}
	baseBranch := repoInfo.GetDefaultBranch()

	fileContent, _, _, err := gh.Repositories.GetContents(ctx, owner, repo, filePath,
		&github.RepositoryContentGetOptions{Ref: baseBranch})
	if err != nil {
		log.Fatalf("get file: %v", err)
	}
	original, err := fileContent.GetContent()
	if err != nil {
		log.Fatalf("decode file: %v", err)
	}

	updated, changed, err := addInfraReadyLabel([]byte(original))
	if err != nil {
		log.Fatalf("apply label: %v", err)
	}
	if !changed {
		log.Printf("%s already has %s=%s, nothing to do", filePath, labelKey, labelValue)
		return
	}

	cluster := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	branch := fmt.Sprintf("infra-ready/%s-%d", cluster, time.Now().Unix())

	baseRef, _, err := gh.Git.GetRef(ctx, owner, repo, "heads/"+baseBranch)
	if err != nil {
		log.Fatalf("get base ref: %v", err)
	}
	newRef := &github.Reference{
		Ref:    github.String("refs/heads/" + branch),
		Object: &github.GitObject{SHA: baseRef.Object.SHA},
	}
	if _, _, err := gh.Git.CreateRef(ctx, owner, repo, newRef); err != nil {
		log.Fatalf("create branch: %v", err)
	}

	msg := fmt.Sprintf("Mark cluster %s as infra-ready", cluster)
	_, _, err = gh.Repositories.UpdateFile(ctx, owner, repo, filePath, &github.RepositoryContentFileOptions{
		Message: github.String(msg),
		Content: updated,
		SHA:     fileContent.SHA,
		Branch:  github.String(branch),
	})
	if err != nil {
		log.Fatalf("update file: %v", err)
	}

	pr, _, err := gh.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String(msg),
		Head:  github.String(branch),
		Base:  github.String(baseBranch),
		Body:  github.String(fmt.Sprintf("Adds `%s: %s` label to `%s`.", labelKey, labelValue, filePath)),
	})
	if err != nil {
		log.Fatalf("create PR: %v", err)
	}

	log.Printf("PR created: %s", pr.GetHTMLURL())
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("required env %s is not set", k)
	}
	return v
}

// addInfraReadyLabel decodes the cluster YAML as a Kubernetes Secret using
// client-go's scheme, sets the infra-ready label, and re-serializes the
// manifest. Returns the new bytes and whether anything changed.
func addInfraReadyLabel(in []byte) ([]byte, bool, error) {
	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(in, nil, nil)
	if err != nil {
		return nil, false, fmt.Errorf("decode secret: %w", err)
	}
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil, false, fmt.Errorf("expected *corev1.Secret, got %T", obj)
	}

	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}
	if secret.Labels[labelKey] == labelValue {
		return in, false, nil
	}
	secret.Labels[labelKey] = labelValue

	// Marshal via sigs.k8s.io/yaml so output is canonical, deterministic YAML.
	out, err := yaml.Marshal(secret)
	if err != nil {
		return nil, false, fmt.Errorf("encode secret: %w", err)
	}
	return out, true, nil
}
