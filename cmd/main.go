// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package main

/*
TODO:
	- Need in depth help as part of cli to explain the usage of the different
	  parameters
*/

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/sdbeard/dirsync/internal/conf"
	syncsvc "github.com/sdbeard/dirsync/internal/service"
	"github.com/sdbeard/dirsync/internal/types"
	"github.com/sdbeard/go-supportlib/common/logging"
	"github.com/sdbeard/go-supportlib/common/util"
	"github.com/sdbeard/service"
)

var (
	config    = flag.String("config", "config.json", "specifies the configuration file to use for the service configuration")
	profile   = flag.String("profile", "", "specifies a base profile to use/process; ignored in a non-interactive session")
	remote    = flag.Bool("remote", false, "interactive only parameter: retrieves the remote directory and file listing from configured remote location; ignored if a profile is not provided)")
	simulate  = flag.Bool("simulate", false, "tells the system to only simulate synchronizing files. No file are acutally copied to S3. (For development purposes only (only valid in stand-alone mode)")
	overwrite = flag.Bool("overwrite", false, "tells the system to overwrite a file on S3.")
	version   = "1.0.0"
	env       = "local"
	build     = ""
	buildDate = ""
	command   = ""
)

/**********************************************************************************/

func init() {
	flag.Parse()
}

func main() {
	cfg, err := conf.LoadSynchronizerConf(*config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading configuration: %w", err)
		os.Exit(1)
	}

	initializeCmdLineParameters(&cfg)

	logger := logging.NewLogger(cfg.LogConf)
	if logger == nil {
		fmt.Fprint(os.Stderr, "could not create the logger")
		os.Exit(1)
	}

	logger.Info("Directory Sync v3.0.0")
	logger.Info("Runtime configuration",
		"version", version,
		"build", buildDate,
		"environment", env,
		"go version", runtime.Version(),
		"pid", os.Getpid(),
	)

	if err := runSyncService(); err != nil {
		logger.Error("running sync service", "error", err)
		os.Exit(1)
	}

	logger.Info("dirsync has completely shutdown")
	os.Exit(0)
}

/**********************************************************************************/

func runSyncService(cfg conf.SynchronizerConf, logger *slog.Logger) error {
	logger.Debug("running sync service")

	syncService, err := syncsvc.NewSynchronizerService()
	if err != nil {
		return err
	}

	if service.Interactive() {
		return runInteractive(syncService)
	}

	// Set the configuration so that any stand-alone mode flags are set to false or empty
	conf.GetSynchronizerConf().ExecFlags.NoSchedule = false
	conf.GetSynchronizerConf().ExecFlags.Simulation = false

	// Run the service
	return fmt.Errorf("not implemented exception")
	//return runAsService(syncService)
}

/**********************************************************************************/

func runInteractive(syncService *syncsvc.SynchronizerService, cfg conf.SynchronizerConf, logger *slog.Logger) error {
	logger.Info("starting: interactive mode")

	profiles := []string{*profile}
	if *profile == "" {
		profiles = util.GetMapKeySlice(conf.GetSynchronizerConf().Profiles)
	}

	// Run all of the configured profiles
	/*if *profile != "" {
		if _, ok := conf.GetSynchronizerConf().Profiles[*profile]; !ok {
			return fmt.Errorf("run interactive: profile not found")
		}
		runSingleSynchronizer(*profile, syncService)
		return nil
	}*/

	// Run the service from an interactive space
	return runInteractiveService(syncService, profiles)

	// Get the service status and print the status to the log
	//types.GetServiceStatus().Log()
}

/*
func createSyncService() error {
	syncService, err := syncsvc.NewSynchronizerService()
	if err != nil {
		return err
	}

	return executeCommand(syncService)
}

func executeCommand(syncService *syncsvc.SynchronizerService) error {
	logger.WithFields(logging.LogEntryContext(logger.Fields{})).Debug()

	if command == "" {
		return runInteractive(syncService)
	}

	// Execute the command
	systemService, err := syncService.GetSystemService()
	if err != nil {
		return err
	}

	processCommand(systemService)

	return nil
}

func runSingleSynchronizer(name string, syncService *syncsvc.SynchronizerService) error {
	logger.WithFields(logging.LogEntryContext(logger.Fields{})).Debug()
	logger.Info("starting: interactive mode")

	// Create the service
	synchronizer := syncService.RetrieveSynchronizer(name)

	if *remote {
		files, err := synchronizer.ListRemote()
		if err != nil {
			return err
		}
		printRemoteFiles(files)
		return nil
	}

	// Run the synchronizer
	synchronizer.Run()

	// Get the service status and print the status to the log
	types.GetServiceStatus().GetSynchronizerStatus(*profile).Log()
}

// runAsService runs the underlying service as a system service instead of a single
// run on the command line
func runAsService(syncService *syncsvc.SynchronizerService) error {
	logger.WithFields(logging.LogEntryContext(logger.Fields{})).Debug()

	systemService, err := syncService.GetSystemService()
	if err != nil {
		return err
	}

	// Configure the logger
	/*
		if err := logging.InitializeLogging(logging.LogConfig{
			Type:        logging.FILE,
			Format:      logging.TEXT,
			MinLogLevel: logging.DEBUG,
			Parameters: map[string]interface{}{
				"filename": filepath.Join(conf.GetSynchronizerConf().WorkingFolder, "syncservice.log"),
			},
		}); err != nil {
			return fmt.Errorf("error running service: %v", err.Error())
		}
*/

