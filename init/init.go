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

func SetupRepo() (error) {
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

	k8s, err := NewKubernetes()
	if err != nil {
		fmt.Println("something went wrong when setting up the kube client")
	}

	policies, err := k8s.GetK8sPolicies()
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

    fmt.Println("git add .")
    _, err = w.Add(".")
    if err != nil {
		return errors.WithMessagef(err, "couldn't git add changes")
	}

    fmt.Println("git commit -m \"test commit number 1a\"")
    _, err = w.Commit("test commit number 1a", &git.CommitOptions{
        Author: &object.Signature{
            Name:  "John Doe",
            Email: "john@doe.org",
            When:  time.Now(),
        },
    })
    if err != nil {
		return errors.WithMessagef(err, "couldn't git commit changes")
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

func main() {
	err := SetupRepo()
	if err != nil {
		panic(err)
	}
}