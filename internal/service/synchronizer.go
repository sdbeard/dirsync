// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package service

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/robfig/cron/v3"
	"github.com/sdbeard/dirsync/internal/conf"
	"github.com/sdbeard/dirsync/internal/types"
	"github.com/sdbeard/go-supportlib/aws/service/s3"
	"github.com/sdbeard/go-supportlib/common/files"
	"github.com/sdbeard/go-supportlib/common/logging"
	"github.com/sdbeard/go-supportlib/common/util"
	logger "github.com/sirupsen/logrus"
)

/**********************************************************************************/

// NewSynchronizer creates and returns a pointer to a new DirectorySyncService
// object. Otherwise an error is returned.
func NewSynchronizer(profile types.Profile) (*Synchronizer, error) {
	// Create a new Synchronizer
	synchronizer := &Synchronizer{
		profile:              profile,
		scheduler:            cron.New(),
		synchronizedFileList: make(map[string]bool),
		processing:           false,
		shuttingDown:         false,
	}

	return synchronizer, synchronizer.initialize()
}

/***** Synchronizer ***************************************************************/

// Synchronizer is the main object that does the directory synchronization. It
// contains the objects required to synchronize the configured directory/directories
type Synchronizer struct {
	profile              types.Profile
	status               *types.SynchronizerStatus
	scheduler            *cron.Cron
	synchronizedFileList map[string]bool
	syncJobs             chan string
	errorJobs            chan string
	jobsCompleted        chan bool
	parentChan           chan bool
	nextRunTime          time.Time
	processing           bool
	shuttingDown         bool
}

/***** exported functions *********************************************************/

// Start starts the synchronizer. If this is a scheduled task then the scheduler is
// started
func (synchronizer *Synchronizer) Startup(finishChan chan bool) {
	synchronizer.parentChan = finishChan

	if synchronizer.profile.RunAtStartup {
		synchronizer.Run()
	}

	synchronizer.setNextRunTime()
	synchronizer.status.Update(func(status *types.SynchronizerStatus) {
		status.NextRunTime = synchronizer.nextRunTime
	})

	synchronizer.scheduler.Start()
}

// Stop gracefully shuts down the synchronizer
func (synchronizer *Synchronizer) Shutdown() {
	// Stop the scheduler
	synchronizer.scheduler.Stop()

	if synchronizer.status.IsRunning {
		// Stop the channels
		synchronizer.closeChannels()
	}

	// Signal completion
	if synchronizer.parentChan != nil {
		synchronizer.parentChan <- true
	}
}

// ListRemote provides a list of the files that are located in the remote location
func (synchronizer *Synchronizer) ListRemote() (map[string]bool, error) {
	defer synchronizer.cleanup()

	// Set the environment
	if !synchronizer.processing {
		synchronizer.setEnvironment(false)
		synchronizer.processing = true
	}

	logger.Infof("Retrieving remote file listing @ %s", time.Now().Format("2006-01-02 15:04:05 MST"))

	// Populate the list of synchronized files
	if err := synchronizer.buildRemoteFileList(); err != nil {
		logger.WithFields(logging.LogEntryContext(map[string]interface{}{})).Error(err.Error())
		return nil, err
	}

	return synchronizer.synchronizedFileList, nil
}

// Run starts the synchronization process
func (synchronizer *Synchronizer) Run() {
	defer synchronizer.cleanup()

	// Set the environment
	synchronizer.setEnvironment(false)
	synchronizer.processing = true
	synchronizer.synchronizedFileList = make(map[string]bool)

	// Set the synchronizer status
	synchronizer.resetSynchronizerStatus()
	synchronizer.status.Update(func(status *types.SynchronizerStatus) {
		status.IsRunning = true
		status.StartTime = time.Now()
	})
	logger.Infof("Starting new synchronization run @ %s", synchronizer.status.StartTime.Format("2006-01-02 15:04:05 MST"))

	// Populate the remote list of files
	if err := synchronizer.buildRemoteFileList(); err != nil {
		logger.WithFields(logging.LogEntryContext(map[string]interface{}{})).Error(err.Error())
		return
	}

	// Build the search file list
	if err := synchronizer.buildLocalFileList(); err != nil {
		logger.WithFields(logging.LogEntryContext(map[string]interface{}{})).Error(err.Error())
		return
	}

	// Take the list files to be synchronized and loop through the 'search.dat' file
	// to synchronize those specific files
	synchronizer.synchronizeFiles()

	//  End the processing time
	synchronizer.status.Update(func(status *types.SynchronizerStatus) {
		status.IsRunning = false
		status.EndTime = time.Now()
		status.ProcessingDuration = status.EndTime.Sub(status.StartTime)
	})
	// TODO: Persist the synchronizer status???
}

