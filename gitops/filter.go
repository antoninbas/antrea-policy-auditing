package gitops

import (
	"time"

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
    filepath := "/resource-auditing-repo"
	if !(*policyResourceName == "") {
		logopts.FileName = policyResourceName
	}

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
