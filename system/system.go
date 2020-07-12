package system

import (
	"os/exec"
)

// CheckBinaryOnPath checks to see if the specified binary is on the system path
func CheckBinaryOnPath(binary string) bool {
	_, err := exec.LookPath(binary)
	if err != nil {
		return false
	}
	return true
}

func RunCmd(cmd *exec.Cmd) (string, error) {
	result, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(result), nil
}