/**********************************************************************************/

// initialize configures the synchronizer to run
func (synchronizer *Synchronizer) initialize() error {
	// Reset the status object
	synchronizer.resetSynchronizerStatus()

	// Configure the scheduler
	_, err := synchronizer.scheduler.AddFunc(synchronizer.profile.ScheduleDef, synchronizer.scheduledTask)

	return err
}

func (synchronizer *Synchronizer) scheduledTask() {
	// Starting the syncService
	synchronizer.Run()

	synchronizer.setNextRunTime()

	if !synchronizer.shuttingDown {
		synchronizer.status.Update(func(status *types.SynchronizerStatus) {
			status.NextRunTime = synchronizer.nextRunTime
		})
	}
}

// setEnvironment looks for an optional file in the current execution folder and
// loads the contents of the '.environ' file into the local execution environment.
// This is especially helpful for loading the AWS credentials that will be used to
// synchronize data to S3
func (synchronizer *Synchronizer) setEnvironment(clear bool) {
	// Retrieve and load the .environ file if it exists in the current folder
	// TODO: This is a magic string
	//environFile, err := os.Open(fmt.Sprintf("%s%s.environ", conf.GetSynchronizerConf().ExecFlags.ExecutionFolder, string(os.PathSeparator)))
	environFile, err := os.Open(filepath.Join(conf.GetSynchronizerConf().WorkingFolder, ".environ"))
	if err != nil {
		logger.WithFields(logging.LogEntryContext(map[string]interface{}{})).Error(err.Error())
		return
	}
	defer environFile.Close()

	if clear {
		synchronizer.cleanEnviron(environFile)
		return
	}

	synchronizer.readEnviron(environFile)
}

// clearEnviron unsets all of the variables that were originally set by from the
// environ file
func (synchronizer *Synchronizer) cleanEnviron(environ *os.File) {
	// Load the environment from the local execution environment
	scanner := bufio.NewScanner(environ)

	for scanner.Scan() {
		envVar := strings.Split(scanner.Text(), "=")
		if len(envVar) == 2 {
			if err := os.Unsetenv(envVar[0]); err != nil {
				logger.Error(err.Error())
			}
		}
	}
}

// readEnvironFile does the work of reading the local envfironmnet file and adds
// them to the execution environment
func (synchronizer *Synchronizer) readEnviron(environ *os.File) {
	// Load the environment from the local execution environment
	scanner := bufio.NewScanner(environ)

	for scanner.Scan() {
		envVar := strings.Split(scanner.Text(), "=")
		if len(envVar) == 2 {
			if err := os.Setenv(envVar[0], envVar[1]); err != nil {
				logger.Error(err.Error())
			}
		}
	}
}

// buildRemoteFileList gathers the list of files that is currently on S3 and
// populates a map with the file name and path in order to easily know if the file
// already exists on S3
func (synchronizer *Synchronizer) buildRemoteFileList() error {
	contents, err := s3.GetBucketKeys(
		synchronizer.profile.S3Config.Connect,
		synchronizer.profile.S3Config.Bucket,
		"",
	)
	if err != nil {
		return err
	}

	// Populate the last of synchronized files
	for _, object := range contents {
		synchronizer.synchronizedFileList[strings.ToLower(strings.Replace(object.Key, "/", string(os.PathSeparator), -1))] = true
	}

	return nil
}

// buildLocalFileList execute step 1 of the service execution process which gathers
// a file list of the available files to synchronize in the the search folder
func (synchronizer *Synchronizer) buildLocalFileList() error {
	sourceFolder := strings.ToLower(synchronizer.profile.SourceFolder)
	if !strings.HasSuffix(sourceFolder, string(os.PathSeparator)) {
		sourceFolder = fmt.Sprintf("%s%s", sourceFolder, string(os.PathSeparator))
	}

	// Check if the search folder exists
	if !util.FileExists(sourceFolder) {
		return fmt.Errorf("the source folder path (%s) does not exist", sourceFolder)
	}

	files := make([]string, 0)
	if err := filepath.Walk(sourceFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Check if the current path is included
			if synchronizer.isIncluded(path) {
				files = append(files, path)
			}
		} else if !synchronizer.profile.Recursive && (path != sourceFolder) {
			return filepath.SkipDir
		}

		return nil
	}); err != nil {
		return err
	}

	return synchronizer.writeSourceFiles(sourceFolder, files)
}

