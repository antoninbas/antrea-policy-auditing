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

func TagToCommit(r *git.Repository, tag string) (*object.Commit, error) {
	ref, err := r.Tag(tag)
	if err != nil {
		klog.ErrorS(err, "could not retrieve tag reference")
		return nil, err
	}
	obj, err := r.TagObject(ref.Hash())
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

func HashToCommit(r *git.Repository, commitSha string) *object.Commit {
	hash := plumbing.NewHash(commitSha)
	commit, err := r.CommitObject(hash)
	if err != nil {
		klog.ErrorS(err, "could not get commit from hash")
	}
	return commit
}

func RollbackRepo(repoDir string, r *git.Repository, targetCommit *object.Commit) error {
	klog.V(2).Infof("Rollback to commit %s initiated, ignoring all non-rollback generated audits",
		targetCommit.Hash.String())

	// Get patch between head and target commit
	w, err := r.Worktree()
	if err != nil {
		klog.ErrorS(err, "unable to get git worktree from repository")
		return err
	}
	h, err := r.Head()
	if err != nil {
		klog.ErrorS(err, "unable to get repo head")
		return err
	}
	headCommit, err := r.CommitObject(h.Hash())
	if err != nil {
		klog.ErrorS(err, "unable to get head commit")
		return err
	}
	patch, err := headCommit.Patch(targetCommit)
	if err != nil {
		klog.ErrorS(err, "unable to get patch between commits")
		return err
	}
	k8s, err := NewKubernetes()
	if err != nil {
		klog.ErrorS(err, "error while setting up new kube clients")
		return err
	}
	// Must do cluster delete requests before resetting in order to be able to read metadata from files
	if err := deletePatch(repoDir, patch, k8s); err != nil {
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
	if err := createUpdatePatch(repoDir, patch, k8s); err != nil {
		klog.ErrorS(err, "could not patch cluster to old commit state (create/update phase)")
		return err
	}

	// Finally commit changes to repo after cluster updates
	// username := "audit-manager"
	// email := "system@audit.antrea.io"
	// message := "Rollback to commit " + targetCommit.Hash.String()
	// if err := AddAndCommit(r, username, email, message); err != nil {
	// 	klog.ErrorS(err, "error while committing rollback")
	// 	return err
	// }
	klog.Infof("Rollback to commit %s successful", targetCommit.Hash.String())
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

func deletePatch(repoDir string, patch *object.Patch, k8s *Kubernetes) error {
	for _, filePatch := range patch.FilePatches() {
		fromFile, toFile := filePatch.Files()
		if toFile == nil {
			if err := DeleteResource(k8s, repoDir+"/"+fromFile.Path()); err != nil {
				klog.ErrorS(err, "unable to delete resource during rollback")
				return err
			}
			klog.Infof("Deleted file at %s", repoDir+"/"+fromFile.Path())
		}
	}
	return nil
}

func createUpdatePatch(repoDir string, patch *object.Patch, k8s *Kubernetes) error {
	for _, filePatch := range patch.FilePatches() {
		_, toFile := filePatch.Files()
		if toFile != nil {
			if err := CreateOrUpdateResource(k8s, repoDir+"/"+toFile.Path()); err != nil {
				klog.ErrorS(err, "unable to create/update new resouce during rollback")
				return err
			}
			klog.Infof("Created/Updated file at %s", repoDir+"/"+toFile.Path())
		}
	}
	return nil
}

func CreateOrUpdateResource(k *Kubernetes, path string) error {
	apiVersion, kind, err := getMetadata(path)
	if err != nil {
		klog.ErrorS(err, "error while retrieving metadata from file")
		return err
	}
	if apiVersion == "networking.k8s.io/v1" {
		resource := &netv1.NetworkPolicy{}
		getResource(resource, path)
		if err := k.CreateOrUpdateK8sPolicy(resource); err != nil {
			klog.ErrorS(err, "unable to create/update K8s network policy")
			return err
		}
	} else if apiVersion == "crd.antrea.io/v1alpha1" {
		switch kind {
		case "NetworkPolicy":
			resource := &v1alpha1.NetworkPolicy{}
			getResource(resource, path)
			if err := k.CreateOrUpdateAntreaPolicy(resource); err != nil {
				klog.ErrorS(err, "unable to create/update Antrea network policy during rollback")
				return err
			}
		case "ClusterNetworkPolicy":
			resource := &v1alpha1.ClusterNetworkPolicy{}
			getResource(resource, path)
			if err := k.CreateOrUpdateAntreaClusterPolicy(resource); err != nil {
				klog.ErrorS(err, "unable to create/update Antrea cluster network policy during rollback")
				return err
			}
		case "Tier":
			resource := &v1alpha1.Tier{}
			getResource(resource, path)
			if err := k.CreateOrUpdateAntreaTier(resource); err != nil {
				klog.ErrorS(err, "unable to create/update Antrea tier during rollback")
				return err
			}
		default:
			klog.ErrorS(err, "unknown kind found")
			return err
		}
	} else {
		klog.ErrorS(err, "unknown apiVersion found")
		return err
	}
	return nil
}

func DeleteResource(k *Kubernetes, path string) error {
	apiVersion, kind, err := getMetadata(path)
	if err != nil {
		klog.ErrorS(err, "error while retrieving metadata from file")
		return err
	}
	if apiVersion == "networking.k8s.io/v1" {
		resource := &netv1.NetworkPolicy{}
		getResource(resource, path)
		if err := k.DeleteK8sPolicy(resource); err != nil {
			klog.ErrorS(err, "unable to delete K8s network policy")
			return err
		}
	} else if apiVersion == "crd.antrea.io/v1alpha1" {
		switch kind {
		case "NetworkPolicy":
			resource := &v1alpha1.NetworkPolicy{}
			getResource(resource, path)
			if err := k.DeleteAntreaPolicy(resource); err != nil {
				klog.ErrorS(err, "unable to delete Antrea network policy during rollback")
				return err
			}
		case "ClusterNetworkPolicy":
			resource := &v1alpha1.ClusterNetworkPolicy{}
			getResource(resource, path)
			if err := k.DeleteAntreaClusterPolicy(resource); err != nil {
				klog.ErrorS(err, "unable to delete Antrea cluster network policy during rollback")
				return err
			}
		case "Tier":
			resource := &v1alpha1.Tier{}
			getResource(resource, path)
			if err := k.DeleteAntreaTier(resource); err != nil {
				klog.ErrorS(err, "unable to delete Antrea tier during rollback")
				return err
			}
		default:
			klog.ErrorS(err, "unknown resource kind found")
			return err
		}
	} else {
		klog.ErrorS(err, "unknown apiVersion found")
		return err
	}
	return nil
}

func getMetadata(path string) (string, string, error) {
	meta := metav1.TypeMeta{}
	y, err := ioutil.ReadFile(path)
	if err != nil {
		klog.ErrorS(err, "error reading file")
		return "", "", err
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

func getResource(resource runtime.Object, path string) {
	y, err := ioutil.ReadFile(path)
	if err != nil {
		klog.ErrorS(err, "error reading file")
		return
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
