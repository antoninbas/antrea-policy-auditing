package gitops

import (
    "os"
    "time"
    "io/ioutil"

    "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing/object"
    "github.com/ghodss/yaml"
    "k8s.io/klog/v2"

    auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

func (cr *CustomRepo) AddAndCommit(username string, email string, message string) (error) {
    w, err := cr.Repo.Worktree()
    if err != nil {
        klog.ErrorS(err, "unable to get git worktree from repository")
        return err
    }
    _, err = w.Add(".")
    if err != nil {
        klog.ErrorS(err, "unable to add git change to worktree")
        return err
    }
    _, err = w.Commit(message, &git.CommitOptions{
        Author: &object.Signature{
            Name: username,
            Email: email,
            When: time.Now(),
        },
    })
    if err != nil {
        klog.ErrorS(err, "unable to commit git change to worktree")
        return err
    }
    return nil
}

func (cr *CustomRepo) modifyFile(event auditv1.Event) (error) {
    y, err := yaml.JSONToYAML(event.ResponseObject.Raw)
    if err != nil {
        klog.ErrorS(err, "unable to convert event ResponseObject from JSON to YAML format")
        return err
    }
    path := getAbsRepoPath(cr.Dir, event)
    if cr.Mode == "disk" {
        if _, err := os.Stat(path); os.IsNotExist(err) {
            os.Mkdir(path, 0700)
        }
        path += getFileName(event)
        if err := ioutil.WriteFile(path, y, 0644); err != nil {
            klog.ErrorS(err, "unable to write/update file in repository")
            return err
        }
    } else {
        path += getFileName(event)
        newfile, err := cr.Fs.Create(path)
        if err != nil {
            klog.ErrorS(err, "unable to create file at: ", "path", path)
            return err
        }
        newfile.Write(y)
        newfile.Close()
    }
    return nil
}

func (cr *CustomRepo) deleteFile(event auditv1.Event) (error) {
    w, err := cr.Repo.Worktree()
    if err != nil {
        klog.ErrorS(err, "unable to get git worktree from repository")
        return err
    }
    path := getRelRepoPath(event) + getFileName(event)
    _, err = w.Remove(path)
    if err != nil {
        klog.ErrorS(err, "unable to remove file at: ", "path", path)
        return err
    }
    return nil
}
