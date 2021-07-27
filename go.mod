module antrea-audit

go 1.16

replace (
	// hcshim repo is modifed to add "AdditionalParams" field to HNSEndpoint struct.
	// We will use this replace before pushing the change to hcshim upstream repo.
	github.com/Microsoft/hcsshim v0.8.9 => github.com/ruicao93/hcsshim v0.8.10-0.20210114035434-63fe00c1b9aa
	// antrea/plugins/octant/go.mod also has this replacement since replace statement in dependencies
	// were ignored. We need to change antrea/plugins/octant/go.mod if there is any change here.
	github.com/contiv/ofnet => github.com/wenyingd/ofnet v0.0.0-20210318032909-171b6795a2da
)

require (
	antrea.io/antrea v1.1.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-git/go-billy/v5 v5.3.1
	github.com/go-git/go-git/v5 v5.4.2
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/apiserver v0.21.0
	k8s.io/client-go v0.21.1
	k8s.io/klog v0.3.0
	k8s.io/klog/v2 v2.8.0
)
