// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package types

/***** FileCopyOptions ************************************************************/

// FileCopyOptions holds all of the fields that optimize the copying of files to S3.
// These options include throttling values, max number of concurrent transfers, etc.
type FileCopyOptions struct {
	ThrottleBucketSizeKB int64 `json:"throttlebucketsizekb"`
	MaxConcurrentCopies  int   `json:"maxconcurrentcopies"`
	ThrottleSync         bool  `json:"throttle"`
}

/***********************************************************************************/
