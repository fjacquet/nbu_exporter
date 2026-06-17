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

// tapePagination is the cursor-style meta block on the v12.0+ tape/disk-pool
// endpoints; a Next of 0 marks the last page.
type tapePagination struct {
	Pagination struct {
		Next int `json:"next"`
	} `json:"pagination"`
}

// TapeVolumePools is the response from GET /storage/tape-volume-pools (API v12.0+).
type TapeVolumePools struct {
	Data []struct {
		Attributes struct {
			VolumePoolName     string `json:"volumePoolName"`
			Description        string `json:"description"`
			PartiallyFullMedia int    `json:"partiallyFullMedia"`
			PoolType           string `json:"poolType"`
		} `json:"attributes"`
	} `json:"data"`
	Meta tapePagination `json:"meta"`
}

// DiskVolume is one volume entry nested inside a disk pool (API v12.0+).
type DiskVolume struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	State string `json:"state"` // UP / DOWN / UNKNOWN
}

// DiskPools is the response from GET /storage/disk-pools (API v12.0+).
type DiskPools struct {
	Data []struct {
		Attributes struct {
			Name            string       `json:"name"`
			SType           string       `json:"sType"`
			StorageCategory string       `json:"storageCategory"` // ADVANCED_DISK / CLOUD / MSDP / OPEN_STORAGE
			DiskPoolState   string       `json:"diskPoolState"`   // UP / DOWN / TRANSIENT
			DiskVolumes     []DiskVolume `json:"diskVolumes"`
		} `json:"attributes"`
	} `json:"data"`
	Meta tapePagination `json:"meta"`
}
