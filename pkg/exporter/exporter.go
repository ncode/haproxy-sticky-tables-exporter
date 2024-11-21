package exporter

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"sync"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/client-native/v6/runtime"
	"github.com/haproxytech/client-native/v6/runtime/options"
	"github.com/prometheus/client_golang/prometheus"
)

type HaproxyExporter struct {
	mutex           sync.Mutex
	runtimeClient   runtime.Runtime
	metrics         map[string]*prometheus.Desc
	selectedMetrics map[string]bool
	selectedTables  map[string]bool
}

func New(socketPath string, selectedMetrics, selectedTables []string) (*HaproxyExporter, error) {
	runtimeOptions := options.MasterSocket(socketPath)
	runtimeClient, err := runtime.New(context.Background(), runtimeOptions)
	if err != nil {
		return nil, fmt.Errorf("error creating configuration client: %v", err)
	}

	metrics, selectedMetricsMap := initializeMetrics(selectedMetrics)
	selectedTablesMap := initializeSelectedTables(selectedTables)

	return &HaproxyExporter{
		runtimeClient:   runtimeClient,
		metrics:         metrics,
		selectedMetrics: selectedMetricsMap,
		selectedTables:  selectedTablesMap,
	}, nil
}

func initializeMetrics(selectedMetrics []string) (map[string]*prometheus.Desc, map[string]bool) {
	allMetrics := map[string]*prometheus.Desc{
		"stick_table_used": prometheus.NewDesc(
			"haproxy_stick_table_used_size",
			"HAProxy stick tables entries in use",
			[]string{"table"},
			nil,
		),
		"stick_table_size": prometheus.NewDesc(
			"haproxy_stick_table_max_size",
			"HAProxy stick table maximum entries",
			[]string{"table"},
			nil,
		),
		"server_id": prometheus.NewDesc(
			"haproxy_stick_table_key_server_id",
			"HAProxy stick table key server association id",
			[]string{"table", "key"},
			nil,
		),
		"exp": prometheus.NewDesc(
			"haproxy_stick_table_key_expiry_milliseconds",
			"HAProxy stick table key expires in ms",
			[]string{"table", "key"},
			nil,
		),
		"use": prometheus.NewDesc(
			"haproxy_stick_table_key_use",
			"HAProxy stick table key use",
			[]string{"table", "key"},
			nil,
		),
		"gpc0": prometheus.NewDesc(
			"haproxy_stick_table_key_gpc0",
			"HAProxy stick table key general purpose counter 0",
			[]string{"table", "key"},
			nil,
		),
		"conn_cnt": prometheus.NewDesc(
			"haproxy_stick_table_key_conn_total",
			"HAProxy stick table key connection counter",
			[]string{"table", "key"},
			nil,
		),
		"conn_rate": prometheus.NewDesc(
			"haproxy_stick_table_key_conn_rate",
			"HAProxy stick table key connection rate",
			[]string{"table", "key", "period"},
			nil,
		),
		"conn_cur": prometheus.NewDesc(
			"haproxy_stick_table_key_conn_cur",
			"Number of concurrent connections for a given key",
			[]string{"table", "key"},
			nil,
		),
		"sess_cnt": prometheus.NewDesc(
			"haproxy_stick_table_key_sess_total",
			"HAProxy stick table key session counter",
			[]string{"table", "key"},
			nil,
		),
		"sess_rate": prometheus.NewDesc(
			"haproxy_stick_table_key_sess_rate",
			"HAProxy stick table key session rate",
			[]string{"table", "key", "period"},
			nil,
		),
		"http_req_cnt": prometheus.NewDesc(
			"haproxy_stick_table_key_http_req_total",
			"HAProxy stick table key HTTP request counter",
			[]string{"table", "key"},
			nil,
		),
		"http_req_rate": prometheus.NewDesc(
			"haproxy_stick_table_key_http_req_rate",
			"HAProxy stick table key HTTP request rate",
			[]string{"table", "key", "period"},
			nil,
		),
		"http_err_cnt": prometheus.NewDesc(
			"haproxy_stick_table_key_http_err_total",
			"HAProxy stick table key HTTP error counter",
			[]string{"table", "key"},
			nil,
		),
		"http_err_rate": prometheus.NewDesc(
			"haproxy_stick_table_key_http_err_rate",
			"HAProxy stick table key HTTP error rate",
			[]string{"table", "key", "period"},
			nil,
		),
		"bytes_in_cnt": prometheus.NewDesc(
			"haproxy_stick_table_key_bytes_in_total",
			"HAProxy stick table key bytes in counter",
			[]string{"table", "key"},
			nil,
		),
		"bytes_in_rate": prometheus.NewDesc(
			"haproxy_stick_table_key_bytes_in_rate",
			"HAProxy stick table key bytes in rate",
			[]string{"table", "key", "period"},
			nil,
		),
		"bytes_out_cnt": prometheus.NewDesc(
			"haproxy_stick_table_key_bytes_out_total",
			"HAProxy stick table key bytes out counter",
			[]string{"table", "key"},
			nil,
		),
		"bytes_out_rate": prometheus.NewDesc(
			"haproxy_stick_table_key_bytes_out_rate",
			"HAProxy stick table key bytes out rate",
			[]string{"table", "key", "period"},
			nil,
		),
	}

	selectedMetricsMap := make(map[string]bool)
	if len(selectedMetrics) == 0 {
		// If no specific metrics are selected, include all metrics
		for metricName := range allMetrics {
			selectedMetricsMap[metricName] = true
		}
	} else {
		// Only include selected metrics
		for _, metricName := range selectedMetrics {
			if _, exists := allMetrics[metricName]; exists {
				selectedMetricsMap[metricName] = true
			} else {
				slog.Warn("Warning: Metric %s is not recognized and will be ignored.", metricName)
			}
		}
	}

	// Filter the metrics map based on selected metrics
	filteredMetrics := make(map[string]*prometheus.Desc)
	for metricName, desc := range allMetrics {
		if selectedMetricsMap[metricName] {
			filteredMetrics[metricName] = desc
		}
	}

	return filteredMetrics, selectedMetricsMap
}

