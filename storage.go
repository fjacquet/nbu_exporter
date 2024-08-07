package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

// getStorage retrieves storage unit information from the NBU server and updates the provided disks map with the free and used capacity for each storage unit.
// The function makes a GET request to the /storage/storage-units endpoint on the NBU server, and unmarshals the JSON response into a Storages struct.
// It then iterates through the storage units, skipping any tape storage units, and updates the disks map with the free and used capacity for each storage unit.
// The disks map is keyed by a string in the format "<storage unit name>|<storage server type>|free" or "<storage unit name>|<storage server type>|used".
func getStorage(disks map[string]float64) {

	var s Storages

	url := nbuRoot + "/storage/storage-units"

	// Create a Resty Client
	Client := resty.New()
	Client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	Client.SetTimeout(1 * time.Minute)

	resp, _ := Client.R().
		SetQueryParams(map[string]string{
			"page[limit]":  "100",
			"page[offset]": "0",
		}).
		SetHeader("Accept", Cfg.NbuServer.ContentType).
		SetHeader("Authorization", Cfg.NbuServer.APIKey).
		Get(url)

	unmarshalError := json.Unmarshal(resp.Body(), &s)

	if unmarshalError != nil {
		fmt.Println(unmarshalError)
		os.Exit(1)
	}

	for i := 0; i < len(s.Data); i++ {
		// fmt.Println(s.Data[i])

		stuName := s.Data[i].Attributes.Name
		free := s.Data[i].Attributes.FreeCapacityBytes
		used := s.Data[i].Attributes.UsedCapacityBytes
		// total := s.Data[i].Attributes.TotalCapacityBytes
		storageType := s.Data[i].Attributes.StorageType
		stuType := s.Data[i].Attributes.StorageServerType
		if storageType != "Tape" {
			key := stuName + "|" + stuType + "|free"
			disks[key] = float64(free)
			key = stuName + "|" + stuType + "|used"
			disks[key] = float64(used)
			// key = stuName + "|" + stuType + "|total"
			// disks[key] = float64(total)
		}

	}

}
