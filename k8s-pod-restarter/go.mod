module github.com/viniciushammett/k8s-pod-restarter

go 1.22

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/prometheus/client_golang v1.19.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/zerolog v1.33.0
	github.com/spf13/cobra v1.8.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.30.3
	k8s.io/apimachinery v0.30.3
	k8s.io/client-go v0.30.3
)
