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
	Repo           *git.Repository
	K8s            *Kubernetes
	RollbackMode   bool
	StorageMode    string
	ServiceAccount string
	Dir            string
	Fs             billy.Filesystem
	Mutex          sync.Mutex
}

func SetupRepo(k *Kubernetes, mode string, dir string) (*CustomRepo, error) {
	if mode != "mem" && mode != "disk" {
		tmp := errors.New("mode must be memory(mem) or disk(disk)")
		klog.ErrorS(tmp, "incorrect mode")
		return nil, tmp
	}
	storer := memory.NewStorage()
	fs := memfs.New()
	svcAcct := "system:serviceaccount:" + GetAuditPodNamespace() + GetAuditServiceAccount()
	cr := CustomRepo{
		K8s:            k,
		RollbackMode:   false,
		StorageMode:    mode,
		ServiceAccount: svcAcct,
		Dir:            dir,
		Fs:             fs,
	}
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	r, err := cr.createRepo(storer)
	if err == git.ErrRepositoryAlreadyExists {
		klog.V(2).InfoS("network policy repository already exists - skipping initialization")
		cr.Repo = r
		return &cr, nil
	} else if err != nil {
		klog.ErrorS(err, "unable to create network policy repository")
		return nil, err
	}
	cr.Repo = r
	if err := cr.addResources(); err != nil {
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

func (cr *CustomRepo) createRepo(storer *memory.Storage) (*git.Repository, error) {
	if cr.StorageMode == "mem" {
		r, err := git.Init(storer, cr.Fs)
		if err == git.ErrRepositoryAlreadyExists {
			klog.V(2).InfoS("network policy repository already exists - skipping initialization")
			return nil, err
		} else if err != nil {
			klog.ErrorS(err, "unable to initialize git repo")
			return nil, err
		}
		return r, nil
	}
	if cr.Dir == "" {
		path, err := os.Getwd()
		if err != nil {
			klog.ErrorS(err, "unable to retrieve the current working directory")
			return nil, err
		}
		cr.Dir = path
	}
	cr.Dir += "/network-policy-repository"
	r, err := git.PlainInit(cr.Dir, false)
	if err == git.ErrRepositoryAlreadyExists {
		klog.V(2).InfoS("network policy repository already exists - skipping initialization")
		r, err := git.PlainOpen(cr.Dir)
		if err != nil {
			klog.ErrorS(err, "unable to retrieve existing repository")
			return nil, err
		}
		return r, git.ErrRepositoryAlreadyExists
	} else if err != nil {
		klog.ErrorS(err, "unable to initialize git repo")
		return nil, err
	}
	return r, nil
}

func (cr *CustomRepo) addResources() error {
	if cr.StorageMode == "disk" {
		os.Mkdir(cr.Dir+"/k8s-policies", 0700)
		os.Mkdir(cr.Dir+"/antrea-policies", 0700)
		os.Mkdir(cr.Dir+"/antrea-cluster-policies", 0700)
		os.Mkdir(cr.Dir+"/antrea-tiers", 0700)
	} else {
		cr.Fs.MkdirAll("k8s-policies", 0700)
		cr.Fs.MkdirAll("antrea-policies", 0700)
		cr.Fs.MkdirAll("antrea-cluster-policies", 0700)
		cr.Fs.MkdirAll("antrea-tiers", 0700)
	}
	if err := cr.addK8sPolicies(); err != nil {
		klog.ErrorS(err, "unable to add K8s network policies to repository")
		return err
	}
	if err := cr.addAntreaPolicies(); err != nil {
		klog.ErrorS(err, "unable to add Antrea network policies to repository")
		return err
	}
	if err := cr.addAntreaClusterPolicies(); err != nil {
		klog.ErrorS(err, "unable to add Antrea cluster network policies to repository")
		return err
	}
	if err := cr.addAntreaTiers(); err != nil {
		klog.ErrorS(err, "unable to add Antrea tiers to repository")
		return err
	}
	return nil
}

func (cr *CustomRepo) addK8sPolicies() error {
	policies, err := cr.K8s.GetK8sPolicies()
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
			if cr.StorageMode == "disk" {
				os.Mkdir(cr.Dir+"/k8s-policies/"+np.Namespace, 0700)
			} else {
				cr.Fs.MkdirAll("k8s-policies/"+np.Namespace, 0700)
			}
		}
		path := cr.Dir + "/k8s-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		klog.V(2).Infof("Added K8s policy at network-policy-repository/k8s-policies/" + np.Namespace + "/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		if cr.StorageMode == "disk" {
			err = ioutil.WriteFile(path, y, 0644)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
		} else {
			newFile, err := cr.Fs.Create(path)
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

func (cr *CustomRepo) addAntreaPolicies() error {
	policies, err := cr.K8s.GetAntreaPolicies()
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
			os.Mkdir(cr.Dir+"/antrea-policies/"+np.Namespace, 0700)
		}
		path := cr.Dir + "/antrea-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		klog.V(2).Infof("Added Antrea policy at network-policy-repository/antrea-policies/" + np.Namespace + "/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		if cr.StorageMode == "disk" {
			err = ioutil.WriteFile(path, y, 0644)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
		} else {
			newFile, err := cr.Fs.Create(path)
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

func (cr *CustomRepo) addAntreaClusterPolicies() error {
	policies, err := cr.K8s.GetAntreaClusterPolicies()
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
		path := cr.Dir + "/antrea-cluster-policies/" + np.Name + ".yaml"
		klog.V(2).Infof("Added Antrea cluster policy at network-policy-repository/antrea-cluster-policies/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		if cr.StorageMode == "disk" {
			err = ioutil.WriteFile(path, y, 0644)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
		} else {
			newFile, err := cr.Fs.Create(path)
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

func (cr *CustomRepo) addAntreaTiers() error {
	tiers, err := cr.K8s.GetAntreaTiers()
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
		path := cr.Dir + "/antrea-tiers/" + tier.Name + ".yaml"
		klog.V(2).Infof("Added Antrea tier at network-policy-repository/antrea-tiers/" + tier.Name + ".yaml")
		y, err := yaml.Marshal(&tier)
		if err != nil {
			klog.ErrorS(err, "unable to marshal tier config")
			return err
		}
		if cr.StorageMode == "disk" {
			err = ioutil.WriteFile(path, y, 0644)
			if err != nil {
				klog.ErrorS(err, "unable to write policy config to file")
				return err
			}
		} else {
			newFile, err := cr.Fs.Create(path)
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
