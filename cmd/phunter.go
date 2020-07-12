package main

import (
	"context"
	"github.com/daniel-cole/phunter/config"
	"github.com/daniel-cole/phunter/process"
	"github.com/daniel-cole/phunter/system"
	"github.com/daniel-cole/phunter/trace"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

type UTCFormatter struct {
	logrus.Formatter
}

func (u UTCFormatter) Format(e *logrus.Entry) ([]byte, error) {
	e.Time = e.Time.UTC()
	return u.Formatter.Format(e)
}

func init() {
	// check required binaries are installed
	requiredBinaries := "ps,pgrep,lsns,phpspy,bash,docker,top"
	missingBinaries := false
	for _, binary := range strings.Split(requiredBinaries, ",") {
		if !system.CheckBinaryOnPath(binary) {
			missingBinaries = true
			logrus.Errorf("Missing binary from path: %s\n", binary)
		}
	}
	if missingBinaries {
		logrus.Fatal("Please install the missing binary(s)")
	}

	switch os.Getenv("PHUNTER_LOG_LEVEL") {
	case "TRACE":
		logrus.SetLevel(logrus.TraceLevel)
	case "DEBUG":
		logrus.SetLevel(logrus.DebugLevel)
	case "WARN":
		logrus.SetLevel(logrus.WarnLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	logrus.SetReportCaller(true)
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(UTCFormatter{&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano}})}

func main() {

	var configFile string
	if configFile = os.Getenv("PHUNTER_CONFIG_FILE"); configFile == "" {
		logrus.Fatal("PHUNTER_CONFIG_FILE environment variable must be set")
	}

	logrus.Infof("attempting to load configuration from file %s", configFile)
	traceConfig := loadConfig(configFile)
	logrus.Infof("successfully loaded configuration")

	printConfig(traceConfig)

	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	listenAddress := "0.0.0.0:9000"
	server := http.Server{
		Addr:         listenAddress,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// fileserver
	fs := http.FileServer(http.Dir(traceConfig.TraceDir))
	http.Handle("/", fs)

	// healthz endpoint
	healthzHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	http.Handle("/healthz", healthzHandler)

	ticker := time.NewTicker(time.Duration(traceConfig.CheckInterval) * time.Second)

	go func() {
		<-quit
		logrus.Infof("phunter is is now stopping...")
		ticker.Stop()
		graceTime := 60 * time.Second

		ctx, cancel := context.WithTimeout(context.Background(), graceTime)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logrus.Infof("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	go func() {
		for {
			select {
			case <-ticker.C:
				go func() { checkProcesses(traceConfig) }()
			}
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatalf("Could not listen on %s: %v\n", listenAddress, err)
	}

	<-done

}

func printConfig(config config.HunterConfig) {

	logrus.Infof("RSS threshold: %d", config.ThresholdParams.RSSThreshold)
	logrus.Infof("RSS trigger count: %d", config.ThresholdParams.RSSTriggerCount)
	logrus.Infof("RSS trigger delay: %d seconds", config.ThresholdParams.RSSTriggerDelay)

	logrus.Infof("CPU threshold: %.2f", config.ThresholdParams.CPUThreshold)
	logrus.Infof("CPU trigger count: %d", config.ThresholdParams.CPUTriggerCount)
	logrus.Infof("CPU trigger delay: %d seconds", config.ThresholdParams.CPUTriggerDelay)

	logrus.Infof("check interval: %d seconds", config.CheckInterval)
	logrus.Infof("trace command: %s", config.ProcessCommand)
	logrus.Infof("application: %s", config.Application)
	logrus.Infof("application version: %s", config.ApplicationVersion)
	logrus.Infof("trace directory: %s", config.TraceDir)
	logrus.Infof("trace duration: %d seconds", config.TraceDuration)
	logrus.Infof("docker: %t", config.Docker)
	logrus.Infof("dryrun: %t", config.Dryrun)

}

func loadConfig(configFile string) config.HunterConfig {
	var traceConfig *config.HunterConfig
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		logrus.Fatal(err)
	}

	err = yaml.Unmarshal(data, &traceConfig)
	if err != nil {
		logrus.Fatal(err)
	}

	return *traceConfig
}

func checkProcesses(traceConfig config.HunterConfig) {
	logrus.Info("checking processes")
	pidList, err := process.GetPIDListByCommand(traceConfig.ProcessCommand)
	if err != nil {
		logrus.Printf("failed to get processes for %s, are any running on the system?", traceConfig.ProcessCommand)
	}

	var wg sync.WaitGroup
	for _, pid := range pidList {
		logrus.WithField("pid", pid).Debugf("checking pid")
		go func(pid int) {
			wg.Add(1)
			defer wg.Done()

			logrus.WithField("pid", pid).Debugf("checking if trace should be triggered")
			p := &process.Process{ID: pid}
			err = trace.AttemptTrace(p, process.Thresholds{
				CPU: false,
				RSS: false,
			}, traceConfig)
			if err != nil {
				logrus.WithField("pid", pid).Errorf("error when attempting to trace process: %v", err)
			}
		}(pid)
	}
	wg.Wait()
	logrus.Info("finished checking processes")
}
