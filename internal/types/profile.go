// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package types

import (
	"encoding/json"

	"github.com/sdbeard/go-supportlib/common/files"
)

/***** Profile ********************************************************************/

type Profile struct {
	Source          files.File      `json:"source"`
	Target          files.File      `json:"target"`
	FileCopyOptions FileCopyOptions `json:"filecopyoptions"`
	Extensions      []string        `json:"ext"`
	Exclusions      []string        `json:"exclusions"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
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
		Source         string `json:"source"`
		Target         string `json:"target"`
		TargetLocation string `json:"targetlocation"`
		Alias
	}{
		Source:         profile.Source.Conf(),
		Target:         profile.Target.Conf(),
		TargetLocation: profile.TargetLocation.String(),
		Alias:          (Alias)(profile),
	})
}

// UnmarshalJSON unmarshals JSON string to a Profile object
func (profile *Profile) UnmarshalJSON(data []byte) error {
	type Alias Profile
	aux := &struct {
		Source         string `json:"source"`
		Target         string `json:"target"`
		TargetLocation string `json:"targetlocation"`
		*Alias
	}{
		Alias: (*Alias)(profile),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	source, err := files.GetFileFromPath(aux.Source)
	if err != nil {
		return err
	}
	profile.Source = source

	target, err := files.GetFileFromPath(aux.Target)
	if err != nil {
		return err
	}
	profile.Target = target

	profile.TargetLocation = LocationTypeFromString(aux.TargetLocation)

	return nil
}

/**********************************************************************************/
