package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

// Config represents the configuration file
type Config struct {
	Port      int      `yaml:"port"`
	DataTypes []string `yaml:"data_types"`
}

// PrometheusData represents the Prometheus-formatted data
type PrometheusData struct {
	Metric string
	Value  string
}

// Define the Prometheus metric
var (
	spDataMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spdata",
			Help: "System Profiler data",
		},
		[]string{"metric", "device", "name", "value"},
	)
)

var dynamicMetrics = sync.Map{}

// Register the Prometheus metric
func init() {
	// Register the GaugeVec with Prometheus's default registry.
	prometheus.MustRegister(spDataMetric)
}

func main() {
	// Define flags
	configFile := flag.String("config", "systemprof_exporter.yml", "Path to the config file")
	flag.Parse()

	// Load the configuration file
	configData, err := os.ReadFile(*configFile)
	if err != nil {
		fmt.Println("Error reading the config file:", err)
		os.Exit(1)
	}
	// Parse the configuration file
	var config Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		fmt.Println("Error parsing the config file:", err)
		os.Exit(1)
	}

	// Map to store outputs for each data type
	dataTypeOutputs := make(map[string]string)

	// Get the system_profiler data requested from the configuration file
	for _, dataType := range config.DataTypes {
		dataTypeOutputs[dataType] = getSystemProfilerData(dataType)
	}

	// Convert the system_profiler data to pairs
	var pairs []string
	for _, output := range dataTypeOutputs {
		dataPairs, err := ConvertJsonToPairs(output)
		if err != nil {
			fmt.Println("Error converting JSON to pairs:", err)
			os.Exit(1)
		}
		pairs = append(pairs, dataPairs...)
	}

	// Now that we have the data in pairs, we can format it as Prometheus-formatted data
	// Assuming spDataMetric is defined and registered as shown in the previous examples
	for _, pair := range pairs {
		pairSplit := strings.Split(pair, ", ")
		if len(pairSplit) < 4 {
			fmt.Println("Invalid pair format:", pair)
			continue
		}

		metricName := "spdata_" + strings.ToLower(pairSplit[0]) // Transform to Prometheus metric name convention.
		device := pairSplit[1]
		name := strings.Join(pairSplit[2:len(pairSplit)-1], "_") // Combine label names.
		value := pairSplit[len(pairSplit)-1]

		// Retrieve or create a GaugeVec for the metric.
		gaugeVec := ensureMetricExists(metricName) // This function needs to implement dynamic GaugeVec creation or retrieval.

		// Set the gauge with labels.
		gaugeVec.WithLabelValues(device, name, value).Set(1)
	}

	// Register Prometheus metrics handler
	http.Handle("/metrics", promhttp.Handler())

	// Start HTTP server to expose metrics
	addr := fmt.Sprintf(":%d", config.Port)
	fmt.Printf("Starting server on port %d\n", config.Port)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Println("Error starting HTTP server:", err)
	}

}

// get the system_profiler data requested from the configuration file
func getSystemProfilerData(dataType string) string {
	// Run system_profiler
	cmd := exec.Command("system_profiler", "-json", dataType)
	// Store the output in a variable named after the dataType above
	dataTypeOutput, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running system_profiler:", err)
		os.Exit(1)
	}
	return string(dataTypeOutput)
}

// ConvertJsonToPairs converts JSON data into pairs of data type, labels, names, and values.
func ConvertJsonToPairs(jsonData string) ([]string, error) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return nil, err
	}

	var pairs []string
	for dataType, value := range data {
		processElement(dataType, value, &pairs, dataType, "")
	}

	return pairs, nil
}

func processElement(prefix string, value interface{}, pairs *[]string, dataType, arrayIndex string) {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, val := range v {
			newPrefix := prefix
			if arrayIndex != "" {
				newPrefix = fmt.Sprintf("%s, %s, %s", prefix, arrayIndex, key)
			} else {
				newPrefix = fmt.Sprintf("%s, %s", prefix, key)
			}
			processElement(newPrefix, val, pairs, dataType, "")
		}
	case []interface{}:
		for i, item := range v {
			index := strconv.Itoa(i)
			processElement(prefix, item, pairs, dataType, index)
		}
	default:
		formattedPrefix := strings.TrimPrefix(prefix, dataType+", ")
		if arrayIndex != "" {
			*pairs = append(*pairs, fmt.Sprintf("%s, %s, %v", dataType, formattedPrefix, v))
		} else {
			*pairs = append(*pairs, fmt.Sprintf("%s, %s, %v", dataType, formattedPrefix, v))
		}
	}
}

// ensureMetricExists checks if a GaugeVec for a given metric name exists;
// if it does, it returns the existing GaugeVec.
// If it doesn't, it creates a new GaugeVec, registers it, and returns it.
func ensureMetricExists(metricName string) *prometheus.GaugeVec {
	// Attempt to load an existing GaugeVec from the map.
	metric, ok := dynamicMetrics.Load(metricName)
	if !ok {
		// If not found, create a new GaugeVec.
		newMetric := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: metricName,
				Help: fmt.Sprintf("Metric %s dynamically created", metricName),
			},
			[]string{"device", "name", "value"},
		)
		// Register the new GaugeVec with Prometheus.
		if err := prometheus.Register(newMetric); err != nil {
			fmt.Printf("Error registering metric %s: %v\n", metricName, err)
			// Handle the error, e.g., if the metric is already registered due to a race condition.
			// This might happen if another goroutine has registered the metric between our `Load` and `Register` calls.
			// In such a case, we attempt to load the metric again.
			if existingMetric, ok := dynamicMetrics.Load(metricName); ok {
				return existingMetric.(*prometheus.GaugeVec)
			}
		}
		// Store the new GaugeVec in the map.
		dynamicMetrics.Store(metricName, newMetric)
		return newMetric
	}
	// Return the loaded GaugeVec if found.
	return metric.(*prometheus.GaugeVec)
}
