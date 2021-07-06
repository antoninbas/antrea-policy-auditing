package gitops

import (
    "time"

    "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing/object"
    "k8s.io/klog/v2"
)

func FilterCommits(r *git.Repository, author *string, since *time.Time, until *time.Time, policyResourceName *string) ([]object.Commit, error) {
    var logopts git.LogOptions
    var filteredCommits []object.Commit

    ref, err := r.Head()
    if err != nil {
        klog.ErrorS(err, "unable to get ref head from repository")
        return filteredCommits, err
    }

    logopts.From = ref.Hash()
    if !(since == nil) {
        logopts.Since = since
    }
    if !(until == nil) {
        logopts.Until = until
    }
    if !(*policyResourceName == "") {
        logopts.FileName = policyResourceName
    }

    cIter, err := r.Log(&logopts)
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
