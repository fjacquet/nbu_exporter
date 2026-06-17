package models

// TapeDrives is the response from GET /storage/drives (JSON:API).
type TapeDrives struct {
	Data []struct {
		Attributes struct {
			DriveName   string `json:"driveName"`
			DriveStatus string `json:"driveStatus"` // DRIVE_STATUS_UP|DOWN|MIXED|DISABLED
			DriveType   string `json:"driveType"`   // DT_HCART, DT_DLT, ...
			RobotType   string `json:"robotType"`   // TLD, ACS, NOT_ROBOTIC, NA
			RobotNumber int    `json:"robotNumber"`
			DeviceHost  string `json:"deviceHost"`
		} `json:"attributes"`
	} `json:"data"`
}

// TapeMedia is the response from GET /storage/tape-media (JSON:API).
type TapeMedia struct {
	Data []struct {
		Attributes struct {
			Barcode     string `json:"barcode"`
			MediaType   string `json:"mediaType"`   // HCART, DLT, HC_CLN, ...
			MediaStatus string `json:"mediaStatus"` // free-form, e.g. "ACTIVE MULTIPLEXED"
			RobotType   string `json:"robotType"`
			RobotNumber int    `json:"robotNumber"`
		} `json:"attributes"`
	} `json:"data"`
}

// RobotDeviceHosts is the response from GET /storage/robots-device-hosts (JSON:API).
// Each entry is a configured robot device host; data[].id is the hostname.
type RobotDeviceHosts struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}