/*
	stopChannel := createStopChannel()

	if err := systemService.Run(); err != nil {
		logger.Fatal("fatal error: execution will complete")
		return fmt.Errorf("critical error: %s", err.Error())
	}

	// Capture the shutdown signal and stop the service
	<-stopChannel
	close(stopChannel)

	// Stop the service gracefully
	systemService.Stop()

	return nil
}
*/

func runInteractiveService(syncService *syncsvc.SynchronizerService, profiles []string, cfg conf.SynchronizerConf, logger *slog.Logger) error {
	var errs []error
	var lock sync.Mutex
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(profiles))

	for _, profile := range profiles {
		go func(currentProfile string) {
			defer waitGroup.Done()

			synchronizer := syncService.RetrieveSynchronizer(profile)

			if err := runListRemote(synchronizer); err != nil {
				lock.Lock()
				defer lock.Unlock()
				errs = append(errs, err)
				return
			}

			// Run the synchronizer
			synchronizer.Run()

			// Get the service status and print the status to the log
			types.GetServiceStatus().GetSynchronizerStatus(profile).Log()
		}(profile)
	}
	waitGroup.Wait()

	return errors.Join(errs...)
}

func runListRemote(synchronizer *types.Synchronizer, cfg conf.SynchronizerConf, logger *slog.Logger) error {
	if *remote {
		files, err := synchronizer.ListRemote()
		if err != nil {
			return err
		}

		printRemoteFiles(files)
	}

	return nil
}

/*

func processCommand(systemService service.Service) {
	configuration := conf.GetSynchronizerConf()
	switch command {
	case "install":
		fmt.Printf("Installing the %s service\n", configuration.ServiceConfiguration.Name)
		if err := systemService.Install(); err != nil {
			logger.Errorf("error: occurred installing the service: %s", err.Error())
		} else {
			fmt.Printf("The %s service installed successfully\n", configuration.ServiceConfiguration.DisplayName)
			fmt.Printf("Attempting to start the '%s' service\n", configuration.ServiceConfiguration.DisplayName)
			err = systemService.Start()
			if err != nil {
				logger.Errorf("error occurred starting the service: %s", err.Error())
			} else {
				fmt.Printf("The %s service started successfully\n", configuration.ServiceConfiguration.DisplayName)
			}
		}
	case "uninstall":
		if err := systemService.Uninstall(); err != nil {
			logger.Errorf("error: occurred un-installing the service: %s", err.Error())
		}
	case "start":
		if err := systemService.Start(); err != nil {
			logger.Errorf("error occurred starting the service: %s", err.Error())
		}
	case "stop":
		if err := systemService.Stop(); err != nil {
			logger.Errorf("error occurred stopping the service: %s", err.Error())
		}
	default:
		logger.Errorf("'%s' is an unsupported or invalid command and no action can be taken", command)
	}
}

*/

func initializeCmdLineParameters(cfg *conf.SynchronizerConf) {
	// Set configuration values based on the flags that have been set
	cfg.ExecFlags.FileOverwrite = *overwrite
	cfg.ExecFlags.Simulation = *simulate

	// Determine if a command was passed on the command line
	for index := 1; index < len(os.Args); index++ {
		// Determine if this is a command or a flag
		if !strings.HasPrefix(os.Args[index], "-") {
			command = os.Args[index]
			// Only one command can be passed and the first one found will be used
			break
		}
	}
}

/*
func createStopChannel() chan os.Signal {
	stopChannel := make(chan os.Signal, 5)
	signal.Notify(stopChannel, os.Interrupt)
	signal.Notify(stopChannel, syscall.SIGTERM)
	signal.Notify(stopChannel, syscall.SIGINT)

	return stopChannel
}
*/

func printRemoteFiles(synchronizedFileList map[string]bool) {
	for fileName := range synchronizedFileList {
		fmt.Println(fileName)
	}
}

/**********************************************************************************/
