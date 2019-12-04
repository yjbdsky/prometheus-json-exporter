package jsonexporter

import (
	"fmt"
	"math"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/yjbdsky/jsonpath" // Originally: "github.com/NickSardo/jsonpath"
	"github.com/yjbdsky/prometheus-exporter-harness/harness"
)

type JsonScraper interface {
	Scrape(data []byte, reg *harness.MetricRegistry) error
}

type ValueScraper struct {
	*Config
	valueJsonPath *jsonpath.Path
}

func NewValueScraper(config *Config) (JsonScraper, error) {
	valuepath, err := compilePath(config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path;path:<%s>,err:<%s>", config.Path, err)
	}

	scraper := &ValueScraper{
		Config:        config,
		valueJsonPath: valuepath,
	}
	return scraper, nil
}

func (vs *ValueScraper) parseValue(bytes []byte) (float64, error) {
	value, err := strconv.ParseFloat(string(bytes), 64)
	if err != nil {
		return -1.0, fmt.Errorf("failed to parse value as float;value:<%s>", bytes)
	}
	return value, nil
}

func (vs *ValueScraper) forTargetValue(data []byte, handle func(*jsonpath.Result)) error {
	eval, err := jsonpath.EvalPathsInBytes(data, []*jsonpath.Path{vs.valueJsonPath})
	if err != nil {
		return fmt.Errorf("failed to eval jsonpath;path:<%s>,json:<%s>", vs.valueJsonPath, data)
	}

	for {
		result, ok := eval.Next()
		if !ok {
			break
		}
		handle(result)
	}
	return nil
}

func (vs *ValueScraper) Scrape(data []byte, reg *harness.MetricRegistry) error {
	isFirst := true
	return vs.forTargetValue(data, func(result *jsonpath.Result) {
		if !isFirst {
			log.Infof("ignoring non-first value;path:<%s>", vs.valueJsonPath)
			return
		}
		isFirst = false

		var value float64
		var err error
		switch result.Type {
		case jsonpath.JsonNumber:
			value, err = vs.parseValue(result.Value)
		case jsonpath.JsonString:
			// If it is a string, lets pull off the quotes and attempt to parse it as a number
			value, err = vs.parseValue(result.Value[1 : len(result.Value)-1])
		case jsonpath.JsonNull:
			value = math.NaN()
		default:
			log.Warnf("skipping not numerical result;path:<%s>,value:<%s>",
				vs.valueJsonPath, result.Value)
			return
		}
		if err != nil {
			// Should never happen.
			log.Errorf("could not parse numerical value as float;path:<%s>,value:<%s>",
				vs.valueJsonPath, result.Value)
			return
		}

		log.Debugf("metric updated;name:<%s>,labels:<%s>,value:<%.2f>", vs.Name, vs.Labels, value)
		reg.Get(vs.Name).(*prometheus.GaugeVec).With(vs.Labels).Set(value)
	})
}

type ObjectScraper struct {
	*ValueScraper
	labelJsonPaths map[string]*jsonpath.Path
	valueJsonPaths map[string]*jsonpath.Path
}

func NewObjectScraper(config *Config) (JsonScraper, error) {
	valueScraper, err := NewValueScraper(config)
	if err != nil {
		return nil, err
	}

	labelPaths, err := compilePaths(config.Labels)
	if err != nil {
		return nil, err
	}
	valuePaths, err := compilePaths(config.Values)
	if err != nil {
		return nil, err
	}
	scraper := &ObjectScraper{
		ValueScraper:   valueScraper.(*ValueScraper),
		labelJsonPaths: labelPaths,
		valueJsonPaths: valuePaths,
	}
	return scraper, nil
}

func (obsc *ObjectScraper) newLabels() map[string]string {
	labels := make(map[string]string)
	for name, value := range obsc.Labels {
		if _, ok := obsc.labelJsonPaths[name]; !ok {
			// Static label value.
			labels[name] = value
		}
	}
	return labels
}

func (obsc *ObjectScraper) extractFirstValue(data []byte, path *jsonpath.Path) (*jsonpath.Result, error) {
	eval, err := jsonpath.EvalPathsInBytes(data, []*jsonpath.Path{path})
	if err != nil {
		return nil, fmt.Errorf("failed to eval jsonpath;err:<%s>", err)
	}

	result, ok := eval.Next()
	if !ok {
		return nil, fmt.Errorf("no value found for path")
	}
	return result, nil
}

func (obsc *ObjectScraper) Scrape(data []byte, reg *harness.MetricRegistry) error {
	return obsc.forTargetValue(data, func(result *jsonpath.Result) {
		if result.Type != jsonpath.JsonObject && result.Type != jsonpath.JsonArray {
			log.Warnf("skipping not structual result;path:<%s>,value:<%s>",
				obsc.valueJsonPath, result.Value)
			return
		}

		labels := obsc.newLabels()
		for name, path := range obsc.labelJsonPaths {
			firstResult, err := obsc.extractFirstValue(result.Value, path)
			if err != nil {
				log.Warnf("could not find value for label path;path:<%s>,json:<%s>,err:<%s>", path, result.Value, err)
				continue
			}
			value := firstResult.Value
			if firstResult.Type == jsonpath.JsonString {
				// Strip quotes
				value = value[1 : len(value)-1]
			}
			labels[name] = string(value)
		}

		for name, configValue := range obsc.Values {
			var metricValue float64
			path := obsc.valueJsonPaths[name]

			if path == nil {
				// Static value
				value, err := obsc.parseValue([]byte(configValue))
				if err != nil {
					log.Errorf("could not use configured value as float number;name:<%s>,err:<%s>", err)
					continue
				}
				metricValue = value
			} else {
				// Dynamic value
				firstResult, err := obsc.extractFirstValue(result.Value, path)
				if err != nil {
					log.Warnf("could not find value for value path;path:<%s>,json:<%s>,err:<%s>", path, result.Value, err)
					continue
				}

				var value float64
				switch firstResult.Type {
				case jsonpath.JsonNumber:
					value, err = obsc.parseValue(firstResult.Value)
				case jsonpath.JsonString:
					// If it is a string, lets pull off the quotes and attempt to parse it as a number
					value, err = obsc.parseValue(firstResult.Value[1 : len(firstResult.Value)-1])
				case jsonpath.JsonNull:
					value = math.NaN()
				default:
					log.Warnf("skipping not numerical result;path:<%s>,value:<%s>",
						obsc.valueJsonPath, result.Value)
					continue
				}
				if err != nil {
					// Should never happen.
					log.Errorf("could not parse numerical value as float;path:<%s>,value:<%s>",
						obsc.valueJsonPath, firstResult.Value)
					continue
				}
				metricValue = value
			}

			fqn := harness.MakeMetricName(obsc.Name, name)
			log.Debugf("metric updated;name:<%s>,labels:<%s>,value:<%.2f>", fqn, labels, metricValue)
			reg.Get(fqn).(*prometheus.GaugeVec).With(labels).Set(metricValue)
		}
	})
}