func (synchronizer *Synchronizer) writeSourceFiles(sourceFolder string, files []string) error {
	// Create the file to write the file entries to
	searchFile, err := os.Create(fmt.Sprintf("%s%ssearch.dat",
		conf.GetSynchronizerConf().WorkingFolder, string(os.PathSeparator))) // TODO: Get rid of magic string
	if err != nil {
		return err
	}
	defer searchFile.Close()

	// Loop through the search files
	for _, file := range files {
		syncFile := strings.Replace(strings.ToLower(file), sourceFolder, "", -1)
		if _, ok := synchronizer.synchronizedFileList[syncFile]; ok && !conf.GetSynchronizerConf().ExecFlags.FileOverwrite {
			continue
		}

		if _, err := searchFile.WriteString(fmt.Sprintf("%s\n", file)); err != nil {
			logger.WithFields(logging.LogEntryContext(map[string]interface{}{})).Error(err.Error())
		}
	}

	return nil
}

// synchronizeFiles is a management function that orchestrates the copying of the
// files listed in 'search.dat' to the destination. In doing so it, manages the
// number of parallel copy operations defined in Max Concurrent Copies and the
// throttling of the copy operation to S3 using passed in configuration parameters
// or standard defaults if not configured.
func (synchronizer *Synchronizer) synchronizeFiles() {
	synchronizer.createChannels()
	defer synchronizer.closeChannels()

	go synchronizer.startErrorListener()

	// Create and start the max concurrent copies Go functions
	for index := 0; index < synchronizer.profile.FileCopyOptions.MaxConcurrentCopies; index++ {
		go synchronizer.synchronize(index)
	}

	// Read the search file
	synchronizer.readSearchFile()
}

func (synchronizer *Synchronizer) createChannels() {
	// Create the required channels for the sync operation
	synchronizer.syncJobs = make(chan string)
	synchronizer.errorJobs = make(chan string)
	synchronizer.jobsCompleted = make(chan bool)
}

func (synchronizer *Synchronizer) closeChannels() {
	// Close the syncJobs channel
	close(synchronizer.syncJobs)

	closedSynchronizers := 0
	for range synchronizer.jobsCompleted {
		closedSynchronizers++
		if closedSynchronizers == synchronizer.profile.FileCopyOptions.MaxConcurrentCopies {
			break
		}
	}

	// Close the error channel
	close(synchronizer.errorJobs)
	<-synchronizer.jobsCompleted

	// Close the jobs completed channel
	close(synchronizer.jobsCompleted)
}

func (synchronizer *Synchronizer) readSearchFile() {

	// Open and iterate over the search.dat file and add each entry to the syncJobs
	// channel
	searchFile, err := os.Open(fmt.Sprintf(
		"%s%ssearch.dat",
		conf.GetSynchronizerConf().WorkingFolder,
		string(os.PathSeparator),
	))
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer searchFile.Close()

	scanner := bufio.NewScanner(searchFile)
	for scanner.Scan() {
		// Send the line from the search file to the syncJobs channel
		synchronizer.syncJobs <- scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		logger.WithFields(logging.LogEntryContext(map[string]interface{}{})).Error(err.Error())
	}
}

func (synchronizer *Synchronizer) synchronize(threadID int) {
	// Set the current thread name
	threadName := fmt.Sprintf("Thread-%d", threadID)

	// Loop over the job channel
	for syncJob := range synchronizer.syncJobs {
		fileInfo, err := os.Stat(syncJob)
		if err != nil {
			synchronizer.errorJobs <- syncJob
			continue
		}

		// Update the active transfer
		synchronizer.status.UpdateActiveTransfer(threadName, fileInfo.Name())

		// Add the new file to the list of active transfers
		syncError := synchronizer.synchronizeFile(syncJob, threadName)
		if syncError != nil {
			// Send the syncJob to the error channel
			synchronizer.errorJobs <- syncJob
			continue
		}
		synchronizer.status.IncrementSyncJob()
	}

	synchronizer.status.RemoveActiveTransfer(threadName)

	// Signal that the go function is exiting by adding a message to the
	// jobs completed channel
	synchronizer.jobsCompleted <- true
}

