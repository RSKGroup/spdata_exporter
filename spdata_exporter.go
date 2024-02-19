package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	configFile := flag.String("config", "spdata_exporter.yml", "Path to the config file")
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

	// Process the pairs and format them as Prometheus metrics
	for _, pair := range pairs {
		pair = strings.ReplaceAll(pair, "-", "_")
		pairSplit := strings.Split(pair, ", ")
		if len(pairSplit) < 4 {
			fmt.Println("Invalid pair format:", pair)
			continue
		}

		metricName := "spdata_" + strings.ToLower(pairSplit[0])
		device := pairSplit[1]
		name := strings.Join(pairSplit[2:len(pairSplit)-1], "-") // Ensure underscores are used
		valueStr := pairSplit[len(pairSplit)-1]

		// Retrieve or create a GaugeVec for the metric
		gaugeVec := ensureMetricExists(metricName)

		// Try parsing valueStr as float64 first
		valueFloat, err := strconv.ParseFloat(valueStr, 64)
		if err == nil {
			// If parsing is successful, convert to int64 and use it
			gaugeVec.WithLabelValues(device, name, strconv.FormatInt(int64(valueFloat), 10)).Set(valueFloat)
		} else {
			// If parsing fails, use valueStr as is and set gauge to 1
			gaugeVec.WithLabelValues(device, name, valueStr).Set(1)
		}
	}
	// Get the count of cvlabel labels
	cvLabelCount, err := getCVLabelCount()
	if err != nil {
		fmt.Println("Error getting cvlabel count:", err)
	}
	// Build the cvlabel count metric
	cvLabelCountMetric := ensureMetricExists("spdata_cvlabelcount")
	cvLabelCountMetric.WithLabelValues("0", "cvlabel", strconv.Itoa(cvLabelCount)).Set(float64(cvLabelCount))

	// Get the latest backup time
	latestBackupTime, err := getLatestBackupTime()
	if err != nil {
		fmt.Println("Error getting latest backup time:", err)
	}
	// Build the latest backup time metric
	latestBackupTimeMetric := ensureMetricExists("spdata_latestbackuptime")
	if latestBackupTime == "" {
		latestBackupTimeMetric.WithLabelValues("0", "latestbackup", "").Set(0)
	} else {
		latestBackupTimeMetric.WithLabelValues("0", "latestbackup", latestBackupTime).Set(1)
	}

	// Get the count of core files
	fsmCount, totalCount, err := countCoresFiles()
	// Build the core files count metric
	coreFilesCountMetric := ensureMetricExists("spdata_corefilescount")
	coreFilesCountMetric.WithLabelValues("0", "total", strconv.Itoa(totalCount)).Set(float64(totalCount))
	coreFilesCountMetric.WithLabelValues("0", "fsm", strconv.Itoa(fsmCount)).Set(float64(fsmCount))

	// Register Prometheus metrics handler and start HTTP server
	http.Handle("/metrics", promhttp.Handler())
	addr := fmt.Sprintf(":%d", config.Port)
	fmt.Printf("Starting server on port %d\n", config.Port)
	if err := http.ListenAndServe(addr, nil); err != nil {
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

// getCVLabelCount executes the command and returns the count as an int
func getCVLabelCount() (int, error) {
	cmd := exec.Command("sh", "-c", "cvlabel -l | wc -l")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(out.String()))
}

// getLatestBackupTime executes the command and returns the output as a string
func getLatestBackupTime() (string, error) {
	cmd := exec.Command("tmutil", "latestbackup", "-t")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	// strip the newline character from the output
	out = *bytes.NewBuffer(bytes.TrimRight(out.Bytes(), "\n"))
	return out.String(), nil
}

// countCoresFiles counts the number of files in the /cores directory
func countCoresFiles() (int, int, error) {
	var fsmCount, totalCount int

	err := filepath.Walk("/cores", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalCount++ // Increment total file count for every file
			// Check if file starts with the pattern 'core.fsm'
			if strings.HasPrefix(filepath.Base(path), "core.fsm") {
				fsmCount++ // Increment fsmCount if the file matches the pattern
			}
		}
		return nil
	})

	if err != nil {
		return 0, 0, err
	}

	return fsmCount, totalCount, nil
}
