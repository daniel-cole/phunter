package trace

import (
	"errors"
	"fmt"
	"github.com/daniel-cole/phunter/config"
	"github.com/daniel-cole/phunter/process"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

var (
	mu          sync.Mutex
	tracePIDMap map[int]bool
)

func init() {
	tracePIDMap = make(map[int]bool)
}

// AttemptTrace will trace the specified process ID for the specified duration
func AttemptTrace(p process.ProcessInterface, thresholds process.Thresholds, config config.HunterConfig) error {

	pid := p.GetID()

	mu.Lock()

	// check if pid already has a running trace
	logrus.WithField("pid", pid).Tracef("checking if trace already running")
	if tracePIDMap[pid] {
		mu.Unlock()
		logrus.WithField("pid", pid).Tracef("trace already running")
		return errors.New(fmt.Sprintf("trace already running"))
	}

	// no trace running, safe to start new trace
	logrus.WithField("pid", pid).Debugf("checking if trace should be triggered")
	tracePIDMap[pid] = true
	mu.Unlock()

	if thresholdType, trigger := process.CheckThresholdTriggers(p, thresholds, config.ThresholdParams); trigger {
		logrus.WithField("pid", pid).Infof("%s trigger fired trace", thresholdType)
		err := runTrace(p, config)
		if err != nil {
			logrus.WithField("pid", pid).Error("failed to run trace")
			unlockTrace(pid)
			return err
		}
	}

	unlockTrace(pid)
	logrus.WithField("pid", pid).Trace("finished trace")
	return nil
}

func unlockTrace(pid int) {
	mu.Lock()
	tracePIDMap[pid] = false
	mu.Unlock()
}

func runTrace(p process.ProcessInterface, config config.HunterConfig) error {
	switch config.Application {
	case "php":
		err := runPHPTrace(p,
			config.ApplicationVersion,
			config.TraceDuration,
			config.TraceDir,
			config.Docker,
			config.Timezone,
			config.Dryrun,
			config.PHPSpyConfig,
		)
		if err != nil {
			logrus.WithField("pid", p.GetID()).Errorf("failed to run trace %v", err)
			return err
		}
	default:
		logrus.Fatal("unsupported application")
	}
	return nil
}

func runPHPTrace(p process.ProcessInterface, phpVersion string, traceDuration int, traceDir string, docker bool,
	timezone string, dryrun bool, spyConfig config.PHPSpyConfig) error {

	var err error
	var containerName string
	var traceFileName string

	pid := p.GetID()
	loc, _ := time.LoadLocation(timezone)
	timestamp := time.Now().In(loc).Format(time.RFC3339)
	// attempt to get the container name if docker has been set to true
	if docker {
		containerName, err = p.FindContainerName()
		if err != nil {
			logrus.WithField("pid", pid).Error("failed to get container name for process")
			return err
		}
		traceFileName = fmt.Sprintf("%s-%d-%s.trace", containerName, pid, timestamp)
	} else {
		traceFileName = fmt.Sprintf("%d-%s.trace", pid, timestamp)
	}

	logrus.WithField("pid", pid).Infof("trace will be written to %s", traceFileName)

	err = os.MkdirAll(traceDir, 0600)
	if err != nil {
		logrus.WithField("pid", pid).Errorf("failed to create trace directory: %s", traceDir)
		return err
	}
	traceFile, err := os.Create(fmt.Sprintf("%s/%s", traceDir, traceFileName))
	if err != nil {
		return err
	}

	var traceCommand *exec.Cmd
	if dryrun {
		traceCommand = exec.Command("echo", "dryrun")
	} else {
		traceCommand = exec.Command("phpspy",
			fmt.Sprintf("-V%s", phpVersion), "-p", strconv.Itoa(pid),
			"-T", spyConfig.Threads,
			"-s", spyConfig.Sleep,
			"-H", spyConfig.Rate,
			"-l", spyConfig.Limit,
		)
	}

	logrus.WithField("pid", pid).Debugf("trace command: %s", traceCommand)

	traceCommand.Stdout = traceFile
	traceCommand.Stderr = traceFile

	if err := traceCommand.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		done <- traceCommand.Wait()
	}()
	select {
	case <-time.After(time.Duration(traceDuration) * time.Second):
		if err := traceCommand.Process.Kill(); err != nil {
			logrus.WithField("pid", pid).Error("failed to kill process after specified trace duration")
			return err
		}
		logrus.WithField("pid", pid).Infof("trace stopped after %d seconds", traceDuration)
		logrus.WithField("pid", pid).Infof("trace written to %s", traceFileName)
		return nil
	case err := <-done:
		if err != nil {
			logrus.WithField("pid", pid).Error("unexpected tracing error")
			return err
		}
		logrus.WithField("pid", pid).Error("trace finished before elasped trace duration")
	}
	return nil
}
