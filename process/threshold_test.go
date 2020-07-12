package process

import (
	"testing"
)

var testThresholdParams = ThresholdParams{
	CPUThreshold:    100,
	CPUTriggerCount: 3,
	CPUTriggerDelay: 5,
	RSSThreshold:    1024,
	RSSTriggerCount: 3,
	RSSTriggerDelay: 5,
}

func TestCheckThresholdsAbove(t *testing.T) {
	var p ProcessInterface
	p = &MockProcess{
		ID: 1,
	}

	aboveCPUThreshold, err := checkCPUThreshold(p, testThresholdParams)
	if err != nil {
		t.Fatal(err)
	}
	if !aboveCPUThreshold {
		t.Error("expected process to be above CPU threshold")
	}

	aboveRSSThreshold, err := checkRSSThreshold(p, testThresholdParams)
	if err != nil {
		t.Fatal(err)
	}
	if !aboveRSSThreshold {
		t.Error("expected process to be above RSS threshold")
	}

}

func TestCheckThresholdsBelow(t *testing.T) {
	var p ProcessInterface
	p = &MockProcess{
		ID: 1,
	}

	params := testThresholdParams
	params.CPUThreshold = 300
	params.RSSThreshold = 2097152

	aboveCPUThreshold, err := checkCPUThreshold(p, params)
	if err != nil {
		t.Fatal(err)
	}
	if aboveCPUThreshold {
		t.Error("expected process to be below CPU threshold")
	}

	aboveRSSThreshold, err := checkRSSThreshold(p, params)
	if err != nil {
		t.Fatal(err)
	}
	processRSS, err := p.GetRSS()
	if err != nil {
		t.Fatal("failed to get RSS for process")
	}
	if aboveRSSThreshold {
		t.Errorf("expected process to be below RSS threshold, threshold: %d RSS: %d",
			params.RSSThreshold, processRSS)
	}
}

func TestCheckThresholds(t *testing.T) {
	var p ProcessInterface
	p = &MockProcess{
		ID: 1,
	}
	thresholds, err := CheckThresholds(p, testThresholdParams)
	if err != nil {
		t.Fatal(err)
	}
	if !thresholds.CPU || !thresholds.RSS {
		t.Error("expected both CPU and RSS to be above the threshold")
	}
}

func TestCheckThresholdTriggers(t *testing.T) {
	var p ProcessInterface
	p = &MockProcess{
		ID: 1,
	}

	params := testThresholdParams
	params.RSSTriggerCount = 2
	params.RSSTriggerDelay = 1
	params.RSSThreshold = 1024

	thresholds, err := CheckThresholds(p, params)
	if err != nil {
		t.Fatal(err)
	}
	triggerType, triggered := CheckThresholdTriggers(p, thresholds, params)
	if !triggered {
		t.Error("expected check threshold triggers to trigger")
	}
	if triggerType != ThresholdTypeRSS {
		t.Errorf("expected trigger type to be %s, instead got: %s", ThresholdTypeRSS, triggerType)
	}
}

func TestRacyCheckThresholdTriggers(t *testing.T) {
	var p ProcessInterface
	p = &MockProcess{
		ID: 1,
	}

	params := testThresholdParams
	params.RSSTriggerCount = 2
	params.RSSTriggerDelay = 1
	params.RSSThreshold = 1024
	params.CPUTriggerCount = 2
	params.CPUTriggerDelay = 1
	params.CPUThreshold = 50

	thresholds, err := CheckThresholds(p, params)
	if err != nil {
		t.Fatal(err)
	}
	_, triggered := CheckThresholdTriggers(p, thresholds, params)
	if !triggered {
		t.Error("expected check threshold triggers to trigger")
	}
}

func TestCheckThresholdTriggerCPUOnly(t *testing.T) {
	var p ProcessInterface
	p = &MockProcess{
		ID: 1,
	}

	params := testThresholdParams
	params.CPUTriggerCount = 2
	params.CPUTriggerDelay = 1
	params.CPUThreshold = 50

	thresholds, err := CheckThresholds(p, params)
	if err != nil {
		t.Fatal(err)
	}
	triggerType, triggered := CheckThresholdTriggers(p, thresholds, params)
	if !triggered {
		t.Error("expected check threshold triggers to trigger")
	}
	if triggerType != ThresholdTypeCPU {
		t.Errorf("expected trigger type to be %s, instead got: %s", ThresholdTypeCPU, triggerType)
	}
}
