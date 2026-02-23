// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package conf

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/kardianos/osext"
	apicfg "github.com/sdbeard/go-supportlib/api/config"
	"github.com/sdbeard/go-supportlib/aws/service/s3"
	"github.com/sdbeard/go-supportlib/common/logging"
	"github.com/sdbeard/service"
	"gopkg.in/yaml.v3"
)

const configurationFile = "config.yaml"

var configuration *SynchronizerServiceConfiguration

/***** FileCopyOptions ************************************************************/

// FileCopyOptions holds all of the fields that optimize the copying of files to S3.
// These options include throttling values, max number of concurrent transfers, etc.
type FileCopyOptions struct {
	ThrottleBucketSizeKB int64 `yaml:"throttlebucketsizekb"`
	ThrottleSync         bool  `yaml:"throttle"`
	MaxConcurrentCopies  int   `yaml:"maxconcurrentcopies"`
}

/***********************************************************************************/

/***** ExecutionFlags *************************************************************/

// ExecutionFlags holds all of the parameters the define the exectuion environment
// for the service to run under, these values determine how to change the way the
// service runs, not the job the service is doing
type ExecutionFlags struct {
	ExecutionFolder string `yaml:"exefolder"`
	RuntimeEnv      string `yaml:"env"`
	NoSchedule      bool   `yaml:"noschedule"`
	Simulation      bool   `yaml:"simulation"`
	FileOverwrite   bool   `yaml:"overwrite"`
}

/***********************************************************************************/

/***** SynchronizerDirectoryConfig ************************************************/

// SynchronizerDirectoryProfile contains the parameters to completely configure an
// instance of a Synchronizer
type SynchronizerDirectoryProfile struct {
	S3Config        s3.Configuration `yaml:"s3config"`
	FileCopyOptions FileCopyOptions  `yaml:"filecopyoptions"`
	Extensions      []string         `yaml:"ext"`
	Exclusions      []string         `yaml:"exclusions"`
	Name            string           `yamo:"name"`
	Description     string           `yaml:"description"`
	SourceFolder    string           `yaml:"source"`
	ScheduleDef     string           `yaml:"scheduledef"`
	Recursive       bool             `yaml:"recursive"`
	RunAtStartup    bool             `yaml:"runatstartup"`
}

/**********************************************************************************/

/***** SynchronizerServiceConfiguration *******************************************/

// SynchronizerServiceConfiguration holds all of the parameters to configure the
// service. This is designed to be read from a JSON file running in the current
// directory. The name of the JSON file is hard coded to "dirsync_cfg.json"
type SynchronizerServiceConfiguration struct {
	SyncProfiles           map[string]SynchronizerDirectoryProfile `yaml:"syncprofiles"`
	ExecFlags              ExecutionFlags                          `yaml:"executionflags"`
	LogConfiguration       logging.LogConfig                       `yaml:"logconfig"`
	ServiceConfiguration   *service.Config                         `yaml:"serviceconfig"`
	StatusAPIConfiguration apicfg.ListenerConfig                   `yaml:"apiconfig"`
}

/***********************************************************************************/

// GetConfiguration retrieves the current configuration read from dirsync_cfg.json,
// and returns a pointer to a DirectorySyncS3ServiceConfiguration struct
func GetConfiguration() *SynchronizerServiceConfiguration {
	return configuration
}

// LoadConfiguration loads the configuration file from yaml to the configuration
func LoadConfiguration() error {
	var configFile, workingFolder string

	if !service.Interactive() {
		exeFolder, err := osext.ExecutableFolder()
		if err != nil {
			return (err)
		}
		configFile = fmt.Sprintf("%s%s%s", exeFolder, string(os.PathSeparator), configurationFile)
		workingFolder = exeFolder
	} else {
		workingDir, err := os.Getwd()
		if err != nil {
			return err
		}

		workingFolder, err = filepath.Abs(workingDir)
		if err != nil {
			return err
		}
		configFile = fmt.Sprintf("%s%s%s", workingFolder,
			string(os.PathSeparator), configurationFile)
	}

	configBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}

	// Create the configuration object
	configuration = &SynchronizerServiceConfiguration{}

	// Unmarshal the configuration file
	err = yaml.Unmarshal(configBytes, configuration)
	if err != nil {
		return err
	}

	// Set the current folder
	configuration.ExecFlags.ExecutionFolder = workingFolder
	configuration.ServiceConfiguration.WorkingDirectory = workingFolder

	return nil
}

/***********************************************************************************/