func initializeSelectedTables(selectedTables []string) map[string]bool {
	selectedTablesMap := make(map[string]bool)
	if len(selectedTables) == 0 {
		// If no specific tables are selected, include all tables
		selectedTablesMap["*"] = true
	} else {
		// Only include selected tables
		for _, tableName := range selectedTables {
			selectedTablesMap[tableName] = true
		}
	}
	return selectedTablesMap
}

func (e *HaproxyExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range e.metrics {
		ch <- metric
	}
}

func (e *HaproxyExporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	slog.Info("Collecting HAProxy metrics...")

	stickTables, err := e.runtimeClient.ShowTables()
	if err != nil {
		slog.Error("Error fetching stick tables: %v", err)
		return
	}

	slog.Info("Number of stick tables found: %d", len(stickTables))

	for _, table := range stickTables {
		// Check if the table is selected
		if !e.isTableSelected(table.Name) {
			continue
		}
		e.collectStickTableMetrics(table, ch)
	}
}

func (e *HaproxyExporter) isTableSelected(tableName string) bool {
	if e.selectedTables["*"] {
		return true
	}
	return e.selectedTables[tableName]
}

func (e *HaproxyExporter) collectStickTableMetrics(table *models.StickTable, ch chan<- prometheus.Metric) {
	if table == nil {
		return
	}

	slog.Info("Collecting metrics for stick table: %s", table.Name)

	// Collect table-level metrics if they are selected
	if e.selectedMetrics["stick_table_used"] {
		used := float64(*table.Used)
		ch <- prometheus.MustNewConstMetric(
			e.metrics["stick_table_used"],
			prometheus.GaugeValue,
			used,
			table.Name,
		)
	}

	if e.selectedMetrics["stick_table_size"] {
		size := float64(*table.Size)
		ch <- prometheus.MustNewConstMetric(
			e.metrics["stick_table_size"],
			prometheus.GaugeValue,
			size,
			table.Name,
		)
	}

	// Prepare filters for GetTableEntries
	var filters []string
	for metric := range e.selectedMetrics {
		// Exclude table-level metrics
		if metric == "stick_table_used" || metric == "stick_table_size" {
			continue
		}
		// Map metric names to data types as per the StickTableEntry struct fields
		filters = append(filters, metric)
	}
	// Remove duplicates and optimize filters
	filters = removeDuplicateFilters(filters)

	// Collect entries from the stick table using filters
	entries, err := e.runtimeClient.GetTableEntries(table.Name, filters, "")
	if err != nil {
		slog.Error("Error fetching entries for table %s: %v", table.Name, err)
		return
	}

	for _, entry := range entries {
		e.collectStickTableEntryMetrics(table.Name, entry, ch)
	}
}

func (e *HaproxyExporter) collectStickTableEntryMetrics(tableName string, entry *models.StickTableEntry, ch chan<- prometheus.Metric) {
	if entry == nil || entry.Key == "" {
		return
	}

	slog.Info("Processing entry with key: %s", entry.Key)

	v := reflect.ValueOf(entry).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		metricName := field.Name

		// Skip ID and Key fields
		if metricName == "ID" || metricName == "Key" {
			continue
		}

		// Check if the metric is selected
		if !e.selectedMetrics[metricName] {
			continue
		}

		// Get the value if it's not nil
		var value float64
		switch fieldValue.Kind() {
		case reflect.Ptr:
			if fieldValue.IsNil() {
				continue
			}
			switch fieldValue.Elem().Kind() {
			case reflect.Int64:
				value = float64(fieldValue.Elem().Int())
			default:
				continue
			}
		case reflect.Bool:
			if !fieldValue.Bool() {
				continue
			}
			value = 1
		default:
			continue
		}

		// Collect the metric
		desc, exists := e.metrics[metricName]
		if !exists {
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			value,
			tableName,
			entry.Key,
		)
	}
}

func removeDuplicateFilters(filters []string) []string {
	filterMap := make(map[string]struct{})
	for _, filter := range filters {
		filterMap[filter] = struct{}{}
	}
	uniqueFilters := make([]string, 0, len(filterMap))
	for filter := range filterMap {
		uniqueFilters = append(uniqueFilters, filter)
	}
	return uniqueFilters
}

func FindStatsSocket(configFile string) (string, error) {
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return "", fmt.Errorf("error reading HAProxy config file: %v", err)
	}

	configContent := string(configData)
	re := regexp.MustCompile(`(?m)^\s*stats\s+socket\s+([^\s]+)`)
	matches := re.FindStringSubmatch(configContent)
	if len(matches) < 2 {
		return "", fmt.Errorf("stats socket not found in %s", configFile)
	}

	socketPath := matches[1]
	return socketPath, nil
}

func Start() error {
	socketPath, err := FindStatsSocket(viper.GetString("haproxy-config"))
	if err != nil {
		return fmt.Errorf("error finding stats socket: %w", err)
	}
	exp, err := New(socketPath, viper.GetStringSlice("metrics"), viper.GetStringSlice("tables"))
	if err != nil {
		return fmt.Errorf("error initializing exporter: %w", err)
	}

	prometheus.MustRegister(exp)
	http.Handle("/metrics", promhttp.Handler())
	port := viper.GetInt("port")
	serverAddr := fmt.Sprintf(":%d", port)
	slog.Info("Listening on %s", serverAddr)
	return http.ListenAndServe(serverAddr, nil)
}
