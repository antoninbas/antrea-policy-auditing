package gitops

import (
	"time"
    "fmt"
    "errors"
    "strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/klog/v2"
)

func (cr *CustomRepo) FilterCommits(author *string, since *time.Time, until *time.Time, resource *string, namespace *string, name *string) ([]object.Commit, error) {
	var logopts git.LogOptions
	var filteredCommits []object.Commit

	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	ref, err := cr.Repo.Head()
	if err != nil {
		klog.ErrorS(err, "unable to get ref head from repository")
		return filteredCommits, err
	}

	logopts.From = ref.Hash()
	if !since.IsZero() && since != nil {
		logopts.Since = since
	}
	if !since.IsZero() && until != nil {
		logopts.Until = until
	}

	if *resource != "" && *namespace == "" && *name != "" {
        return filteredCommits, errors.New("error (FilterCommits): cannot provide a resource without a namespace")
    } else if *resource == "" && *namespace != "" {
        resources := []string{"k8s-policies", "antrea-policies", "antrea-cluster-policies", "antrea-tiers"}
        for _, r := range resources {
            logopts.PathFilter = func(path string) bool {
                tempPath := r + "/" + *namespace + "/" + *name
                if strings.Contains(path, tempPath) {
                    return true
                }
                fmt.Println("false: " + path)
                fmt.Println(tempPath)
                return false
            }
            tempCommits, err := cr.filter(author, logopts)
            if err != nil {
                return filteredCommits, err
            }
            for _, tc := range tempCommits {
                filteredCommits = append(filteredCommits, tc)
            }
        }
	} else {
        filepath := ""
        if *resource != "" {
            filepath += *resource + "/"
        }
        if *namespace != "" {
            filepath += *namespace + "/"
        }
        if *name != "" {
            filepath += *name
        }
        if filepath != "" {
            logopts.PathFilter = func(path string) bool {
                if strings.Contains(path, filepath) {
                    return true
                }
                fmt.Println("false: " + path)
                fmt.Println(filepath)
                return false
            }
        }
        filteredCommits, err = cr.filter(author, logopts)
    }
    return filteredCommits, err
}

func (cr *CustomRepo) filter(author *string, logopts git.LogOptions) ([]object.Commit, error) {
	var filteredCommits []object.Commit
    cIter, err := cr.Repo.Log(&logopts)
	if err != nil {
		klog.ErrorS(err, "unable get logs from repository")
		return filteredCommits, err
	}

	err = cIter.ForEach(func(c *object.Commit) error {
		if *author == "" || c.Author.Name == *author {
			filteredCommits = append(filteredCommits, *c)
		}
		return nil
	})
	return filteredCommits, err
}
