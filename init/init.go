package main

import (
    "fmt"
    "os"
    "time"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
    "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing/object"

    . "audit_init/setup"
)

var directory string
var namespaces []string

func SetupRepo(k *Kubernetes) (error) {
    if directory == "" {
        path, err := os.Getwd()
        if err != nil {
            return errors.WithMessagef(err, "could not retrieve the current working directory")
        }
        directory = path
    }
    os.Mkdir(directory+"network-policy-repository", 0700)
    r, err := git.PlainInit(directory+"/network-policy-repository/", false)
    if err != nil {
		return errors.WithMessagef(err, "could not initialize git repo")
	}
    w, err := r.Worktree()
    if err != nil {
		return errors.WithMessagef(err, "could not intialize git worktree")
	}

    os.Mkdir(directory + "/network-policy-repository/k8s-policy", 0700)
    os.Mkdir(directory + "/network-policy-repository/antrea-policy", 0700)
    os.Mkdir(directory + "/network-policy-repository/antrea-cluster-policy", 0700)

	if err := addPolicies(k); err != nil {
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
	return nil
}

func addPolicies(k *Kubernetes) error {
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

func addK8sPolicies(k *Kubernetes) error {
	policies, err := k.GetK8sPolicies()
	if err != nil {
		return err
	}
	for _, np := range policies.Items {
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			os.Mkdir(directory + "/network-policy-repository/k8s-policy/" + np.Namespace, 0700)
		}
		path := directory + "/network-policy-repository/k8s-policy/" + np.Namespace + "/" + np.Name + ".yaml"
		fmt.Println(path)
		y, err := yaml.JSONToYAML([]byte(np.Annotations["kubectl.kubernetes.io/last-applied-configuration"]))
		if err != nil {
			return errors.Wrapf(err, "unable to convert network policy object")
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			return errors.Wrapf(err, "unable to write policy config to file")
		}
	}
	return nil
}

func addAntreaPolicies(k *Kubernetes) error {
	return nil
}

func addAntreaClusterPolicies(k *Kubernetes) error {
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

func main() {
	k8s, err := NewKubernetes()
	if err != nil {
		fmt.Println(err)
	}
	
	if err := SetupRepo(k8s); err != nil {
		fmt.Println(err)
	}
}