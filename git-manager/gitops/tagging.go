package gitops

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"k8s.io/klog/v2"
)

func TagCommit(r *git.Repository, commit_sha string, tag string) error {
	hash := plumbing.NewHash(commit_sha)
	_, err := r.CommitObject(hash)
	if err != nil {
		klog.ErrorS(err, "could not get commit object")
		return err
	}
	_, err = setTag(r, hash, tag, &object.Signature{
		Name:  "stan",
		Email: "swong394@gmail.com",
		When:  time.Now()})
	if err != nil {
		klog.ErrorS(err, "create tag error")
		return err
	}
	return nil
}

func tagExists(r *git.Repository, tag string) bool {
	tagFoundErr := "tag was found"
	tags, err := r.TagObjects()
	if err != nil {
		klog.Errorf("get tags error: %s", err)
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
		klog.Errorf("iterate tags error: %s", err)
		return false
	}
	return res
}

func setTag(r *git.Repository, commit_sha plumbing.Hash, tag string, tagger *object.Signature) (bool, error) {
	if tagExists(r, tag) {
		klog.Infof("tag %s already exists", tag)
		return false, nil
	}
	_, err := r.CreateTag(tag, commit_sha, &git.CreateTagOptions{
		Tagger:  tagger,
		Message: tag,
	})
	if err != nil {
		klog.Errorf("create tag error: %s", err)
		return false, err
	}

	return true, nil
}
