package webhook

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"antrea-audit/gitops"

	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/klog/v2"
)

type Change struct {
	Sha     string `json:"sha"`
	Author  string `json:"author"`
	Message string `json:"Message"`
}

type Filters struct {
	Author    string    `json:"author"`
	Since     time.Time `json:"since"`
	Until     time.Time `json:"until"`
	Resource  string    `json:"resource"`
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
}

type rollbackRequest struct {
	Tag string `json:"tag,omitempty"`
	Sha string `json:"sha,omitempty"`
}

type TagRequestType string

const (
	TagCreate TagRequestType = "create"
	TagDelete TagRequestType = "delete"
)

type tagRequest struct {
	Type   TagRequestType `json:"type,omitempty"`
	Tag    string         `json:"tag,omitempty"`
	Sha    string         `json:"sha,omitempty"`
	Author string         `json:"author,omitempty"`
	Email  string         `json:"email,omitempty"`
}

func events(w http.ResponseWriter, r *http.Request, cr *gitops.CustomRepo) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.ErrorS(err, "unable to read audit body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	klog.V(3).Infof("Audit received: %s", string(body))
	if err := cr.HandleEventList(body); err != nil {
		if err.Error() == "rollback in progress" {
			klog.ErrorS(err, "audit received during rollback")
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			klog.ErrorS(err, "unable to process audit event list")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
}

func changes(w http.ResponseWriter, r *http.Request, cr *gitops.CustomRepo) {
	defer r.Body.Close()
	if r.Method != "GET" {
		klog.Errorf("get command does not accept non-GET request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.ErrorS(err, "unable to read audit body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	klog.V(3).Infof("Filters received: %s", string(body))

	filts := r.URL.Query()
	layout := "2006-01-02T15:04:05.000Z"
	author := ""
	if len(filts["author"]) > 0 {
		author = filts["author"][0]
	}
	since := time.Time{}
	if len(filts["since"]) > 0 {
		since, _ = time.Parse(layout, filts["since"][0])
	}
	until := time.Time{}
	if len(filts["until"]) > 0 {
		until, _ = time.Parse(layout, filts["until"][0])
	}
	resource := ""
	if len(filts["resource"]) > 0 {
		resource = filts["resource"][0]
	}
	namespace := ""
	if len(filts["namespace"]) > 0 {
		namespace = filts["namespace"][0]
	}
	name := ""
	if len(filts["name"]) > 0 {
		name = filts["name"][0]
	}
	commits, err := cr.FilterCommits(&author, &since, &until, &resource, &namespace, &name)
	if err != nil {
		klog.ErrorS(err, "unable to process audit event list")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var changes []Change
	for _, c := range commits {
		chg := Change{}
		chg.Sha = c.Hash.String()
		chg.Author = c.Author.Name
		chg.Message = c.Message
		changes = append(changes, chg)
	}
	jsonstring, err := json.Marshal(changes)
	if err != nil {
		klog.ErrorS(err, "unable to marshal list of changes")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(jsonstring)
	if err != nil {
		klog.ErrorS(err, "unable to write json to response writer")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func tag(w http.ResponseWriter, r *http.Request, cr *gitops.CustomRepo) {
	defer r.Body.Close()
	if r.Method != "POST" {
		klog.Errorf("tag does not accept non-POST request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.ErrorS(err, "unable to read audit body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tagRequest := tagRequest{}
	if err := json.Unmarshal(body, &tagRequest); err != nil {
		klog.ErrorS(err, "unable to marshal request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if tagRequest.Type == TagCreate {
		signature := object.Signature{
			Name:  tagRequest.Author,
			Email: tagRequest.Email,
			When:  time.Now(),
		}
		sha, err := cr.TagCommit(tagRequest.Sha, tagRequest.Tag, &signature)
		if err != nil {
			klog.ErrorS(err, "failed to tag commit")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Commit " + sha + " tagged"))
	} else if tagRequest.Type == TagDelete {
		tag, err := cr.RemoveTag(tagRequest.Tag)
		if err != nil {
			klog.ErrorS(err, "failed to delete tag")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Tag " + tag + " deleted"))
	} else {
		klog.ErrorS(err, "unknown tag request type found")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func rollback(w http.ResponseWriter, r *http.Request, cr *gitops.CustomRepo) {
	defer r.Body.Close()
	if r.Method != "POST" {
		klog.Errorf("rollback does not accept non-POST request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.ErrorS(err, "unable to read audit body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rollbackRequest := rollbackRequest{}
	if err := json.Unmarshal(body, &rollbackRequest); err != nil {
		klog.ErrorS(err, "unable to marshal request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var commit *object.Commit
	if rollbackRequest.Tag != "" {
		commit, err = cr.TagToCommit(rollbackRequest.Tag)
	} else if rollbackRequest.Sha != "" {
		commit, err = cr.HashToCommit(rollbackRequest.Sha)
	}
	if err != nil {
		klog.ErrorS(err, "unable to convert user input into commit object")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sha, err := cr.RollbackRepo(commit)
	if err != nil {
		klog.ErrorS(err, "failed to rollback repo")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Rollback to commit " + sha + " successful"))
}

func ReceiveEvents(port string, cr *gitops.CustomRepo) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		events(w, r, cr)
	})
	http.HandleFunc("/changes", func(w http.ResponseWriter, r *http.Request) {
		changes(w, r, cr)
	})
	http.HandleFunc("/rollback", func(w http.ResponseWriter, r *http.Request) {
		rollback(w, r, cr)
	})
	http.HandleFunc("/tag", func(w http.ResponseWriter, r *http.Request) {
		tag(w, r, cr)
	})
	klog.V(2).Infof("Audit webhook server started, listening on port %s", port)
	if err := http.ListenAndServe(":"+string(port), nil); err != nil {
		klog.ErrorS(err, "Audit webhook service died")
		return err
	}
	return nil
}
