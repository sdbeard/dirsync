// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package targets

import "github.com/sdbeard/go-supportlib/common/files"

/**********************************************************************************/

type SyncTarget interface {
	RetrieveTargetFileList() ([]string, error)
	SyncFile(file string) error
	RetrieveTarget() files.File
}

/**********************************************************************************/
