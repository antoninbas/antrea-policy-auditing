package gitops

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"k8s.io/klog/v2"
)

func (cr *CustomRepo) TagCommit(commit_sha string, tag string, tagger *object.Signature) error {
	hash := plumbing.NewHash(commit_sha)
	_, err := cr.Repo.CommitObject(hash)
	if err != nil {
		return fmt.Errorf("unable to get commit object")
	}
	if err = setTag(cr.Repo, hash, tag, tagger); err != nil {
		return fmt.Errorf("unable to create tag: %w", err)
	}
	return nil
}

func (cr *CustomRepo) RemoveTag(tag string) error {
	if err := cr.Repo.DeleteTag(tag); err != nil {
		return fmt.Errorf("unable to delete tag")
	}
	return nil
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
			return fmt.Errorf("error creating tag")
		}
	}
	klog.V(2).InfoS("Tag created", "tagName", tag)
	return nil
}
