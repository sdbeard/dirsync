// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package types

import "strings"

/*****enum: LocationType **********************************************************/

type LocationType int

const (
	// LOCAL represents a location type that is local to the running service
	LOCAL LocationType = iota
	// AWSS3 represents an AWS S3 bucket location
	AWSS3
	// NONE represents an empty/noop location
	NONE
)

var (
	locations = []string{"local", "awss3", "none"}
)

// String returns the string representation for the location type
func (location LocationType) String() string {
	return locations[location]
}

// LocationTypeFromString returns the Location Type from the locationTypeString
func LocationTypeFromString(locationTypeString string) LocationType {
	for index, location := range locations {
		if strings.EqualFold(location, locationTypeString) {
			return LocationType(index)
		}
	}

	return NONE
}

/**********************************************************************************/
