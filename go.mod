module github.com/slok/sloth

go 1.16

require (
	github.com/OpenSLO/oslo v0.2.2-0.20210629193748-b882029ce777
	github.com/go-playground/validator/v10 v10.9.0
	github.com/oklog/run v1.1.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.51.1
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.51.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.31.1
	github.com/prometheus/prometheus v1.8.2-0.20211001113022-b30db03f3565 // v2.30.2 (Avoid semver incompatibilies with commit).
	github.com/sirupsen/logrus v1.8.1
	github.com/slok/reload v0.1.0
	github.com/spotahome/kooper/v2 v2.1.0
	github.com/stretchr/testify v1.7.0
	github.com/traefik/yaegi v0.10.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
)
