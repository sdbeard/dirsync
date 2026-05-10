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

// LoadConfiguration loads the configuration file from yaml to the configuration
func LoadSynchronizerConf(file string) (SynchronizerConf, error) {
	// Get the working folder
	workingDir, _ := os.Getwd()
	workingFolder, _ := filepath.Abs(workingDir)

	// Create the configuration object and set defaults
	synchronizerConf := &SynchronizerConf{}
	if file != "" {
		fileBytes, err := util.ReadFile(filepath.Join(workingFolder, file))
		if err != nil {
			return *synchronizerConf, err
		}

		// Unmarshal the configuration file
		if err = json.Unmarshal(fileBytes, synchronizerConf); err != nil {
			return *synchronizerConf, err
		}
	}

	// Set the current folder
	synchronizerConf.WorkingFolder = workingFolder
	synchronizerConf.ServiceConfiguration.WorkingDirectory = workingFolder

	return *synchronizerConf, nil
}

/***********************************************************************************/
