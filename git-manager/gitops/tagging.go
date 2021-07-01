package gitops

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"k8s.io/klog/v2"
)

func TagCommit(r *git.Repository, commit_sha string, tag string, tagger *object.Signature) error {
	hash := plumbing.NewHash(commit_sha)
	_, err := r.CommitObject(hash)
	if err != nil {
		klog.ErrorS(err, "Unable to get commit object")
		return err
	}
	if err = setTag(r, hash, tag, tagger); err != nil {
		klog.ErrorS(err, "Unable to create tag")
		return err
	}
	return nil
}

func RemoveTag(r *git.Repository, tag string) error {
	if err := r.DeleteTag(tag); err != nil {
		klog.ErrorS(err, "Unable to delete tag")
		return err
	}
	return nil
}

func setTag(r *git.Repository, commit_sha plumbing.Hash, tag string, tagger *object.Signature) error {
	if tagExists(r, tag) {
		klog.V(2).Infof("Unable to create tag: %s already exists", tag)
		return nil
	}
	_, err := r.CreateTag(tag, commit_sha, &git.CreateTagOptions{
		Tagger:  tagger,
		Message: tag,
	})
	if err != nil {
		klog.ErrorS(err, "Error creating tag")
		return err
	}
	klog.V(2).Infof("Tag created: %s", tag)
	return nil
}

func tagExists(r *git.Repository, tag string) bool {
	tagFoundErr := "Tag already exists"
	tags, err := r.TagObjects()
	if err != nil {
		klog.ErrorS(err, "Error while getting tags")
		return false
	}
	res := false
	err = tags.ForEach(func(t *object.Tag) error {
		if t.Name == tag {
			res = true
			return fmt.Errorf(tagFoundErr)
		}
		return nil
	})
	if err != nil && err.Error() != tagFoundErr {
		klog.ErrorS(err, "Error while iterating through tags")
		return false
	}
	return res
}
