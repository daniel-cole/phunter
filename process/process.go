package process

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type ProcessInterface interface {
	GetCPU() (float64, error)
	GetRSS() (int64, error)
	GetID() int
	PrintPIDResourceUsage() error
	FindContainerName() (string, error)
}

// Process represents a process running on the system
type Process struct {
	ID int
}

// GetCPU returns the current CPU utilisation of the specified process
func (p *Process) GetCPU() (float64, error) {
	cmd := fmt.Sprintf("top -bp %d -n1 | awk '/%d/{print $9}'", p.ID, p.ID)
	result, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return -1, err
	}
	pidCPU, err := strconv.ParseFloat(tidyPSOutput(string(result)), 64)
	if err != nil {
		return -1, errors.New(fmt.Sprintf("failed to parse process cpu, process probably dissapeared. error: %v", err))
	}
	return pidCPU, err
}

// GetRSS returns the current resident memory in KiB of the specified process
func (p *Process) GetRSS() (int64, error) {
	result, err := exec.Command("ps", "-p", strconv.Itoa(p.ID), "-o", "rss=").Output()
	if err != nil {
		return -1, err
	}
	pidRSS, err := strconv.ParseInt(tidyPSOutput(string(result)), 10, 64)
	if err != nil {
		return -1, err
	}
	return pidRSS, err
}

func (p *Process) GetID() int {
	return p.ID
}

func (p *Process) PrintPIDResourceUsage() error {
	cpu, err := p.GetCPU()
	if err != nil {
		return err
	}
	rss, err := p.GetRSS()
	if err != nil {
		return err
	}
	fmt.Printf("%d: CPU: %.2f RSS: %d\n", p.ID, cpu, rss)
	return nil
}

// GetPIDListByCommand returns the list of process IDs returned from the command
// runs pgrep <command> on the system to obtain the list
func GetPIDListByCommand(command string) ([]int, error) {
	result, err := exec.Command("pgrep", command).Output()
	if err != nil {
		return nil, err
	}
	// trim newline character on end
	foundPIDs := strings.Split(strings.TrimRight(string(result), "\r\n"), "\n")
	var pidList []int
	for _, pidAsStr := range foundPIDs {
		pidInt, err := strconv.Atoi(pidAsStr)
		if err != nil {
			return nil, err
		}
		pidList = append(pidList, pidInt)
	}
	return pidList, nil
}

// Remove any spaces and newline characters from output so it can be parsed into the appropriate primitive
func tidyPSOutput(output string) string {
	return strings.TrimLeft(strings.TrimRight(strings.TrimSpace(output), "\n"), "\n")
}
