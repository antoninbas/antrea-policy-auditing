package webhook

import (
	"io/ioutil"
	"net/http"

	"antrea-audit/git-manager/gitops"

	"k8s.io/klog/v2"
)

func ReceiveEvents(dir string, port string) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			klog.ErrorS(err, "unable to read audit body")
			w.WriteHeader(http.StatusBadRequest)
		}
		klog.V(3).Infof("Audit received: %s", string(body))
		if err := gitops.HandleEventList(dir, body); err != nil {
			klog.ErrorS(err, "unable to process audit event list")
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	klog.V(2).Infof("Audit webhook server started, listening on port %s", port)
	if err := http.ListenAndServe(":"+string(port), nil); err != nil {
		klog.ErrorS(err, "Audit webhook service died")
		return err
	}
	return nil
}
