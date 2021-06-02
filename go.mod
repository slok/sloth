module github.com/slok/sloth

go 1.16

require (
	github.com/go-playground/validator/v10 v10.6.1
	github.com/oklog/run v1.1.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.48.1
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.48.0
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/common v0.25.0
	github.com/prometheus/prometheus v1.8.2-0.20210512173212-24c9b61221f7 // v2.27.0 (Avoid semver incompatibilies with commit).
	github.com/sirupsen/logrus v1.8.1
	github.com/spotahome/kooper/v2 v2.0.0-rc.2
	github.com/stretchr/testify v1.7.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
)
