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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/sdbeard/dirsync/internal/conf"
	syncsvc "github.com/sdbeard/dirsync/internal/service"
	"github.com/sdbeard/dirsync/internal/types"
	"github.com/sdbeard/go-supportlib/common/logging"
	"github.com/sdbeard/service"
	logger "github.com/sirupsen/logrus"
)

var (
	profile   = flag.String("profile", "", "specifies a base profile to use/process; ignored in a non-interactive session")
	remote    = flag.Bool("remote", false, "interactive only parameter: retrieves the remote directory and file listing from configured remote location; ignored if a profile is not provided)")
	simulate  = flag.Bool("simulate", false, "tells the system to only simulate synchronizing files. No file are acutally copied to S3. (For development purposes only (only valid in stand-alone mode)")
	overwrite = flag.Bool("overwrite", false, "tells the system to overwrite a file on S3.")
	env       = "local"

	command     string
	compileDate string
	version     string
)

/**********************************************************************************/

func init() {
	if err := conf.LoadConfiguration(); err != nil {
		panic(err)
	}
	initializeCmdLineParameters()
}

func main() {
	//configuration := types.GetConfiguration()
	if !service.Interactive() {
		// Set the configuration so that any stand-alone mode flags are set to false or empty
		conf.GetConfiguration().ExecFlags.NoSchedule = false
		conf.GetConfiguration().ExecFlags.Simulation = false

		// Run the service
		if err := runAsService(); err != nil {
			logger.Fatal(err.Error())
			os.Exit(99)
		}

		os.Exit(0)
	}

	if err := logging.InitializeDefaultLogging(); err != nil {
		panic(err)
	}

	logger.Info("KPS Directory Sync to S3 v2.0.0")
	logger.WithFields(logging.LogEntryContext(map[string]interface{}{
		"App Version": version,
		"Build":       compileDate,
		"Environment": env,
		"GO Version":  runtime.Version(),
		"PID":         os.Getpid(),
	})).Infof("Runtime configuration")

	syncService, err := syncsvc.NewSynchronizerService()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(99)
	}

	// Execute the command
	if command != "" {
		systemService, err := syncService.GetSystemService()
		if err != nil {
			logger.Error(err.Error())
			os.Exit(99)
		}
		processCommand(systemService)
		os.Exit(0)
	}

	runInteractive(syncService)

	logger.Info("dirsynctos3 service has completely shutdown")

	// Exit the application
	os.Exit(0)
}

/**********************************************************************************/

func runInteractive(syncService *syncsvc.SynchronizerService) {
	logger.Info("Starting dirsynctos3 service in standard mode (i.e. not as a service)....")

	// Run all of the configured profiles
	if *profile != "" {
		if _, ok := conf.GetConfiguration().SyncProfiles[*profile]; !ok {
			logger.Error("profile not found")
			return
		}
		runSingleSynchronizer(*profile, syncService)
		return
	}

	// Run the service from an interactive space
	runInteractiveService(syncService)

	// Get the service status and print the status to the log
	types.GetServiceStatus().Log()
}

func runSingleSynchronizer(name string, syncService *syncsvc.SynchronizerService) {
	logger.Info("Starting dirsynctos3 service in standard mode (i.e. not as a service)....")

	// Create the service
	synchronizer := syncService.RetrieveSynchronizer(name)

	if *remote {
		files, err := synchronizer.ListRemote()
		if err != nil {
			logger.Error(err.Error())
			return
		}
		printRemoteFiles(files)
		return
	}

	// Run the synchronizer
	synchronizer.Run()

	// Get the service status and print the status to the log
	types.GetServiceStatus().GetSynchronizerStatus(*profile).Log()
}

// runAsService runs the underlying service as a system service instead of a single
// run on the command line
func runAsService() error {
	syncService, err := syncsvc.NewSynchronizerService()
	if err != nil {
		return err
	}

	systemService, err := syncService.GetSystemService()
	if err != nil {
		return err
	}

	// Configure the logger
	if err := logging.InitializeLogging(logging.LogConfig{
		Type:        logging.FILE,
		Format:      logging.TEXT,
		MinLogLevel: logging.DEBUG,
		Parameters: map[string]interface{}{
			"filename": fmt.Sprintf("%s%ssyncservice.log", conf.GetConfiguration().ExecFlags.ExecutionFolder, string(os.PathSeparator)),
		},
	}); err != nil {
		return fmt.Errorf("error running service: %v", err.Error())
	}

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

// runAsService runs the underlying service as a system service instead of a single
// run on the command line
func runInteractiveService(syncService *syncsvc.SynchronizerService) {
	stopChannel := createStopChannel()

	go syncService.Start(nil)

	// Capture the shutdown signal and stop the service
	<-stopChannel
	close(stopChannel)

	// Stop the service gracefully
	syncService.Stop(nil)

	logger.Info("dirsynctos3 service has completely shutdown")
}

func processCommand(systemService service.Service) {
	configuration := conf.GetConfiguration()
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

// initializeCmdLineParameters updates the DirectorySyncS3ServiceConfiguration
// parameter with the flags read from the command line
func initializeCmdLineParameters() {
	configuration := conf.GetConfiguration()

	// Parse the command line flags
	flag.Parse()

	if !configuration.ExecFlags.FileOverwrite {
		configuration.ExecFlags.FileOverwrite = *overwrite
	}

	// Set the execution flags
	//configuration.ExecFlags.NoSchedule = cmdNoSchedule
	configuration.ExecFlags.Simulation = *simulate

	// Determine if a command was passed on the command line, the acceptable
	// commands only relate to service management install|uninstall|help are the
	// primary commands. If not simply execute the service outside of the service
	// construct
	for index := 1; index < len(os.Args); index++ {
		// Determine if this is a command or a flag
		if !strings.HasPrefix(os.Args[index], "-") {
			command = os.Args[index]
			// Only one command can be passed and the first one found will be used
			break
		}
	}
}

func createStopChannel() chan os.Signal {
	stopChannel := make(chan os.Signal, 5)
	signal.Notify(stopChannel, os.Interrupt)
	signal.Notify(stopChannel, syscall.SIGTERM)
	signal.Notify(stopChannel, syscall.SIGINT)

	return stopChannel
}

func printRemoteFiles(synchronizedFileList map[string]bool) {
	for fileName := range synchronizedFileList {
		fmt.Println(fileName)
	}
}
