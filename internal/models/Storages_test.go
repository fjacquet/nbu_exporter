package models

import (
	"encoding/json"
	"os"
	"testing"
)

func TestStoragesUnmarshalJSONWithOptionalFields(t *testing.T) {
	// Read the test fixture with 10.5 API response
	data, err := os.ReadFile("../../testdata/api-10.5/storage-units-response.json")
	if err != nil {
		t.Fatalf("Failed to read test fixture: %v", err)
	}

	var storages Storages
	err = json.Unmarshal(data, &storages)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify we have the expected number of storage units
	if len(storages.Data) != 3 {
		t.Errorf("Expected 3 storage units, got %d", len(storages.Data))
	}

	// Test first storage unit (disk-pool-1) with all optional fields populated
	diskPool := storages.Data[0]
	if diskPool.Attributes.Name != "disk-pool-1" {
		t.Errorf("Expected name 'disk-pool-1', got '%s'", diskPool.Attributes.Name)
	}
	if diskPool.Attributes.StorageType != "DISK" {
		t.Errorf("Expected storageType 'DISK', got '%s'", diskPool.Attributes.StorageType)
	}
	if diskPool.Attributes.FreeCapacityBytes != 5368709120000 {
		t.Errorf("Expected freeCapacityBytes 5368709120000, got %d", diskPool.Attributes.FreeCapacityBytes)
	}
	if diskPool.Attributes.UsedCapacityBytes != 5368709120000 {
		t.Errorf("Expected usedCapacityBytes 5368709120000, got %d", diskPool.Attributes.UsedCapacityBytes)
	}

	// Test new optional fields for disk-pool-1
	if diskPool.Attributes.StorageCategory != "PRIMARY" {
		t.Errorf("Expected storageCategory 'PRIMARY', got '%s'", diskPool.Attributes.StorageCategory)
	}
	if !diskPool.Attributes.ReplicationCapable {
		t.Error("Expected replicationCapable to be true")
	}
	if !diskPool.Attributes.ReplicationSourceCapable {
		t.Error("Expected replicationSourceCapable to be true")
	}
	if diskPool.Attributes.ReplicationTargetCapable {
		t.Error("Expected replicationTargetCapable to be false")
	}
	if !diskPool.Attributes.Independent {
		t.Error("Expected independent to be true")
	}
	if !diskPool.Attributes.Primary {
		t.Error("Expected primary to be true")
	}
	if diskPool.Attributes.ScaleOutEnabled {
		t.Error("Expected scaleOutEnabled to be false")
	}
	if diskPool.Attributes.WormCapable {
		t.Error("Expected wormCapable to be false")
	}
	if diskPool.Attributes.UseWorm {
		t.Error("Expected useWorm to be false")
	}

	// Test second storage unit (cloud-stu-1) with different optional field values
	cloudSTU := storages.Data[1]
	if cloudSTU.Attributes.Name != "cloud-stu-1" {
		t.Errorf("Expected name 'cloud-stu-1', got '%s'", cloudSTU.Attributes.Name)
	}
	if cloudSTU.Attributes.StorageType != "CLOUD" {
		t.Errorf("Expected storageType 'CLOUD', got '%s'", cloudSTU.Attributes.StorageType)
	}
	if cloudSTU.Attributes.StorageCategory != "CLOUD" {
		t.Errorf("Expected storageCategory 'CLOUD', got '%s'", cloudSTU.Attributes.StorageCategory)
	}
	if cloudSTU.Attributes.ReplicationTargetCapable != true {
		t.Error("Expected replicationTargetCapable to be true for cloud storage")
	}
	if !cloudSTU.Attributes.Snapshot {
		t.Error("Expected snapshot to be true for cloud storage")
	}
	if !cloudSTU.Attributes.ScaleOutEnabled {
		t.Error("Expected scaleOutEnabled to be true for cloud storage")
	}
	if !cloudSTU.Attributes.WormCapable {
		t.Error("Expected wormCapable to be true for cloud storage")
	}
	if !cloudSTU.Attributes.UseWorm {
		t.Error("Expected useWorm to be true for cloud storage")
	}

	// Test third storage unit (tape-stu-1) without optional fields
	tapeSTU := storages.Data[2]
	if tapeSTU.Attributes.Name != "tape-stu-1" {
		t.Errorf("Expected name 'tape-stu-1', got '%s'", tapeSTU.Attributes.Name)
	}
	if tapeSTU.Attributes.StorageType != "TAPE" {
		t.Errorf("Expected storageType 'TAPE', got '%s'", tapeSTU.Attributes.StorageType)
	}
	// Verify optional fields are empty/false when not present
	if tapeSTU.Attributes.StorageCategory != "" {
		t.Errorf("Expected empty storageCategory for tape, got '%s'", tapeSTU.Attributes.StorageCategory)
	}
}

