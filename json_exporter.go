package main

import (
	"github.com/yjbdsky/prometheus-exporter-harness/harness"
	"github.com/yjbdsky/prometheus-json-exporter/jsonexporter"
)

func main() {
	opts := harness.NewExporterOpts("json_exporter", jsonexporter.Version)
	opts.Usage = "[OPTIONS] HTTP_ENDPOINT CONFIG_PATH"
	opts.Init = jsonexporter.Init
	harness.Main(opts)
}
