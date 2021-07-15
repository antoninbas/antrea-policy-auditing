package gitops

import (
	"errors"
	"io/ioutil"
	"os"
	"sync"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	"k8s.io/klog/v2"

	billy "github.com/go-git/go-billy/v5"
	memfs "github.com/go-git/go-billy/v5/memfs"
	memory "github.com/go-git/go-git/v5/storage/memory"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CustomRepo struct {
	Repo  *git.Repository
	K8s *Kubernetes
	Mode  string
	Dir   string
	Fs    billy.Filesystem
	Mutex sync.Mutex
}

func SetupRepo(k *Kubernetes, mode string, dir string) (*CustomRepo, error) {
	if mode != "mem" && mode != "disk" {
		tmp := errors.New("mode must be memory(mem) or disk(disk)")
		klog.ErrorS(tmp, "incorrect mode")
		return nil, tmp
	}
	storer := memory.NewStorage()
	fs := memfs.New()
	cr := CustomRepo{
		K8s: k,
		Mode: mode,
		Dir:  dir,
		Fs:   fs,
	}
	r, err := createRepo(cr.K8s, mode, &cr.Dir, storer, cr.Fs)
	if err == git.ErrRepositoryAlreadyExists {
		klog.V(2).InfoS("network policy repository already exists - skipping initialization")
		return nil, nil
	} else if err != nil {
		klog.ErrorS(err, "unable to create network policy repository")
		return nil, err
	}
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	cr.Repo = r
	if err := addResources(cr.K8s, mode, cr.Dir, cr.Fs); err != nil {
		klog.ErrorS(err, "unable to add resource yamls to repository")
		return nil, err
	}
	if err := cr.AddAndCommit("audit-init", "system@audit.antrea.io", "Initial commit of existing policies"); err != nil {
		klog.ErrorS(err, "unable to add and commit existing resources to repository")
		return nil, err
	}
	klog.V(2).Infof("Repository successfully initialized at %s", cr.Dir)
	return &cr, nil
}

func createRepo(k *Kubernetes, mode string, dir *string, storer *memory.Storage, fs billy.Filesystem) (*git.Repository, error) {
	if mode == "mem" {
		r, err := git.Init(storer, fs)
		if err == git.ErrRepositoryAlreadyExists {
			klog.V(2).InfoS("network policy repository already exists - skipping initialization")
			return nil, err
		} else if err != nil {
			klog.ErrorS(err, "unable to initialize git repo")
			return nil, err
		}
		return r, nil
	}
	if *dir == "" {
		path, err := os.Getwd()
		if err != nil {
			klog.ErrorS(err, "unable to retrieve the current working directory")
			return nil, err
		}
		*dir = path
	}
	*dir += "/network-policy-repository"
	r, err := git.PlainInit(*dir, false)
	if err == git.ErrRepositoryAlreadyExists {
		return nil, err
	} else if err != nil {
		klog.ErrorS(err, "unable to initialize git repo")
		return nil, err
	}
	return r, nil
}

func addResources(k *Kubernetes, mode, dir string, fs billy.Filesystem) error {
	if mode == "disk" {
		os.Mkdir(dir+"/k8s-policies", 0700)
		os.Mkdir(dir+"/antrea-policies", 0700)
		os.Mkdir(dir+"/antrea-cluster-policies", 0700)
		os.Mkdir(dir+"/antrea-tiers", 0700)
	} else {
		fs.MkdirAll("k8s-policies", 0700)
		fs.MkdirAll("antrea-policies", 0700)
		fs.MkdirAll("antrea-cluster-policies", 0700)
		fs.MkdirAll("antrea-tiers", 0700)
	}
	if err := addK8sPolicies(k, mode, dir, fs); err != nil {
		klog.ErrorS(err, "unable to add K8s network policies to repository")
		return err
	}
	if err := addAntreaPolicies(k, mode, dir, fs); err != nil {
		klog.ErrorS(err, "unable to add Antrea network policies to repository")
		return err
	}
	if err := addAntreaClusterPolicies(k, mode, dir, fs); err != nil {
		klog.ErrorS(err, "unable to add Antrea cluster network policies to repository")
		return err
	}
	if err := addAntreaTiers(k, mode, dir, fs); err != nil {
		klog.ErrorS(err, "unable to add Antrea tiers to repository")
		return err
	}
	return nil
}

func addK8sPolicies(k *Kubernetes, mode string, dir string, fs billy.Filesystem) error {
	policies, err := k.GetK8sPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		}
		np.ObjectMeta.UID = ""
		np.ObjectMeta.Generation = 0
		np.ObjectMeta.ManagedFields = nil
		delete(np.ObjectMeta.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
		if !StringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			if mode == "disk" {
				os.Mkdir(dir+"/k8s-policies/"+np.Namespace, 0700)
			} else {
				fs.MkdirAll("k8s-policies/"+np.Namespace, 0700)
			}
		}
		path := dir + "/k8s-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		klog.V(2).Infof("Added K8s policy at network-policy-repository/k8s-policies/" + np.Namespace + "/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		if mode == "disk" {
			err = ioutil.WriteFile(path, y, 0644)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
		} else {
			newFile, err := fs.Create(path)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
			newFile.Write(y)
			newFile.Close()
		}
	}
	return nil
}

