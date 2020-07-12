package process

import (
	"github.com/sirupsen/logrus"
	"time"
)

const (
	ThresholdTypeCPU = "CPU"
	ThresholdTypeRSS = "RSS"
)

var thresholdTypes = []string{ThresholdTypeRSS, ThresholdTypeCPU}

type ThresholdParams struct {
	CPUThreshold    float64 `yaml:"cpu_threshold"`     // the threshold at which to trigger a trace based on CPU
	CPUTriggerCount int     `yaml:"cpu_trigger_count"` // number of times CPU threshold should be reached before triggering a trace
	CPUTriggerDelay int     `yaml:"cpu_trigger_delay"` // how long to wait before the next CPU check when CPU trigger count > 1
	RSSThreshold    int64   `yaml:"rss_threshold"`     // the threshold on at which to trigger a trace based on RSS
	RSSTriggerCount int     `yaml:"rss_trigger_count"` // number of times RSS threshold should be reached before triggering a trace
	RSSTriggerDelay int     `yaml:"rss_trigger_delay"` // how long to wait before the next RSS check when RSS trigger count > 1
}

type Thresholds struct {
	CPU bool
	RSS bool
}

func CheckThresholds(p ProcessInterface, thresholdParams ThresholdParams) (Thresholds, error) {
	pid := p.GetID()
	var thresholdReached Thresholds
	cpuThresholdReached, err := checkCPUThreshold(p, thresholdParams)
	if err != nil {
		return Thresholds{}, err
	}
	if cpuThresholdReached {
		logrus.WithField("pid", pid).Infof("CPU threshold reached")
		thresholdReached.CPU = true
	}

	rssThresholdReached, err := checkRSSThreshold(p, thresholdParams)
	if err != nil {
		return Thresholds{}, err
	}
	if rssThresholdReached {
		logrus.WithField("pid", pid).Infof("RSS threshold reached")
		thresholdReached.RSS = true
	}
	return thresholdReached, err
}

// CheckThresholdTriggers checks to see if any of the thresholds for a particular process have been exceeded
// this takes into account the specified amount of times a threshold should be reached before triggerring
// returns the threshold type (RSS or CPU) and true if the process has reached the specified threshold
func CheckThresholdTriggers(p ProcessInterface, thresholds Thresholds, thresholdParams ThresholdParams) (string, bool) {

	trigger := make(chan string)
	abort := make(chan struct{})
	for _, thresholdType := range thresholdTypes {
		go func(thresholdType string) {
			if checkTrigger(p, thresholds, thresholdParams, thresholdType) {
				trigger <- thresholdType
			} else {
				abort <- struct{}{}
			}
		}(thresholdType)
	}

	checkedTriggers := 0
	numTriggerChecks := len(thresholdTypes)
	for checkedTriggers < numTriggerChecks {
		select {
		case thresholdType := <-trigger:
			return thresholdType, true
		case <-abort:
			checkedTriggers++
		}
	}
	return "", false
}

func checkCPUThreshold(p ProcessInterface, thresholdParams ThresholdParams) (bool, error) {
	pid := p.GetID()
	pidCPU, err := p.GetCPU()
	if err != nil {
		return false, err
	}

	if pidCPU > thresholdParams.CPUThreshold {
		logrus.WithField("pid", pid).Infof("CPU above threshold: %2.f/%2.f", pidCPU, thresholdParams.CPUThreshold)
		return true, nil
	}
	logrus.WithField("pid", pid).Tracef("current process CPU: %2.f", pidCPU)
	return false, nil
}

func checkRSSThreshold(p ProcessInterface, thresholdParams ThresholdParams) (bool, error) {
	pid := p.GetID()

	pidRSS, err := p.GetRSS()
	if err != nil {
		return false, err
	}
	if pidRSS > thresholdParams.RSSThreshold {
		logrus.WithField("pid", pid).Infof("RSS above threshold: %d/%d", pidRSS, thresholdParams.RSSThreshold)
		return true, nil
	}
	logrus.WithField("pid", pid).Tracef("current process RSS: %d", pidRSS)
	return false, nil
}

func checkTrigger(p ProcessInterface, thresholds Thresholds, thresholdParams ThresholdParams, thresholdType string) bool {

	pid := p.GetID()
	logrus.WithField("pid", pid).Debugf("checking if trigger conditions met")
	var checkDelay int
	var thresholdReached bool
	var triggerCount int

	switch thresholdType {
	case ThresholdTypeRSS:
		thresholdReached = thresholds.RSS
		triggerCount = thresholdParams.RSSTriggerCount
		checkDelay = thresholdParams.RSSTriggerDelay
	case ThresholdTypeCPU:
		thresholdReached = thresholds.CPU
		triggerCount = thresholdParams.CPUTriggerCount
		checkDelay = thresholdParams.CPUTriggerDelay
	default:
		panic("unknown threshold type")
	}

	// checkTriggers is only called once the initial trigger event is fired
	if thresholdReached && triggerCount <= 1 {
		logrus.WithField("pid", pid).Infof("trigger count is <=1 - trigger fired after initial check for %s", thresholdType)
		return true
	}

	trigger := make(chan bool)

	var err error

	go func() {
		for check := 1; check <= triggerCount; check++ {
			thresholds, err = CheckThresholds(p, thresholdParams)
			if err != nil {
				logrus.WithField("pid", pid).Errorf("something went wrong when checking thresholds for trigger: %v", err)
				trigger <- false
			}
			aboveThreshold := false
			switch thresholdType {
			case ThresholdTypeRSS:
				aboveThreshold = thresholds.RSS
			case ThresholdTypeCPU:
				aboveThreshold = thresholds.CPU
			}

			if aboveThreshold {
				if check >= triggerCount {
					logrus.WithField("pid", pid).Infof("%s trigger count reached %d/%d. Trigger fired",
						thresholdType, check, triggerCount)
					trigger <- true
					return
				}
				logrus.WithField("pid", pid).Infof("%s trigger count %d/%d. Waiting %d seconds before next check",
					thresholdType, check, triggerCount, checkDelay)
				time.Sleep(time.Second * time.Duration(checkDelay))
			} else {
				// thresholdParams was not reached so do not trigger
				logrus.WithField("pid", pid).
					Debugf("%s trigger was not fired as it was below threshold", thresholdType)
				trigger <- false
				return
			}
		}
		trigger <- false
	}()
	return <-trigger
}
