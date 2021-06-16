package init

import (
    "fmt"
    "os"
    "time"
	"io/ioutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	billy "github.com/go-git/go-billy/v5"
	memory "github.com/go-git/go-git/v5/storage/memory"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
    "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing/object"
)

var directory string

// TODO: Refactor code to get rid of redundant InMem functions

func SetupRepo(k *Kubernetes) error {
    if directory == "" {
        path, err := os.Getwd()
        if err != nil {
            return errors.WithMessagef(err, "could not retrieve the current working directory")
        }
        directory = path
    }
    r, err := git.PlainInit(directory + "/network-policy-repository/", false)
    if err == git.ErrRepositoryAlreadyExists {
		return errors.WithMessagef(err, "Repository already exists, skipping initialization")
	} else if err != nil {
		return errors.WithMessagef(err, "could not initialize git repo")
	}

    w, err := r.Worktree()
    if err != nil {
		return errors.WithMessagef(err, "could not intialize git worktree")
	}
	if err := addNetworkPolicies(k); err != nil {
		return errors.WithMessagef(err, "couldn't write network policies")
	}
    _, err = w.Add(".")
    if err != nil {
		return errors.WithMessagef(err, "couldn't git add changes")
	}
    _, err = w.Commit("initial commit of existing policies", &git.CommitOptions{
        Author: &object.Signature{
            Name:  "audit-init",
            Email: "system@audit.antrea.io",
            When:  time.Now(),
        },
    })
    if err != nil {
		return errors.WithMessagef(err, "couldn't git commit changes")
	}
	fmt.Println("Repository successfully initialized")
	return nil
}

func SetupRepoInMem(k *Kubernetes, storer *memory.Storage, fs billy.Filesystem) error {
	r, err := git.Init(storer, fs)
    if err == git.ErrRepositoryAlreadyExists {
		return errors.WithMessagef(err, "Repository already exists, skipping initialization")
	} else if err != nil {
		return errors.WithMessagef(err, "could not initialize git repo")
	}

    w, err := r.Worktree()
    if err != nil {
		return errors.WithMessagef(err, "could not intialize git worktree")
	}
	if err := addNetworkPoliciesInMem(k, fs); err != nil {
		return errors.WithMessagef(err, "couldn't write network policies")
	}
    _, err = w.Add(".")
    if err != nil {
		return errors.WithMessagef(err, "couldn't git add changes")
	}
    _, err = w.Commit("initial commit of existing policies", &git.CommitOptions{
        Author: &object.Signature{
            Name:  "audit-init",
            Email: "system@audit.antrea.io",
            When:  time.Now(),
        },
    })
    if err != nil {
		return errors.WithMessagef(err, "couldn't git commit changes")
	}
	fmt.Println("Repository successfully initialized")
	return nil
}

func addNetworkPolicies(k *Kubernetes) error {
    os.Mkdir(directory + "/network-policy-repository/k8s-policy", 0700)
    os.Mkdir(directory + "/network-policy-repository/antrea-policy", 0700)
    os.Mkdir(directory + "/network-policy-repository/antrea-cluster-policy", 0700)
	if err := addK8sPolicies(k); err != nil {
		return err
	}
	if err := addAntreaPolicies(k); err != nil {
		return err
	}
	if err := addAntreaClusterPolicies(k); err != nil {
		return err
	}
	return nil
}

func addNetworkPoliciesInMem(k *Kubernetes, fs billy.Filesystem) error {
	fs.MkdirAll("k8s-policy", 0700)
	fs.MkdirAll("antrea-policy", 0700)
	fs.MkdirAll("antrea-cluster-policy", 0700)
	if err := addK8sPoliciesInMem(k, fs); err != nil {
		return err
	}
	if err := addAntreaPoliciesInMem(k, fs); err != nil {
		return err
	}
	if err := addAntreaClusterPoliciesInMem(k, fs); err != nil {
		return err
	}
	return nil
}

func addK8sPolicies(k *Kubernetes) error {
	policies, err := k.GetK8sPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			os.Mkdir(directory + "/network-policy-repository/k8s-policy/" + np.Namespace, 0700)
		}
		path := directory + "/network-policy-repository/k8s-policy/" + np.Namespace + "/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			return errors.Wrapf(err, "unable to write policy config to file")
		}
	}
	return nil
}

func addK8sPoliciesInMem(k *Kubernetes, fs billy.Filesystem) error {
	policies, err := k.GetK8sPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			fs.MkdirAll("k8s-policy/" + np.Namespace, 0700)
		}
		path := "k8s-policy/" + np.Namespace + "/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		newFile, err := fs.Create(path)
		if err != nil {
			return errors.Wrapf(err, "unable to create file")
		}
		newFile.Write(y)
		newFile.Close()
	}
	return nil
}

func addAntreaPolicies(k *Kubernetes) error {
	policies, err := k.GetAntreaPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			os.Mkdir(directory + "/network-policy-repository/antrea-policy/" + np.Namespace, 0700)
		}
		path := directory + "/network-policy-repository/antrea-policy/" + np.Namespace + "/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			return errors.Wrapf(err, "unable to write policy config to file")
		}
	}
	return nil
}

func addAntreaPoliciesInMem(k *Kubernetes, fs billy.Filesystem) error {
	policies, err := k.GetAntreaPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			fs.MkdirAll("antrea-policy/" + np.Namespace, 0700)
		}
		path := "antrea-policy/" + np.Namespace + "/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		newFile, err := fs.Create(path)
		if err != nil {
			return errors.Wrapf(err, "unable to create file")
		}
		newFile.Write(y)
		newFile.Close()
	}
	return nil
}

func addAntreaClusterPolicies(k *Kubernetes) error {
	policies, err := k.GetAntreaClusterPolicies()
	if err != nil {
		return err
	}
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "ClusterNetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		path := directory + "/network-policy-repository/antrea-cluster-policy/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			return errors.Wrapf(err, "unable to write policy config to file")
		}
	}
	return nil
}

func addAntreaClusterPoliciesInMem(k *Kubernetes, fs billy.Filesystem) error {
	policies, err := k.GetAntreaClusterPolicies()
	if err != nil {
		return err
	}
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "ClusterNetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		path := "antrea-cluster-policy/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		newFile, err := fs.Create(path)
		if err != nil {
			return errors.Wrapf(err, "unable to create file")
		}
		newFile.Write(y)
		newFile.Close()
	}
	return nil
}

func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}
