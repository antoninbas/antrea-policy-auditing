package gitops

import (
	"encoding/json"
	"time"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
	"k8s.io/klog/v2"
)

func (cr *CustomRepo) AddAndCommit(username string, email string, message string) error {
	w, err := cr.Repo.Worktree()
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

func (cr *CustomRepo) modifyFile(event auditv1.Event) error {
	resource := unstructured.Unstructured{}
	if err := json.Unmarshal(event.ResponseObject.Raw, &resource); err != nil {
		klog.ErrorS(err, "unable to unmarshal ResponseObject resource config")
		return err
	}
	clearFields(&resource)
	y, err := yaml.Marshal(&resource)
	if err != nil {
		klog.ErrorS(err, "unable to marshal new resource config")
		return err
	}
	path := getAbsRepoPath("", event)
	path += getFileName(event)
	newfile, err := cr.Fs.Create(path)
	if err != nil {
		klog.ErrorS(err, "unable to create file at: ", "path", path)
		return err
	}
	newfile.Write(y)
	newfile.Close()
	return nil
}

func (cr *CustomRepo) deleteFile(event auditv1.Event) error {
	w, err := cr.Repo.Worktree()
	if err != nil {
		klog.ErrorS(err, "unable to get git worktree from repository")
		return err
	}
	path := getRelRepoPath(event) + getFileName(event)
	_, err = w.Remove(path)
	if err != nil {
		klog.ErrorS(err, "unable to remove file at: ", "path", path)
		return err
	}
	return nil
}
