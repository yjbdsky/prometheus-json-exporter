module github.com/yjbdsky/prometheus-json-exporter

go 1.13

replace (
	github.com/Sirupsen/logrus v1.2.0 => github.com/sirupsen/logrus v1.2.0
	github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.2.0
)

require (
	github.com/Sirupsen/logrus v1.4.2
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/prometheus/client_golang v1.2.1
	github.com/prometheus/client_model v0.0.0-20191202183732-d1d2010b5bee // indirect
	github.com/prometheus/procfs v0.0.8 // indirect
	github.com/urfave/cli v1.22.2
	github.com/yjbdsky/jsonpath v0.0.0-20160208140654-5c448ebf9735
	github.com/yjbdsky/prometheus-exporter-harness v1.2.0 // direct
	golang.org/x/crypto v0.0.0-20191202143827-86a70503ff7e // indirect
	golang.org/x/sys v0.0.0-20191204072324-ce4227a45e2e // indirect
	gopkg.in/yaml.v2 v2.2.7
)
