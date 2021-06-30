module k8s.io/test-infra

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190301231843-5614ed5bae6f

// Pin all k8s.io staging repositories to kubernetes v0.17.3
// When bumping Kubernetes dependencies, you should update each of these lines
// to point to the same kubernetes v0.KubernetesMinor.KubernetesPatch version
// before running update-deps.sh.
replace (
	cloud.google.com/go => cloud.google.com/go v0.44.3
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	golang.org/x/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	k8s.io/api => k8s.io/api v0.17.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.3
	k8s.io/client-go => k8s.io/client-go v0.17.3
	k8s.io/code-generator => k8s.io/code-generator v0.17.3
)

require (
	cloud.google.com/go v0.47.0
	gitee.com/openeuler/go-gitee v0.0.0-20210625081244-7ccfc5621d02
	github.com/Azure/azure-sdk-for-go v38.0.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-autorest/autorest v0.9.6
	github.com/Azure/go-autorest/autorest/adal v0.8.2
	github.com/GoogleCloudPlatform/testgrid v0.0.7
	github.com/NYTimes/gziphandler v0.0.0-20170623195520-56545f4a5d46
	github.com/andygrunwald/go-gerrit v0.0.0-20190120104749-174420ebee6c
	github.com/antihax/optional v1.0.0
	github.com/aws/aws-k8s-tester v1.0.0
	github.com/aws/aws-sdk-go v1.30.5
	github.com/bazelbuild/buildtools v0.0.0-20190917191645-69366ca98f89
	github.com/blang/semver v3.5.1+incompatible
	github.com/bndr/gojenkins v1.0.1
	github.com/bwmarrin/snowflake v0.0.0
	github.com/clarketm/json v1.13.4
	github.com/client9/misspell v0.3.4
	github.com/containerd/containerd v1.3.3 // indirect
	github.com/djherbis/atime v1.0.0
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/frankban/quicktest v1.8.1 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/fsouza/fake-gcs-server v0.0.0-20180612165233-e85be23bdaa8
	github.com/go-bindata/go-bindata/v3 v3.1.3
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/go-openapi/spec v0.19.6
	github.com/go-openapi/swag v0.19.7 // indirect
	github.com/go-test/deep v1.0.4
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.4 // indirect
	github.com/gomodule/redigo v1.7.0
	github.com/google/go-cmp v0.4.0
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/gorilla/csrf v1.6.2
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.0
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huaweicloud/golangsdk v0.0.0-20200917123023-f1123a3e25a4
	github.com/influxdata/influxdb v0.0.0-20161215172503-049f9b42e9a5
	github.com/jinzhu/gorm v1.9.12
	github.com/jinzhu/now v1.1.1 // indirect
	github.com/klauspost/compress v1.10.2 // indirect
	github.com/klauspost/pgzip v1.2.1
	github.com/mattn/go-zglob v0.0.1
	github.com/pelletier/go-toml v1.6.0
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.9.1
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/shurcooL/githubv4 v0.0.0-20191102174205-af46314aec7b
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v0.0.6
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/tektoncd/pipeline v0.11.0
	go.opencensus.io v0.22.3 // indirect
	gocloud.dev v0.19.0
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20200302150141-5c8b2ff67527 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	golang.org/x/tools v0.0.0-20200303214625-2b0b585e22fe
	google.golang.org/api v0.15.0
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/ini.v1 v1.52.0 // indirect
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/yaml.v3 v3.0.0-20190709130402-674ba3eaed22
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v9.0.0+incompatible
	k8s.io/code-generator v0.17.3
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
	knative.dev/pkg v0.0.0-20200207155214-fef852970f43
	mvdan.cc/xurls/v2 v2.0.0
	sigs.k8s.io/controller-runtime v0.5.0
	sigs.k8s.io/yaml v1.2.0
)

go 1.13
