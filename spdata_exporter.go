package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"

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

	// Print the raw json data
	fmt.Println(dataTypeOutputs)

	for dataType, jsonData := range dataTypeOutputs {
		prometheusMetrics := jsonToPrometheus(jsonData, dataType)
		// Now, prometheusMetrics contains the converted data for this dataType.
		// You can print or process these metrics as needed.
		for _, metric := range prometheusMetrics {
			fmt.Println(metric.Metric) // Print or further process each metric
		}
	}

	// // Register Prometheus metrics handler
	// http.Handle("/metrics", promhttp.Handler())

	// // Start HTTP server to expose metrics
	// addr := fmt.Sprintf(":%d", config.Port)
	// fmt.Printf("Starting server on port %d\n", config.Port)
	// err = http.ListenAndServe(addr, nil)
	// if err != nil {
	// 	fmt.Println("Error starting HTTP server:", err)
	// }
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

func jsonToPrometheus(jsonData string, dataType string) []PrometheusData {
	// Unmarshal JSON data into a suitable Go data structure.
	// For simplicity, let's assume it's a slice of maps (similar to previous examples).
	var data []map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		fmt.Printf("Error unmarshaling JSON for %s: %v\n", dataType, err)
		return nil
	}
	var prometheusMetrics []PrometheusData
	// Iterate over the unmarshaled data and convert to Prometheus format.
	for deviceIndex, item := range data {
		for key, value := range item {
			// Convert each item to a Prometheus metric.
			// This is a simplified example; you'll need to adjust it based on your data structure and requirements.
			metric := fmt.Sprintf(`%s{device="%d", key="%s"} 1`, dataType, deviceIndex, key)
			prometheusMetrics = append(prometheusMetrics, PrometheusData{Metric: metric, Value: fmt.Sprintf("%v", value)})
		}
	}
	return prometheusMetrics
}
