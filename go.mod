module github.com/slok/sloth

go 1.16

require (
	github.com/OpenSLO/oslo v0.2.2-0.20210629193748-b882029ce777
	github.com/go-playground/validator/v10 v10.9.0
	github.com/oklog/run v1.1.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.50.0
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.50.0
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.31.0
	github.com/prometheus/prometheus v1.8.2-0.20210701133801-b0944590a1c9 // v2.28.1 (Avoid semver incompatibilies with commit).
	github.com/sirupsen/logrus v1.8.1
	github.com/slok/reload v0.1.0
	github.com/spotahome/kooper/v2 v2.0.0
	github.com/stretchr/testify v1.7.0
	github.com/traefik/yaegi v0.10.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v0.22.0
)
