package gitops

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/klog/v2"

	v1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	klog.V(2).Infof("Rollback to commit %s initiated, ignoring all non-rollback generated audits",
		targetCommit.Hash.String())
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
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
	err = resetWorktree(w, targetCommit.Hash, true)
	if err != nil {
		klog.ErrorS(err, "unable to hard reset repo")
		return err
	}
	err = resetWorktree(w, h.Hash(), false)
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

// Resets worktree - resetMode boolean determines hard or soft reset
func resetWorktree(w *git.Worktree, hash plumbing.Hash, resetMode bool) error {
	var options *git.ResetOptions
	if resetMode {
		options = &git.ResetOptions{
			Commit: hash,
			Mode:   git.HardReset,
		}
	} else {
		options = &git.ResetOptions{
			Commit: hash,
			Mode:   git.SoftReset,
		}
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
			if err := cr.deleteResourceByPath(cr.Dir + "/" + fromFile.Path()); err != nil {
				klog.ErrorS(err, "unable to delete resource during rollback")
				return err
			}
			klog.V(2).Infof("(Rollback) Deleted file at %s", cr.Dir+"/"+fromFile.Path())
		}
	}
	return nil
}

func (cr *CustomRepo) doCreateUpdatePatch(patch *object.Patch) error {
	for _, filePatch := range patch.FilePatches() {
		_, toFile := filePatch.Files()
		if toFile != nil {
			if err := cr.createOrUpdateResourceByPath(cr.Dir + "/" + toFile.Path()); err != nil {
				klog.ErrorS(err, "unable to create/update new resouce during rollback")
				return err
			}
			klog.V(2).Infof("(Rollback) Created/Updated file at %s", cr.Dir+"/"+toFile.Path())
		}
	}
	return nil
}

func (cr *CustomRepo) createOrUpdateResourceByPath(path string) error {
	apiVersion, kind, err := cr.getMetadata(path)
	if err != nil {
		klog.ErrorS(err, "error while retrieving metadata from file")
		return err
	}
	if apiVersion == "networking.k8s.io/v1" {
		resource := &netv1.NetworkPolicy{}
		cr.getResource(resource, path)
		if err := cr.K8s.CreateOrUpdateK8sPolicy(resource); err != nil {
			klog.ErrorS(err, "unable to create/update K8s network policy during rollback")
			return err
		}
	} else if apiVersion == "crd.antrea.io/v1alpha1" {
		switch kind {
		case "NetworkPolicy":
			resource := &v1alpha1.NetworkPolicy{}
			cr.getResource(resource, path)
			if err := cr.K8s.CreateOrUpdateAntreaPolicy(resource); err != nil {
				klog.ErrorS(err, "unable to create/update Antrea network policy during rollback")
				return err
			}
		case "ClusterNetworkPolicy":
			resource := &v1alpha1.ClusterNetworkPolicy{}
			cr.getResource(resource, path)
			if err := cr.K8s.CreateOrUpdateAntreaClusterPolicy(resource); err != nil {
				klog.ErrorS(err, "unable to create/update Antrea cluster network policy during rollback")
				return err
			}
		case "Tier":
			resource := &v1alpha1.Tier{}
			cr.getResource(resource, path)
			if err := cr.K8s.CreateOrUpdateAntreaTier(resource); err != nil {
				klog.ErrorS(err, "unable to create/update Antrea tier during rollback")
				return err
			}
		default:
			klog.ErrorS(err, "unknown kind found", "kind", kind)
			return err
		}
	} else {
		klog.ErrorS(err, "unknown apiVersion found", "version", apiVersion)
		return err
	}
	return nil
}

func (cr *CustomRepo) deleteResourceByPath(path string) error {
	apiVersion, kind, err := cr.getMetadata(path)
	if err != nil {
		klog.ErrorS(err, "error while retrieving metadata from file")
		return err
	}
	if apiVersion == "networking.k8s.io/v1" {
		resource := &netv1.NetworkPolicy{}
		cr.getResource(resource, path)
		if err := cr.K8s.DeleteK8sPolicy(resource); err != nil {
			klog.ErrorS(err, "unable to delete K8s network policy during rollback")
			return err
		}
	} else if apiVersion == "crd.antrea.io/v1alpha1" {
		switch kind {
		case "NetworkPolicy":
			resource := &v1alpha1.NetworkPolicy{}
			cr.getResource(resource, path)
			if err := cr.K8s.DeleteAntreaPolicy(resource); err != nil {
				klog.ErrorS(err, "unable to delete Antrea network policy during rollback")
				return err
			}
		case "ClusterNetworkPolicy":
			resource := &v1alpha1.ClusterNetworkPolicy{}
			cr.getResource(resource, path)
			if err := cr.K8s.DeleteAntreaClusterPolicy(resource); err != nil {
				klog.ErrorS(err, "unable to delete Antrea cluster network policy during rollback")
				return err
			}
		case "Tier":
			resource := &v1alpha1.Tier{}
			cr.getResource(resource, path)
			if err := cr.K8s.DeleteAntreaTier(resource); err != nil {
				klog.ErrorS(err, "unable to delete Antrea tier during rollback")
				return err
			}
		default:
			klog.ErrorS(err, "unknown resource kind found", "kind", kind)
			return err
		}
	} else {
		klog.ErrorS(err, "unknown apiVersion found", "version", apiVersion)
		return err
	}
	return nil
}

func (cr *CustomRepo) getMetadata(path string) (string, string, error) {
	meta := metav1.TypeMeta{}
	var y []byte
	if cr.StorageMode == StorageModeDisk {
		y, _ = ioutil.ReadFile(path)
	} else {
		fstat, _ := cr.Fs.Stat(path)
		y = make([]byte, fstat.Size())
		f, err := cr.Fs.Open(path)
		if err != nil {
			klog.ErrorS(err, "error opening file")
			return "", "", err
		}
		f.Read(y)
	}
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		klog.ErrorS(err, "error converting from YAML to JSON")
		return "", "", err
	}
	if err := json.Unmarshal(j, &meta); err != nil {
		klog.ErrorS(err, "error while unmarshalling from file")
		return "", "", err
	}
	return meta.APIVersion, meta.Kind, nil
}

func (cr *CustomRepo) getResource(resource runtime.Object, path string) {
	var y []byte
	if cr.StorageMode == StorageModeDisk {
		y, _ = ioutil.ReadFile(path)
	} else {
		fstat, _ := cr.Fs.Stat(path)
		y = make([]byte, fstat.Size())
		f, err := cr.Fs.Open(path)
		if err != nil {
			klog.ErrorS(err, "error opening file")
			return
		}
		f.Read(y)
	}
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		klog.ErrorS(err, "error converting from YAML to JSON")
		return
	}
	if err := json.Unmarshal(j, resource); err != nil {
		klog.ErrorS(err, "error while unmarshalling from file")
		return
	}
}
