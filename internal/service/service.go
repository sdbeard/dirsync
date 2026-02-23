// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package service

import (
	"github.com/robfig/cron"
	"github.com/sdbeard/dirsync/api"
	"github.com/sdbeard/dirsync/internal/conf"
	"github.com/sdbeard/service"
	logger "github.com/sirupsen/logrus"
)

// NewSynchronizerService create and returns a reference to a new
// SynchronizerService object
func NewSynchronizerService() (*SynchronizerService, error) {
	newService := &SynchronizerService{
		synchronizers: make(map[string]*Synchronizer),
		scheduler:     cron.New(),
		statusAPI:     api.NewSynchronizerStatusAPI(conf.GetSynchronizerConf().StatusAPIConfiguration),
		shuttingDown:  false,
	}

	// initialize the service
	err := newService.initialize()

	return newService, err
}

/***** SynchronizerService ********************************************************/

// SynchronizerService is the logical object that wraps the Synchronizer with the
// daemon/Windows SCM functionality to run it as system level service
type SynchronizerService struct {
	systemService service.Service
	synchronizers map[string]*Synchronizer
	scheduler     *cron.Cron
	statusAPI     *api.SynchronizerStatusAPI
	finishChan    chan bool
	shuttingDown  bool
}

/***** kardianos/service program interface implementation *************************/

// Start initializes the newly created DirectorySyncServiceProgram and starts the
// underlying DirectorySyncService. Returns an error if one occurred while starting
// the service otherwise returns nil
// Start the system service
func (syncsvc *SynchronizerService) Start(service service.Service) error {
	syncsvc.run()
	return nil
}

// Stop shuts down the DirectorySyncService and gracefully exits the program.
// Returns an error if one occurred while stopingthe service otherwise returns nil
func (syncsvc *SynchronizerService) Stop(svc service.Service) error {
	for _, synchronizer := range syncsvc.synchronizers {
		synchronizer.Shutdown()
	}

	synchronizerCount := 0
	for range syncsvc.finishChan {
		synchronizerCount++
		if synchronizerCount == len(syncsvc.synchronizers) {
			break
		}
	}
	close(syncsvc.finishChan)

	// Stop the API
	syncsvc.statusAPI.Stop()

	return nil
}

// run kicks off the work to be done on the underlying service. This function also
// orchestrates/manages the scheduling function used to determine how often the
// service needs to run
func (syncsvc *SynchronizerService) run() {
	// Create the new channel
	syncsvc.finishChan = make(chan bool, len(conf.GetSynchronizerConf().SyncProfiles))

	// Start the API
	go syncsvc.statusAPI.Start()

	for _, synchronizer := range syncsvc.synchronizers {
		go synchronizer.Startup(syncsvc.finishChan)
	}
}

/***** exported functions *********************************************************/

// RetrieveSynchronizer retrieves a single synchronizer by name
func (syncsvc *SynchronizerService) RetrieveSynchronizer(name string) *Synchronizer {
	synchronizer, ok := syncsvc.synchronizers[name]
	if !ok {
		logger.Infof("the synchronizer (%s) was not found", name)
		return nil
	}
	return synchronizer
}

// Run execute a single synchronizer by name
func (syncsvc *SynchronizerService) RunSynchronizer(name string) {
	synchronizer, ok := syncsvc.synchronizers[name]
	if !ok {
		logger.Infof("the synchronizer (%s) was not found", name)
		return
	}
	synchronizer.Run()
}

// GetSystemService creates the system level service
func (syncsvc *SynchronizerService) GetSystemService() (service.Service, error) {
	if syncsvc.systemService == nil {
		// Create the system service
		newSystemService, err := service.New(syncsvc, conf.GetSynchronizerConf().ServiceConfiguration)
		if err != nil {
			return nil, err
		}
		syncsvc.systemService = newSystemService
	}

	return syncsvc.systemService, nil
}

/**********************************************************************************/

// initialize takes the configuration and system service and configures the service
// to be run. Primarily configuring the scheduler
func (syncsvc *SynchronizerService) initialize() error {
	// Create, configure and add all of the synchronizers
	for _, profileConfiguration := range conf.GetSynchronizerConf().SyncProfiles {
		synchronizer, err := NewSynchronizer(profileConfiguration)
		if err != nil {
			return err
		}
		syncsvc.synchronizers[profileConfiguration.Name] = synchronizer
	}

	return nil
}

/**********************************************************************************/
