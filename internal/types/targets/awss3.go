// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package targets

import (
	"fmt"
	"os"
	"strings"

	"github.com/sdbeard/go-supportlib/aws/service/s3"
	"github.com/sdbeard/go-supportlib/common/files"
	"github.com/sdbeard/go-supportlib/common/logging"
	logger "github.com/sirupsen/logrus"
)

/**********************************************************************************/

// NewAWSS3Target
func NewAWSS3Target(conf s3.Configuration) *AWSS3Target {
	return &AWSS3Target{
		S3Conf: conf,
	}
}

/**********************************************************************************/

type AWSS3Target struct {
	S3Conf s3.Configuration
}

/***** Target interface implementation*********************************************/

// RetrieveTargetFileList
func (target *AWSS3Target) RetrieveTargetFileList() ([]string, error) {
	logger.WithFields(logging.LogEntryContext(logger.Fields{})).Debug()

	s3Contents, err := s3.GetBucketKeys(
		target.S3Conf.Connect,
		target.S3Conf.Bucket,
		"",
	)
	if err != nil {
		return nil, err
	}

	// Populate the list of synchronized files
	fileList := make([]string, len(s3Contents))
	for index := 0; index < len(s3Contents); index++ {
		fileList[index] = strings.ToLower(strings.Replace(s3Contents[index].Key, "/", string(os.PathSeparator), -1))
	}

	return fileList, nil
}

func (target *AWSS3Target) SyncFile(file string) error {
	logger.WithFields(logging.LogEntryContext(logger.Fields{})).Debug()
	return fmt.Errorf("not implemented exception")
}

func (target *AWSS3Target) RetrieveTarget() files.File {
	logger.WithFields(logging.LogEntryContext(logger.Fields{})).Debug()
	logger.Error("not implemented exception")

	return nil
}

/**********************************************************************************/

/***** AWSS3TargetConf ************************************************************/

type AWSS3TargetConf struct {
	S3Conf s3.Configuration `json:"targetconf"`
}

/**********************************************************************************/
