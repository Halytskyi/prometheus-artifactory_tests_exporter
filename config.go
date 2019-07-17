package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// Config structure
type Config struct {
	ListenAddress string                    `yaml:"listen_address,omitempty"`
	MetricsPath   string                    `yaml:"metrics_path,omitempty"`
	Interval      time.Duration             `yaml:"interval,omitempty"`
	Timeout       time.Duration             `yaml:"timeout,omitempty"`
	TestFilesPath string                    `yaml:"test_files_path,omitempty"`
	Debug         bool                      `yaml:"debug,omitempty"`
	Artifactory   ArtifactoryParams         `yaml:"artifactory,omitempty"`
	TestFiles     map[string]TestFileParams `yaml:"test_files"`
}

// SafeConfig structure
type SafeConfig struct {
	sync.RWMutex
	C *Config
}

// LoadConfig function
func (sc *SafeConfig) LoadConfig(confFile string) (err error) {
	var c = &Config{}

	yamlFile, err := ioutil.ReadFile(confFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %s", err)
	}
	if err := yaml.UnmarshalStrict(yamlFile, c); err != nil {
		return fmt.Errorf("Error parsing config file: %s", err)
	}

	sc.Lock()
	sc.C = c
	sc.Unlock()

	return nil
}

// TestFileParams structure
type TestFileParams struct {
	Size                int           `yaml:"size,omitempty"`
	HistogramBucketPush []float64     `yaml:"histogram_bucket_push,omitempty"`
	HistogramBucketPull []float64     `yaml:"histogram_bucket_pull,omitempty"`
	TimeoutPush         time.Duration `yaml:"timeout_push,omitempty"`
	TimeoutPull         time.Duration `yaml:"timeout_pull,omitempty"`
	VerifyChecksum      bool          `yaml:"verify_checksum,omitempty"`
}

// ArtifactoryParams structure
type ArtifactoryParams struct {
	URL      string `yaml:"url,omitempty"`
	RepoPath string `yaml:"repo_path,omitempty"`
}

// UnmarshalYAML function for Config
func (s *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Config
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	if s.TestFiles == nil {
		return errors.New("Parameters should be defined for at least one test file")
	}
	if s.Artifactory.URL == "" {
		return errors.New("Artifactory 'URL' must be defined")
	}
	if s.Artifactory.RepoPath == "" {
		return errors.New("Artifactory 'RepoPath' must be defined")
	}
	return nil
}

// UnmarshalYAML function for Record
func (s *TestFileParams) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain TestFileParams
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	if s.Size == 0 {
		return errors.New("File 'size' must be defined")
	}
	if s.HistogramBucketPush == nil {
		return errors.New("File 'histogram_bucket_push' must be defined")
	}
	if s.HistogramBucketPull == nil {
		return errors.New("File 'histogram_bucket_pull' must be defined")
	}
	return nil
}

// UnmarshalYAML function for ArtifactoryParams
func (s *ArtifactoryParams) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain ArtifactoryParams
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	return nil
}
