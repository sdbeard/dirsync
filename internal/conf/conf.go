// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package conf

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sdbeard/dirsync/internal/types"
	apicfg "github.com/sdbeard/go-supportlib/api/config"
	"github.com/sdbeard/go-supportlib/common/logging"
	"github.com/sdbeard/go-supportlib/common/util"
	"github.com/sdbeard/service"
)

const confFile = "config.yaml"

var synchronizerConf *SynchronizerConf

/***** FileCopyOptions ************************************************************/

// FileCopyOptions holds all of the fields that optimize the copying of files to S3.
// These options include throttling values, max number of concurrent transfers, etc.
//type FileCopyOptions struct {
//	ThrottleBucketSizeKB int64 `yaml:"throttlebucketsizekb"`
//	ThrottleSync         bool  `yaml:"throttle"`
//	MaxConcurrentCopies  int   `yaml:"maxconcurrentcopies"`
//}

/***********************************************************************************/

/***** ExecutionFlags *************************************************************/

// ExecutionFlags holds all of the parameters the define the exectuion environment
// for the service to run under, these values determine how to change the way the
// service runs, not the job the service is doing
type ExecutionFlags struct {
	//ExecutionFolder string `yaml:"exefolder"`
	RuntimeEnv    string `json:"env"`
	NoSchedule    bool   `json:"noschedule"`
	Simulation    bool   `json:"simulation"`
	FileOverwrite bool   `json:"overwrite"`
}

/***********************************************************************************/

/***** SynchronizerDirectoryConfig ************************************************/

// SynchronizerDirectoryProfile contains the parameters to completely configure an
// instance of a Synchronizer
//type SynchronizerDirectoryProfile struct {
//	S3Config        s3.Configuration `yaml:"s3config"`
//	FileCopyOptions FileCopyOptions  `yaml:"filecopyoptions"`
//	Extensions      []string         `yaml:"ext"`
//	Exclusions      []string         `yaml:"exclusions"`
//	Name            string           `yamo:"name"`
//	Description     string           `yaml:"description"`
//	SourceFolder    string           `yaml:"source"`
//	ScheduleDef     string           `yaml:"scheduledef"`
//	Recursive       bool             `yaml:"recursive"`
//	RunAtStartup    bool             `yaml:"runatstartup"`
//}

/**********************************************************************************/

/***** SynchronizerConf ***********************************************************/

// SynchronizerConf holds all of the parameters to configure the
// service. This is designed to be read from a JSON file running in the current
// directory. The name of the JSON file is hard coded to "dirsync_cfg.json"
type SynchronizerConf struct {
	Profiles             map[string]types.Profile `json:"profiles"`
	ExecFlags            ExecutionFlags           `json:"executionflags"`
	LogConf              logging.LogConfig        `json:"logconfig"`
	ServiceConfiguration *service.Config          `json:"serviceconfig"`
	APIConf              apicfg.ListenerConfig    `json:"apiconf"`
	WorkingFolder        string                   `json:"-"`
}

/***********************************************************************************/

// GetConfiguration retrieves the current configuration read from dirsync_cfg.json,
// and returns a pointer to a DirectorySyncS3ServiceConfiguration struct
func GetSynchronizerConf() *SynchronizerConf {
	return synchronizerConf
}

// LoadConfiguration loads the configuration file from yaml to the configuration
func LoadSynchronizerConf(file string) error {
	// Get the working folder
	workingDir, _ := os.Getwd()
	workingFolder, _ := filepath.Abs(workingDir)

	// Create the configuration object and set defaults
	synchronizerConf = &SynchronizerConf{}
	if file != "" {
		fileBytes, err := util.ReadFile(filepath.Join(workingFolder, file))
		if err != nil {
			return err
		}

		// Unmarshal the configuration file
		if err = json.Unmarshal(fileBytes, synchronizerConf); err != nil {
			return err
		}
	}

	/*
		if !service.Interactive() {
			exeFolder, err := osext.ExecutableFolder()
			if err != nil {
				return (err)
			}
			configFile = fmt.Sprintf("%s%s%s", exeFolder, string(os.PathSeparator), confFile)
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
				string(os.PathSeparator), confFile)
		}
	*/

	// Set the current folder
	synchronizerConf.WorkingFolder = workingFolder
	//synchronizerConf.ExecFlags.ExecutionFolder = workingFolder
	synchronizerConf.ServiceConfiguration.WorkingDirectory = workingFolder

	return nil
}

/***********************************************************************************/
