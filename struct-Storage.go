package main

type Storage struct {
	Data struct {
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"links"`
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Name                       string `json:"name"`
			StorageType                string `json:"storageType"`
			StorageSubType             string `json:"storageSubType"`
			StorageServerType          string `json:"storageServerType"`
			UseAnyAvailableMediaServer bool   `json:"useAnyAvailableMediaServer"`
			Accelerator                bool   `json:"accelerator"`
			InstantAccessEnabled       bool   `json:"instantAccessEnabled"`
			IsCloudSTU                 bool   `json:"isCloudSTU"`
			FreeCapacityBytes          int64  `json:"freeCapacityBytes"`
			TotalCapacityBytes         int64  `json:"totalCapacityBytes"`
			UsedCapacityBytes          int64  `json:"usedCapacityBytes"`
			MaxFragmentSizeMegabytes   int    `json:"maxFragmentSizeMegabytes"`
			MaxConcurrentJobs          int    `json:"maxConcurrentJobs"`
			OnDemandOnly               bool   `json:"onDemandOnly"`
		} `json:"attributes"`
		Relationships struct {
			DiskPool struct {
				Links struct {
					Related struct {
						Href string `json:"href"`
					} `json:"related"`
				} `json:"links"`
				Data struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			} `json:"diskPool"`
		} `json:"relationships"`
	} `json:"data"`
}
