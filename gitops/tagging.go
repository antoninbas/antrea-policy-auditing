package gitops

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"k8s.io/klog/v2"
)

func (cr *CustomRepo) TagCommit(commit_sha string, tag string, tagger *object.Signature) (string, error) {
	hash := plumbing.NewHash(commit_sha)
	_, err := cr.Repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("unable to get commit object: %w", err)
	}
	if err = setTag(cr.Repo, hash, tag, tagger); err != nil {
		return "", fmt.Errorf("unable to create tag: %w", err)
	}
	klog.V(2).InfoS("tag created", "tagName", tag, "commit", commit_sha)
	return commit_sha, nil
}

func (cr *CustomRepo) RemoveTag(tag string) (string, error) {
	if err := cr.Repo.DeleteTag(tag); err != nil {
		return "", fmt.Errorf("unable to delete tag: %w", err)
	}
	klog.V(2).InfoS("tag deleted", "tagName", tag)
	return tag, nil
}

func setTag(r *git.Repository, commit_sha plumbing.Hash, tag string, tagger *object.Signature) error {
	_, err := r.CreateTag(tag, commit_sha, &git.CreateTagOptions{
		Tagger:  tagger,
		Message: tag,
	})
	if err != nil {
		if err.Error() == "tag already exists" {
			return err
		} else {
			return fmt.Errorf("error creating tag: %w", err)
		}
	}
	return nil
}
