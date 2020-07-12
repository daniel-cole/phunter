package process

type MockProcess struct {
	ID int
}

func (p *MockProcess) FindContainerName() (string, error) {
	return "container", nil
}

// GetCPU returns the current CPU utilisation of the specified process
func (p *MockProcess) GetCPU() (float64, error) {
	return 200, nil
}

// GetRSS returns the current resident memory in KiB of the specified process
func (p *MockProcess) GetRSS() (int64, error) {
	return 1048576, nil
}

func (p *MockProcess) GetID() int {
	return p.ID
}

func (p *MockProcess) PrintPIDResourceUsage() error {
	return nil
}

func (p *MockProcess) FindAndSetContainerName() error {
	return nil
}
