package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"strconv"
)

type ProcessGaugeCollect struct {
	// 进程 cpu
	CPUGaugeCollect *prometheus.Desc
	// 进程 memory
	MemoryGaugeCollect *prometheus.Desc
	// 进程 文件打开数
	OpenFilesGaugeCollect *prometheus.Desc
}

func NewProcessGaugeCollect() *ProcessGaugeCollect {
	return &ProcessGaugeCollect{
		CPUGaugeCollect: prometheus.NewDesc(
			"describe_node_process_cpu_info",
			"node process cpu monitor",
			[]string{"name", "pid", "cmd", "user"},
			nil,
		),

		MemoryGaugeCollect: prometheus.NewDesc(
			"describe_node_process_memory_info",
			"node process memory monitor",
			[]string{"name", "pid", "cmd", "user"},
			nil,
		),

		OpenFilesGaugeCollect: prometheus.NewDesc(
			"describe_node_process_openfiles_info",
			"node process openfiles monitor",
			[]string{"name", "pid", "cmd", "user"},
			nil,
		),
	}
}

func (pgc ProcessGaugeCollect) Describe(docs chan<- *prometheus.Desc) {
	docs <- pgc.CPUGaugeCollect
	docs <- pgc.MemoryGaugeCollect
	docs <- pgc.OpenFilesGaugeCollect
}

func (pgc ProcessGaugeCollect) Collect(metric chan<- prometheus.Metric) {
	processes, err := process.Processes()
	if err != nil {
		log.Fatal(err)
		return
	}

	for _, proc := range processes {
		registerMetric(proc, proc.Pid, metric, pgc)
	}
}

func registerMetric(proc *process.Process, pid int32, metric chan<- prometheus.Metric, pgc ProcessGaugeCollect) {
	user, err := proc.Username()
	if err != nil {
		return
	}

	if user != "root" {
		return
	}

	name, err := proc.Name()
	if err != nil {
		return
	}

	cpu, err := getProcCPU(proc)
	if err != nil {
		return
	}

	mem, err := getProcMEM(proc)
	if err != nil {
		return
	}

	openFiles, err := getProcOpenFileNumber(proc)
	if err != nil {
		return
	}

	cmdline, err := proc.Cmdline()
	if err != nil {
		return
	}

	if cpu != 0 {
		metric <- prometheus.MustNewConstMetric(
			pgc.CPUGaugeCollect,
			prometheus.GaugeValue,
			cpu,
			name,
			strconv.Itoa(int(pid)),
			cmdline,
			user,
		)
	}

	if mem != 0 {
		metric <- prometheus.MustNewConstMetric(
			pgc.MemoryGaugeCollect,
			prometheus.GaugeValue,
			mem,
			name,
			strconv.Itoa(int(pid)),
			cmdline,
			user,
		)
	}

	if openFiles != 0 {
		metric <- prometheus.MustNewConstMetric(
			pgc.OpenFilesGaugeCollect,
			prometheus.GaugeValue,
			float64(openFiles),
			name,
			strconv.Itoa(int(pid)),
			cmdline,
			user,
		)
	}
}

func main() {
	procMetric := NewProcessGaugeCollect()
	registry := prometheus.NewRegistry()
	registry.MustRegister(procMetric)

	http.HandleFunc("/metrics", func(writer http.ResponseWriter, request *http.Request) {
		promhttp.HandlerFor(registry,
			promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError}).ServeHTTP(writer, request)
	})

	logrus.Info("Service started!")

	if err := http.ListenAndServe(":"+"9002", nil); err != nil {
		log.Fatal(err)
	}
}

func getProcCPU(proc *process.Process) (float64, error) {
	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		logrus.Errorf(err.Error())
		return 0, err
	}

	return cpuPercent, nil
}

func getProcMEM(proc *process.Process) (float64, error) {
	procMem, err := proc.MemoryInfo()
	if err != nil {
		logrus.Errorf(err.Error())
		return 0, err
	}

	nodeMem, err := mem.VirtualMemory()
	if err != nil {
		logrus.Errorf(err.Error())
		return 0, err
	}

	// 乘以 100，得出正确的百分比
	procMemBytes := procMem.RSS * 100
	return float64(procMemBytes) / float64(nodeMem.Total), nil
}

func getProcOpenFileNumber(proc *process.Process) (int, error) {
	files, err := proc.OpenFiles()
	if err != nil {
		return 0, err
	}

	return len(files), nil
}