func TestStoragesUnmarshalJSONWithoutOptionalFields(t *testing.T) {
	// Test JSON without optional fields (backward compatibility)
	jsonData := `{
		"data": [{
			"type": "storageUnit",
			"id": "test-stu",
			"links": {
				"self": {
					"href": "/netbackup/storage/storage-units/test-stu"
				}
			},
			"attributes": {
				"name": "test-stu",
				"storageType": "DISK",
				"storageSubType": "AdvancedDisk",
				"storageServerType": "MEDIA_SERVER",
				"useAnyAvailableMediaServer": true,
				"accelerator": false,
				"instantAccessEnabled": false,
				"isCloudSTU": false,
				"freeCapacityBytes": 1000000000,
				"totalCapacityBytes": 2000000000,
				"usedCapacityBytes": 1000000000,
				"maxFragmentSizeMegabytes": 2048,
				"maxConcurrentJobs": 4,
				"onDemandOnly": false
			},
			"relationships": {
				"diskPool": {
					"links": {
						"related": {
							"href": "/netbackup/storage/disk-pools/dp-1"
						}
					},
					"data": {
						"type": "diskPool",
						"id": "dp-1"
					}
				}
			}
		}],
		"meta": {
			"pagination": {
				"count": 1,
				"offset": 0,
				"limit": 100,
				"first": 0,
				"last": 0,
				"page": 1,
				"pages": 1
			}
		},
		"links": {
			"self": {
				"href": "/netbackup/storage/storage-units"
			},
			"first": {
				"href": "/netbackup/storage/storage-units"
			},
			"last": {
				"href": "/netbackup/storage/storage-units"
			}
		}
	}`

	var storages Storages
	err := json.Unmarshal([]byte(jsonData), &storages)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(storages.Data) != 1 {
		t.Fatalf("Expected 1 storage unit, got %d", len(storages.Data))
	}

	stu := storages.Data[0]
	if stu.Attributes.Name != "test-stu" {
		t.Errorf("Expected name 'test-stu', got '%s'", stu.Attributes.Name)
	}
	if stu.Attributes.FreeCapacityBytes != 1000000000 {
		t.Errorf("Expected freeCapacityBytes 1000000000, got %d", stu.Attributes.FreeCapacityBytes)
	}

	// Verify optional fields have zero values when not present
	if stu.Attributes.StorageCategory != "" {
		t.Errorf("Expected empty storageCategory, got '%s'", stu.Attributes.StorageCategory)
	}
	if stu.Attributes.ReplicationCapable {
		t.Error("Expected replicationCapable to be false when not present")
	}
	if stu.Attributes.WormCapable {
		t.Error("Expected wormCapable to be false when not present")
	}
}

func TestStoragesPagination(t *testing.T) {
	// Read the test fixture
	data, err := os.ReadFile("../../testdata/api-10.5/storage-units-response.json")
	if err != nil {
		t.Fatalf("Failed to read test fixture: %v", err)
	}

	var storages Storages
	err = json.Unmarshal(data, &storages)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify pagination metadata
	if storages.Meta.Pagination.Count != 3 {
		t.Errorf("Expected pagination count 3, got %d", storages.Meta.Pagination.Count)
	}
	if storages.Meta.Pagination.Offset != 0 {
		t.Errorf("Expected pagination offset 0, got %d", storages.Meta.Pagination.Offset)
	}
	if storages.Meta.Pagination.Limit != 100 {
		t.Errorf("Expected pagination limit 100, got %d", storages.Meta.Pagination.Limit)
	}
	if storages.Meta.Pagination.Pages != 1 {
		t.Errorf("Expected pagination pages 1, got %d", storages.Meta.Pagination.Pages)
	}
	if storages.Meta.Pagination.First != 0 {
		t.Errorf("Expected pagination first 0, got %d", storages.Meta.Pagination.First)
	}
	if storages.Meta.Pagination.Last != 2 {
		t.Errorf("Expected pagination last 2, got %d", storages.Meta.Pagination.Last)
	}

	// Verify links
	if storages.Links.Self.Href == "" {
		t.Error("Expected self link to be present")
	}
	if storages.Links.First.Href == "" {
		t.Error("Expected first link to be present")
	}
	if storages.Links.Last.Href == "" {
		t.Error("Expected last link to be present")
	}
}

func TestStoragesRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
	}{
		{
			name: "all required fields present",
			jsonData: `{
				"data": [{
					"type": "storageUnit",
					"id": "test",
					"attributes": {
						"name": "test",
						"storageType": "DISK",
						"storageServerType": "MEDIA_SERVER",
						"freeCapacityBytes": 1000,
						"usedCapacityBytes": 500
					}
				}],
				"meta": {"pagination": {"count": 1}}
			}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var storages Storages
			err := json.Unmarshal([]byte(tt.jsonData), &storages)
			if (err != nil) != tt.expectError {
				t.Errorf("Unmarshal() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