func addAntreaPolicies(k *Kubernetes, mode string, dir string, fs billy.Filesystem) error {
	policies, err := k.GetAntreaPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		np.ObjectMeta.UID = ""
		np.ObjectMeta.Generation = 0
		np.ObjectMeta.ManagedFields = nil
		delete(np.ObjectMeta.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
		if !StringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			os.Mkdir(dir+"/antrea-policies/"+np.Namespace, 0700)
		}
		path := dir + "/antrea-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		klog.V(2).Infof("Added Antrea policy at network-policy-repository/antrea-policies/" + np.Namespace + "/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		if mode == "disk" {
			err = ioutil.WriteFile(path, y, 0644)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
		} else {
			newFile, err := fs.Create(path)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
			newFile.Write(y)
			newFile.Close()
		}
	}
	return nil
}

func addAntreaClusterPolicies(k *Kubernetes, mode string, dir string, fs billy.Filesystem) error {
	policies, err := k.GetAntreaClusterPolicies()
	if err != nil {
		return err
	}
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind:       "ClusterNetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		np.ObjectMeta.UID = ""
		np.ObjectMeta.Generation = 0
		np.ObjectMeta.ManagedFields = nil
		delete(np.ObjectMeta.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
		path := dir + "/antrea-cluster-policies/" + np.Name + ".yaml"
		klog.V(2).Infof("Added Antrea cluster policy at network-policy-repository/antrea-cluster-policies/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		if mode == "disk" {
			err = ioutil.WriteFile(path, y, 0644)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
		} else {
			newFile, err := fs.Create(path)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
			newFile.Write(y)
			newFile.Close()
		}
	}
	return nil
}

func addAntreaTiers(k *Kubernetes, mode string, dir string, fs billy.Filesystem) error {
	tiers, err := k.GetAntreaTiers()
	if err != nil {
		return err
	}
	for _, tier := range tiers.Items {
		tier.TypeMeta = metav1.TypeMeta{
			Kind:       "Tier",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		tier.ObjectMeta.UID = ""
		tier.ObjectMeta.Generation = 0
		tier.ObjectMeta.ManagedFields = nil
		delete(tier.ObjectMeta.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
		path := dir + "/antrea-tiers/" + tier.Name + ".yaml"
		klog.V(2).Infof("Added Antrea tier at network-policy-repository/antrea-tiers/" + tier.Name + ".yaml")
		y, err := yaml.Marshal(&tier)
		if err != nil {
			klog.ErrorS(err, "unable to marshal tier config")
			return err
		}
		if mode == "disk" {
			err = ioutil.WriteFile(path, y, 0644)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
		} else {
			newFile, err := fs.Create(path)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
			newFile.Write(y)
			newFile.Close()
		}
	}
	return nil
}
