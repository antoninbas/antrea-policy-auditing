package gitops

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (cr *CustomRepo) TagToCommit(tag string) (*object.Commit, error) {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	ref, err := cr.Repo.Tag(tag)
	if err != nil {
		klog.ErrorS(err, "could not retrieve tag reference")
		return nil, err
	}
	obj, err := cr.Repo.TagObject(ref.Hash())
	if err != nil {
		klog.ErrorS(err, "could not retrieve tag object")
		return nil, err
	}
	commit, err := obj.Commit()
	if err != nil {
		klog.ErrorS(err, "could not get commit from tag object")
		return nil, err
	}
	return commit, nil
}

func (cr *CustomRepo) HashToCommit(commitSha string) *object.Commit {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	hash := plumbing.NewHash(commitSha)
	commit, err := cr.Repo.CommitObject(hash)
	if err != nil {
		klog.ErrorS(err, "could not get commit from hash")
	}
	return commit
}

func (cr *CustomRepo) RollbackRepo(targetCommit *object.Commit) error {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()

	klog.V(2).Infof("Rollback to commit %s initiated, ignoring all non-rollback generated audits",
		targetCommit.Hash.String())
	cr.RollbackMode = true

	// Get patch between head and target commit
	w, err := cr.Repo.Worktree()
	if err != nil {
		klog.ErrorS(err, "unable to get git worktree from repository")
		return err
	}
	h, err := cr.Repo.Head()
	if err != nil {
		klog.ErrorS(err, "unable to get repo head")
		return err
	}
	headCommit, err := cr.Repo.CommitObject(h.Hash())
	if err != nil {
		klog.ErrorS(err, "unable to get head commit")
		return err
	}
	patch, err := headCommit.Patch(targetCommit)
	if err != nil {
		klog.ErrorS(err, "unable to get patch between commits")
		return err
	}

	// Must do cluster delete requests before resetting in order to be able to read metadata from files
	if err := cr.doDeletePatch(patch); err != nil {
		klog.ErrorS(err, "could not patch cluster to old commit state (delete phase)")
		return err
	}

	// Update repo using resets
	err = resetWorktree(w, targetCommit.Hash, git.HardReset)
	if err != nil {
		klog.ErrorS(err, "unable to hard reset repo")
		return err
	}
	err = resetWorktree(w, h.Hash(), git.SoftReset)
	if err != nil {
		klog.ErrorS(err, "unable to soft reset repo")
		return err
	}

	// Must similarly do cluster update/create requests after resetting
	if err := cr.doCreateUpdatePatch(patch); err != nil {
		klog.ErrorS(err, "could not patch cluster to old commit state (create/update phase)")
		return err
	}

	// Finally commit changes to repo after cluster updates
	username := "audit-manager"
	email := "system@audit.antrea.io"
	message := "Rollback to commit " + targetCommit.Hash.String()
	if err := cr.AddAndCommit(username, email, message); err != nil {
		klog.ErrorS(err, "error while committing rollback")
		return err
	}
	cr.RollbackMode = false
	klog.V(2).Infof("Rollback to commit %s successful", targetCommit.Hash.String())
	return nil
}

func resetWorktree(w *git.Worktree, hash plumbing.Hash, mode git.ResetMode) error {
	options := &git.ResetOptions{
		Commit: hash,
		Mode:   mode,
	}
	if err := w.Reset(options); err != nil {
		klog.ErrorS(err, "unable to reset worktree")
		return err
	}
	return nil
}

func (cr *CustomRepo) doDeletePatch(patch *object.Patch) error {
	for _, filePatch := range patch.FilePatches() {
		fromFile, toFile := filePatch.Files()
		if toFile == nil {
			path := filepath.Join(cr.Dir, fromFile.Path())
			resource, err := cr.getResourceByPath(path)
			if err != nil {
				klog.Errorf("unable to read resource at path %s", path)
				return err
			}
			if err := cr.K8s.DeleteResource(resource); err != nil {
				klog.Errorf("unable to delete resource %s", resource.GetName())
				return err
			}
			klog.V(2).Infof("(Rollback) Deleted file at %s", path)
		}
	}
	return nil
}

func (cr *CustomRepo) doCreateUpdatePatch(patch *object.Patch) error {
	for _, filePatch := range patch.FilePatches() {
		_, toFile := filePatch.Files()
		if toFile != nil {
			path := filepath.Join(cr.Dir, toFile.Path())
			resource, err := cr.getResourceByPath(path)
			if err != nil {
				klog.Errorf("unable to read resource at path %s", path)
				return err
			}
			if err := cr.K8s.CreateOrUpdateResource(resource); err != nil {
				klog.Errorf("unable to create/update resource %s", resource.GetName())
				return err
			}
			klog.V(2).Infof("(Rollback) Created/Updated file at %s", path)
		}
	}
	return nil
}

func (cr *CustomRepo) getResourceByPath(path string) (*unstructured.Unstructured, error) {
	resource := &unstructured.Unstructured{}
	gvk := schema.GroupVersionKind{}
	if err := cr.readResource(resource, path); err != nil {
		klog.ErrorS(err, "unable to read resource")
		return nil, err
	}
	apiVersion := resource.GetAPIVersion()
	kind := resource.GetKind()
	if apiVersion == "networking.k8s.io/v1" {
		gvk.Group = "networking.k8s.io"
		gvk.Version = "v1"
	} else if apiVersion == "crd.antrea.io/v1alpha1" {
		gvk.Group = "crd.antrea.io"
		gvk.Version = "v1alpha1"
	} else {
		return nil, fmt.Errorf("unknown apiVersion found: %s", apiVersion)
	}
	gvk.Kind = kind
	resource.SetGroupVersionKind(gvk)
	return resource, nil
}

func (cr *CustomRepo) readResource(resource *unstructured.Unstructured, path string) error {
	var y []byte
	if cr.StorageMode == StorageModeDisk {
		y, _ = ioutil.ReadFile(path)
	} else {
		fstat, _ := cr.Fs.Stat(path)
		y = make([]byte, fstat.Size())
		f, err := cr.Fs.Open(path)
		if err != nil {
			klog.ErrorS(err, "error opening file")
			return err
		}
		f.Read(y)
	}
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		klog.ErrorS(err, "error converting from YAML to JSON")
		return err
	}
	if err := json.Unmarshal(j, &resource.Object); err != nil {
		klog.ErrorS(err, "error while unmarshalling from file")
		return err
	}
	return nil
}
