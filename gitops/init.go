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
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type StorageModeType string

const (
	StorageModeDisk     StorageModeType = "Disk"
	StorageModeInMemory StorageModeType = "InMemory"
)

type CustomRepo struct {
	Repo           *git.Repository
	K8s            *K8sClient
	RollbackMode   bool
	StorageMode    StorageModeType
	ServiceAccount string
	Dir            string
	Fs             billy.Filesystem
	Mutex          sync.Mutex
}

func SetupRepo(k8s *K8sClient, mode StorageModeType, dir string) (*CustomRepo, error) {
	if mode != StorageModeDisk && mode != StorageModeInMemory {
		tmp := errors.New("mode must be memory(mem) or disk(disk)")
		klog.ErrorS(tmp, "incorrect mode")
		return nil, tmp
	}
    if mode == "mem" && dir != "" {
        return git.ErrRepositoryAlreadyExists
    }
	storer := memory.NewStorage()
	fs := memfs.New()
	svcAcct := "system:serviceaccount:" + GetAuditPodNamespace() + ":" + GetAuditServiceAccount()
	cr := CustomRepo{
		K8s:            k8s,
		RollbackMode:   false,
		StorageMode:    mode,
		ServiceAccount: svcAcct,
		Dir:            dir,
		Fs:             fs,
	}
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	r, err := cr.createRepo(storer)
	cr.Repo = r
	if err == git.ErrRepositoryAlreadyExists {
		klog.V(2).InfoS("network policy repository already exists - skipping initialization")
		return &cr, nil
	} else if err != nil {
		klog.ErrorS(err, "unable to create network policy repository")
		return nil, err
	}
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
	if cr.StorageMode == StorageModeInMemory {
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
		if path != "/" {
			cr.Dir = path
		}
	}
	cr.Dir += "/resource-auditing-repo"
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
	if cr.StorageMode == StorageModeDisk {
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
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicyList",
	})
	policies, err := cr.K8s.ListResource(list)
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		clearFields(&np)
		name := np.GetName()
		namespace := np.GetNamespace()
		if !stringInSlice(namespace, namespaces) {
			namespaces = append(namespaces, namespace)
			if cr.StorageMode == StorageModeDisk {
				os.Mkdir(cr.Dir+"/k8s-policies/"+namespace, 0700)
			} else {
				cr.Fs.MkdirAll("k8s-policies/"+namespace, 0700)
			}
		}
		path := computePath(cr.Dir, "k8s-policies", namespace, name+".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		if err := cr.writeFileToPath(path, y); err != nil {
			klog.Errorf("unable to write k8s policy yaml to path %s", path)
			return err
		}
		klog.V(2).Infof("Added K8s policy at resource-auditing-repo/k8s-policies/" + namespace + "/" + name + ".yaml")
	}
	return nil
}

func (cr *CustomRepo) addAntreaPolicies() error {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicyList",
	})
	policies, err := cr.K8s.ListResource(list)
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		clearFields(&np)
		name := np.GetName()
		namespace := np.GetNamespace()
		if !stringInSlice(namespace, namespaces) {
			namespaces = append(namespaces, namespace)
			os.Mkdir(cr.Dir+"/antrea-policies/"+namespace, 0700)
		}
		path := computePath(cr.Dir, "antrea-policies", namespace, name+".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		if err := cr.writeFileToPath(path, y); err != nil {
			klog.Errorf("unable to write antrea policy yaml to path %s", path)
			return err
		}
		klog.V(2).Infof("Added Antrea policy at resource-auditing-repo/antrea-policies/" + namespace + "/" + name + ".yaml")
	}
	return nil
}

func (cr *CustomRepo) addAntreaClusterPolicies() error {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "ClusterNetworkPolicyList",
	})
	policies, err := cr.K8s.ListResource(list)
	if err != nil {
		return err
	}
	for _, np := range policies.Items {
		clearFields(&np)
		name := np.GetName()
		path := computePath(cr.Dir, "antrea-cluster-policies", "", name+".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		if err := cr.writeFileToPath(path, y); err != nil {
			klog.Errorf("unable to write antrea cluster policy yaml to path %s", path)
			return err
		}
		klog.V(2).Infof("Added Antrea cluster policy at resource-auditing-repo/antrea-cluster-policies/" + name + ".yaml")
	}
	return nil
}

func (cr *CustomRepo) addAntreaTiers() error {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "TierList",
	})
	tiers, err := cr.K8s.ListResource(list)
	if err != nil {
		return err
	}
	for _, tier := range tiers.Items {
		clearFields(&tier)
		name := tier.GetName()
		path := computePath(cr.Dir, "antrea-tiers", "", name+".yaml")
		y, err := yaml.Marshal(&tier)
		if err != nil {
			klog.ErrorS(err, "unable to marshal tier config")
			return err
		}
		if err := cr.writeFileToPath(path, y); err != nil {
			klog.Errorf("unable to write tier yaml to path %s", path)
			return err
		}
		klog.V(2).Infof("Added Antrea tier at resource-auditing-repo/antrea-tiers/" + name + ".yaml")
	}
	return nil
}

func (cr *CustomRepo) writeFileToPath(path string, yaml []byte) error {
	if cr.StorageMode == StorageModeDisk {
		err := ioutil.WriteFile(path, yaml, 0644)
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
		newFile.Write(yaml)
		newFile.Close()
	}
	return nil
}
