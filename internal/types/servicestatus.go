// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package types

import (
	"encoding/json"
	"sync"
)

var (
	statusMutex sync.Mutex
	status      = make(map[string]*SynchronizerStatus)
)

/**********************************************************************************/

// GetServiceStatus returns the current service status object
func GetServiceStatus() ServiceStatus {
	return status
}

/**********************************************************************************/

/***** ServiceStatus **************************************************************/

// ServiceStatus provides a container for the status objects covering all of the
// configured synchronizers
type ServiceStatus map[string]*SynchronizerStatus

// Log leverages the current configured logrus logger and writes the current
// status to the log
func (status ServiceStatus) Log() {
	for _, synchronizerStatus := range status {
		synchronizerStatus.Log()
	}
}

// YAML converts the status object to a status string
func (status ServiceStatus) JSON() []byte {
	statusBytes, err := json.Marshal(status)
	if err != nil {
		return []byte{}
	}

	return statusBytes
}

func (status ServiceStatus) Add(syncStatus *SynchronizerStatus) {
	statusMutex.Lock()
	status[syncStatus.ProfileConfiguration.Name] = syncStatus
	statusMutex.Unlock()
}

func (status ServiceStatus) GetSynchronizerStatus(synchronizerName string) *SynchronizerStatus {
	return status[synchronizerName]
}

/**********************************************************************************/
