// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package types

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sdbeard/dirsync/internal/conf"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var syncStatusMutex sync.Mutex

/***** StatusUpdateFunc ***********************************************************/

// StatusUpdateFunc defines a type for a function that passed in a a reference to a
// SycnhronizerStatus object for the purpose of having an appliction defined method
// for updating the synchronizer status
type StatusUpdateFunc func(status *SynchronizerStatus)

/**********************************************************************************/

/***** SynchronizerStatus *********************************************************/

// SynchronizerStatus provides the statistics from the current or latest run of the
// service. This struct provides all of the instrumentation that is resulting from
// the current run or the latest
type SynchronizerStatus struct {
	ProfileConfiguration conf.SynchronizerDirectoryProfile `json:"profileconfig" yaml:"profileconfig"`
	StartTime            time.Time                         `json:"start" yaml:"start"`
	EndTime              time.Time                         `json:"end" yaml:"end"`
	NextRunTime          time.Time                         `json:"nextruntime" yaml:"nextruntime"`
	ProcessingDuration   time.Duration                     `json:"processingduration" yaml:"processingduration"`
	ActiveTransfers      map[string]string                 `json:"activetransfers" yaml:"activetransfers"`
	SyncJobCount         int                               `json:"syncjobcount" yaml:"syncjobcount"`
	ErrorCount           int                               `json:"errorcount" yaml:"errorcount"`
	IsRunning            bool                              `json:"isrunning" yaml:"isrunning"`
}

// MarshalJSON is a custom JSON serializer to convert the time based fields to more
// readable versions
func (status *SynchronizerStatus) MarshalJSON() ([]byte, error) {
	type Alias SynchronizerStatus
	return json.Marshal(&struct {
		StartTime          string `json:"start"`
		EndTime            string `json:"end"`
		ProcessingDuration string `json:"processingduration"`
		*Alias
	}{
		StartTime:          status.StartTime.Format("2006-01-02 15:04:05 MST"),
		EndTime:            status.EndTime.Format("2006-01-02 15:04:05 MST"),
		ProcessingDuration: fmt.Sprintf("%.3f seconds", status.ProcessingDuration.Seconds()),
		Alias:              (*Alias)(status),
	})
}

// MarshalYAML is a custom YAML serializer to convert the time based fields to more
// readable versions
func (status *SynchronizerStatus) MarshalYAML() ([]byte, error) {
	type Alias SynchronizerStatus
	return yaml.Marshal(&struct {
		StartTime          string `yaml:"start"`
		EndTime            string `yaml:"end"`
		ProcessingDuration string `yaml:"processingduration"`
		*Alias
	}{
		StartTime:          status.StartTime.Format("2006-01-02 15:04:05 MST"),
		EndTime:            status.EndTime.Format("2006-01-02 15:04:05 MST"),
		ProcessingDuration: fmt.Sprintf("%.3f seconds", status.ProcessingDuration.Seconds()),
		Alias:              (*Alias)(status),
	})
}

// Log leverages the current configured logrus logger and writes the current
// status to the log
func (status SynchronizerStatus) Log() {
	logger.WithFields(map[string]interface{}{
		"Duration":     fmt.Sprintf("%.3fs", status.ProcessingDuration.Seconds()),
		"SyncJobCount": status.SyncJobCount,
		"ErrorCount":   status.ErrorCount,
		"StartTime":    status.StartTime.Format("2006-01-02 15:04:05 MST"),
		"EndTime":      status.EndTime.Format("2006-01-02 15:04:05 MST"),
		"IsRunning":    status.IsRunning,
	}).Info("Last Run Service Status")
}

// JSON converts the status object to a json status string
func (status SynchronizerStatus) JSON() []byte {
	statusBytes, err := json.Marshal(status)
	if err != nil {
		return []byte{}
	}
	return statusBytes
}

// YAML converts the status object to a yaml status string
func (status SynchronizerStatus) YAML() []byte {
	statusBytes, err := yaml.Marshal(status)
	if err != nil {
		return []byte{}
	}
	return statusBytes
}

// Update updates a given status value by passing in a function of type
// StatusUpdateFunc the updates the status object in a thread safe manner
func (status *SynchronizerStatus) Update(statusUpdate StatusUpdateFunc) {
	syncStatusMutex.Lock()
	defer syncStatusMutex.Unlock()

	statusUpdate(status)
}

// UpdateActiveTransfer updats the status object ofr the current state of a
// transfer. It is either added or removed depending on the value of the remove
// boolean
func (status *SynchronizerStatus) UpdateActiveTransfer(threadName, fileName string) {
	syncStatusMutex.Lock()
	defer syncStatusMutex.Unlock()

	status.ActiveTransfers[threadName] = fileName
}

// RemoveActiveTransfer removes the active thread name transfer if it exists
func (status *SynchronizerStatus) RemoveActiveTransfer(threadName string) {
	syncStatusMutex.Lock()
	defer syncStatusMutex.Unlock()

	delete(status.ActiveTransfers, threadName)
}

// IncrementSyncJob updates the status object by increment the SyncJobCount
func (status *SynchronizerStatus) IncrementSyncJob() {
	syncStatusMutex.Lock()
	defer syncStatusMutex.Unlock()

	status.SyncJobCount++
}

/***** exported functions *********************************************************/

// CreateSyncServiceStatus creates a new instance of the DirectorySyncServiceStatus
// for the running API
func CreateSynchronizerStatus(configuration conf.SynchronizerDirectoryProfile) *SynchronizerStatus {
	return &SynchronizerStatus{
		ProfileConfiguration: configuration,
		ProcessingDuration:   0 * time.Second,
		ActiveTransfers:      make(map[string]string),
		SyncJobCount:         0,
		ErrorCount:           0,
	}
}

/**********************************************************************************/
