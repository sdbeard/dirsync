// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package types

import "encoding/json"

/***** Profile ********************************************************************/

type Profile struct {
	TargetConf      any             `json:"targetconf"`
	FileCopyOptions FileCopyOptions `json:"filecopyoptions"`
	Extensions      []string        `json:"ext"`
	Exclusions      []string        `json:"exclusions"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	SourceFolder    string          `json:"source"`
	ScheduleDef     string          `json:"scheduledef"`
	TargetLocation  LocationType    `json:"targetlocation"`
	Recursive       bool            `json:"recursive"`
	RunAtStartup    bool            `json:"runatstartup"`
}

/***** Marshaler interface definitions ********************************************/

// MarshalJSON marshals the Profile to a JSON string
func (profile Profile) MarshalJSON() ([]byte, error) {
	type Alias Profile
	return json.Marshal(&struct {
		TargetLocation string `json:"targetlocation"`
		Alias
	}{
		TargetLocation: profile.TargetLocation.String(),
		Alias:          (Alias)(profile),
	})
}

// UnmarshalJSON unmarshals JSON string to a Profile object
func (profile *Profile) UnmarshalJSON(data []byte) error {
	type Alias Profile
	aux := &struct {
		TargetLocation string `json:"targetlocation"`
		*Alias
	}{
		Alias: (*Alias)(profile),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	profile.TargetLocation = LocationTypeFromString(aux.TargetLocation)

	return nil
}

/**********************************************************************************/