// synchronizeFile takes the full path of the file to be synchronized to S3 as read
// from the 'search.dat' file. It manages the execution of the file copy and returns
// an error if one occurs, otherwise returns nil
// TODO: Need to add logic to throttle the copy operation from the configuration
func (synchronizer *Synchronizer) synchronizeFile(fileToSync string, threadName string) error {
	startTime := time.Now()

	// Build the S3 PATH value - based on the original search folder
	s3Path := filepath.Dir(fileToSync)
	source := synchronizer.profile.SourceFolder
	if strings.Contains(strings.ToLower(s3Path), strings.ToLower(source)) {
		s3Path = s3Path[len(source):]
	}
	s3Path = strings.TrimPrefix(s3Path, string(os.PathSeparator))

	// **Windows Mainly** Flip the back slashes to a forward slash since they impact
	// the path in the S3 bucket
	s3Path = strings.Replace(s3Path, "\\", "/", -1)

	// Create the copy file request
	copyFileRequest := files.CopyFileRequest{
		Source: files.NewLocalFile(filepath.Dir(fileToSync), filepath.Base(fileToSync)),
		Destination: files.NewS3File(
			synchronizer.profile.S3Config.Connect,
			synchronizer.profile.S3Config.Bucket,
			s3Path,
			filepath.Base(fileToSync),
			synchronizer.profile.S3Config.StorageClass.String(),
		),
	}

	// Check if we need to set the connection throttling
	if synchronizer.profile.FileCopyOptions.ThrottleSync {
		copyFileRequest.Throttled = &files.CopyThrottleConfiguration{
			ThrottleBucketSize: synchronizer.profile.FileCopyOptions.ThrottleBucketSizeKB,
		}
	}

	// Check if this is a simulation
	if !conf.GetSynchronizerConf().ExecFlags.Simulation {
		// Copy the fileToSync to S3 using the CopyFileRequest
		if _, err := files.CopyFile(copyFileRequest); err != nil {
			return err
		}
	}

	prefix := ""
	if conf.GetSynchronizerConf().ExecFlags.Simulation {
		prefix = "SIMULATED: "
	}

	logger.WithFields(map[string]interface{}{
		"thread": threadName,
		"file":   fileToSync,
	}).Infof("%s'%s' synchronized successfully in %.3f seconds",
		prefix,
		copyFileRequest.Source.Name(),
		time.Since(startTime).Seconds(),
	)

	return nil
}

func (synchronizer *Synchronizer) startErrorListener() {
	// Start the error go function for managing the file copy operations that encounter an error
	errorFile, err := os.Create(fmt.Sprintf("%s%serrors.dat",
		conf.GetSynchronizerConf().WorkingFolder, string(os.PathSeparator)))
	if err != nil {
		logger.WithFields(logging.LogEntryContext(map[string]interface{}{})).Error(err.Error())
		return
	}
	defer errorFile.Close()

	for errorJob := range synchronizer.errorJobs {
		errorFile.WriteString(fmt.Sprintf("%s\n", errorJob))
	}

	// Signal this process complete once the errorJobs channel is closed
	synchronizer.jobsCompleted <- true
}

// isIncluded takes the path parameter and the FileOperationRequest to determine if
// the current path value should be included in an operation
func (synchronizer *Synchronizer) isIncluded(path string) bool {
	include := true
	if len(synchronizer.profile.Extensions) > 0 {
		include = synchronizer.hasExtension(path, synchronizer.profile.Extensions)
	}

	exclude := false
	if len(synchronizer.profile.Exclusions) > 0 {
		exclude = synchronizer.hasExtension(path, synchronizer.profile.Exclusions)
	}

	return include && !exclude
}

// hasExtension takes the passed in path and extensions and checks if the path
// matches one of the parameters and returns true if the path is matched , otherwise
// false
func (synchronizer *Synchronizer) hasExtension(path string, extensions []string) bool {
	for _, ext := range extensions {
		if !strings.HasPrefix(ext, ".") {
			ext = fmt.Sprintf(".%s", ext)
		}

		pathExt := filepath.Ext(path)
		if strings.EqualFold(pathExt, ext) {
			return true
		}
	}

	return false
}

func (synchronizer *Synchronizer) resetSynchronizerStatus() {
	newSyncStatus := types.CreateSynchronizerStatus(synchronizer.profile)
	synchronizer.status = newSyncStatus
	types.GetServiceStatus().Add(newSyncStatus)
}

// setNextRunTime calculates the next run time for the service
func (synchronizer *Synchronizer) setNextRunTime() {
	expression := cronexpr.MustParse(synchronizer.profile.ScheduleDef)
	synchronizer.nextRunTime = expression.Next(time.Now())
	logger.Infof("The next synchronization run will be at: %s", synchronizer.nextRunTime.Format("2006-01-02 15:04:05 MST"))
}

func (synchronizer *Synchronizer) cleanup() {
	synchronizer.setEnvironment(true)
	synchronizer.processing = false
}
