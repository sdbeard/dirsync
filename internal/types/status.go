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

	logger "github.com/sirupsen/logrus"
)

var syncStatusMutex sync.Mutex

/***** StatusUpdateFunc ***********************************************************/

// StatusUpdateFunc defines a type for a function that passed in a a reference to a
// SynchronizerStatus object for the purpose of having an appliction defined method
// for updating the synchronizer status
type StatusUpdateFunc func(status *SynchronizerStatus)

/**********************************************************************************/

/***** SynchronizerStatus *********************************************************/

// SynchronizerStatus provides the statistics from the current or latest run of the
// service. This struct provides all of the instrumentation that is resulting from
// the current run or the latest
type SynchronizerStatus struct {
	ProfileConfiguration Profile           `json:"profile"`
	StartTime            time.Time         `json:"start"`
	EndTime              time.Time         `json:"end"`
	NextRunTime          time.Time         `json:"nextruntime"`
	ProcessingDuration   time.Duration     `json:"processingduration"`
	ActiveTransfers      map[string]string `json:"activetransfers"`
	SyncJobCount         int               `json:"syncjobcount"`
	ErrorCount           int               `json:"errorcount"`
	IsRunning            bool              `json:"isrunning"`
}

/***** Marshal/Unmarshal functions ************************************************/

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

/***** exported functions *********************************************************/

// CreateSyncServiceStatus creates a new instance of the DirectorySyncServiceStatus
// for the running API
func CreateSynchronizerStatus(configuration Profile) *SynchronizerStatus {
	return &SynchronizerStatus{
		ProfileConfiguration: configuration,
		ProcessingDuration:   0 * time.Second,
		ActiveTransfers:      make(map[string]string),
		SyncJobCount:         0,
		ErrorCount:           0,
	}
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

/**********************************************************************************/
