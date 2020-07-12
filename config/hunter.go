package config

import "github.com/daniel-cole/phunter/process"

type HunterConfig struct {
	Application        string                  `yaml:"application"`
	ApplicationVersion string                  `yaml:"application_version"`
	ProcessCommand     string                  `yaml:"process_command"`
	CheckInterval      int                     `yaml:"check_interval"`
	TraceDuration      int                     `yaml:"trace_duration"`
	TraceDir           string                  `yaml:"trace_dir"`
	Docker             bool                    `yaml:"docker"`
	Dryrun             bool                    `yaml:"dryrun"`
	Timezone           string                  `yaml:"timezone"`
	ThresholdParams    process.ThresholdParams `yaml:"threshold_params"`
	PHPSpyConfig       PHPSpyConfig            `yaml:"phpspy"`
}

// For configuration options see: https://github.com/adsr/phpspy
type PHPSpyConfig struct {
	Threads   string `yaml:"threads"` // -T
	Sleep     string `yaml:"sleep"` // -s
	Rate      string `yaml:"rate"` // -H
	Limit     string `yaml:"limit"` // -l
}
