package gitops

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
	"k8s.io/klog/v2"
)

func AddAndCommit(r *git.Repository, username string, email string, message string) error {
	w, err := r.Worktree()
	if err != nil {
		klog.ErrorS(err, "unable to get git worktree from repository")
		return err
	}
	_, err = w.Add(".")
	if err != nil {
		klog.ErrorS(err, "unable to add git change to worktree")
		return err
	}
	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  username,
			Email: email,
			When:  time.Now(),
		},
	})
	if err != nil {
		klog.ErrorS(err, "unable to commit git change to worktree")
		return err
	}
	return nil
}

func modifyFile(dir string, event auditv1.Event) error {
	y, err := yaml.JSONToYAML(event.ResponseObject.Raw)
	if err != nil {
		klog.ErrorS(err, "unable to convert event ResponseObject from JSON to YAML format")
		return err
	}
	path := getAbsRepoPath(dir, event)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0700)
	}
	path += getFileName(event)
	if err := ioutil.WriteFile(path, y, 0644); err != nil {
		klog.ErrorS(err, "unable to write/update file in repository")
		return err
	}
	return nil
}

func deleteFile(r *git.Repository, dir string, event auditv1.Event) error {
	w, err := r.Worktree()
	if err != nil {
		klog.ErrorS(err, "unable to get git worktree from repository")
		return err
	}
	path := getRelRepoPath(event) + getFileName(event)
	_, err = w.Remove(path)
	if err != nil {
		klog.ErrorS(err, "unable to remove file", "path", path)
		return err
	}
	return nil
}
