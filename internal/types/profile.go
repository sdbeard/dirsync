// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package types

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
	Target          int             `json:"target"`
	Recursive       bool            `json:"recursive"`
	RunAtStartup    bool            `json:"runatstartup"`
}

/**********************************************************************************/
