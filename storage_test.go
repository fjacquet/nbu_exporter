package main

// func TestStorage_UnmarshalJSON(t *testing.T) {
// 	tests := []struct {
// 		name    string
// 		input   []byte
// 		want    *Storage
// 		wantErr bool
// 	}{
// 		{
// 			name: "valid input",
// 			input: []byte(`{
// 				"data": {
// 					"links": {
// 						"self": {
// 							"href": "https://example.com/api/storage/123"
// 						}
// 					},
// 					"type": "storage",
// 					"id": "123",
// 					"attributes": {
// 						"name": "My Storage",
// 						"storageType": "disk",
// 						"storageSubType": "basic",
// 						"storageServerType": "media",
// 						"useAnyAvailableMediaServer": true,
// 						"accelerator": false,
// 						"instantAccessEnabled": true,
// 						"isCloudSTU": false,
// 						"freeCapacityBytes": 1024,
// 						"totalCapacityBytes": 2048,
// 						"usedCapacityBytes": 1024,
// 						"maxFragmentSizeMegabytes": 512,
// 						"maxConcurrentJobs": 10,
// 						"onDemandOnly": false
// 					},
// 					"relationships": {
// 						"diskPool": {
// 							"links": {
// 								"related": {
// 									"href": "https://example.com/api/diskpools/456"
// 								}
// 							},
// 							"data": {
// 								"type": "diskPool",
// 								"id": "456"
// 							}
// 						}
// 					}
// 				}
// 			}`),
// 			want: &Storage{
// 				Data: struct {
// 					Links struct {
// 						Self struct {
// 							Href string `json:"href"`
// 						} `json:"self"`
// 					} `json:"links"`
// 					Type       string `json:"type"`
// 					ID         string `json:"id"`
// 					Attributes struct {
// 						Name                       string `json:"name"`
// 						StorageType                string `json:"storageType"`
// 						StorageSubType             string `json:"storageSubType"`
// 						StorageServerType          string `json:"storageServerType"`
// 						UseAnyAvailableMediaServer bool   `json:"useAnyAvailableMediaServer"`
// 						Accelerator                bool   `json:"accelerator"`
// 						InstantAccessEnabled       bool   `json:"instantAccessEnabled"`
// 						IsCloudSTU                 bool   `json:"isCloudSTU"`
// 						FreeCapacityBytes          int64  `json:"freeCapacityBytes"`
// 						TotalCapacityBytes         int64  `json:"totalCapacityBytes"`
// 						UsedCapacityBytes          int64  `json:"usedCapacityBytes"`
// 						MaxFragmentSizeMegabytes   int    `json:"maxFragmentSizeMegabytes"`
// 						MaxConcurrentJobs          int    `json:"maxConcurrentJobs"`
// 						OnDemandOnly               bool   `json:"onDemandOnly"`
// 					} `json:"attributes"`
// 					Relationships struct {
// 						DiskPool struct {
// 							Links struct {
// 								Related struct {
// 									Href string `json:"href"`
// 								} `json:"related"`
// 							} `json:"links"`
// 							Data struct {
// 								Type string `json:"type"`
// 								ID   string `json:"id"`
// 							} `json:"data"`
// 						} `json:"diskPool"`
// 					} `json:"relationships"`
// 				}{
// 					Links: struct {
// 						Self struct {
// 							Href string `json:"href"`
// 						} `json:"self"`
// 					}{
// 						Self: struct {
// 							Href string `json:"href"`
// 						}{
// 							Href: "https://example.com/api/storage/123",
// 						},
// 					},
// 					Type: "storage",
// 					ID:   "123",
// 					Attributes: struct {
// 						Name                       string `json:"name"`
// 						StorageType                string `json:"storageType"`
// 						StorageSubType             string `json:"storageSubType"`
// 						StorageServerType          string `json:"storageServerType"`
// 						UseAnyAvailableMediaServer bool   `json:"useAnyAvailableMediaServer"`
// 						Accelerator                bool   `json:"accelerator"`
// 						InstantAccessEnabled       bool   `json:"instantAccessEnabled"`
// 						IsCloudSTU                 bool   `json:"isCloudSTU"`
// 						FreeCapacityBytes          int64  `json:"freeCapacityBytes"`
// 						TotalCapacityBytes         int64  `json:"totalCapacityBytes"`
// 						UsedCapacityBytes          int64  `json:"usedCapacityBytes"`
// 						MaxFragmentSizeMegabytes   int    `json:"maxFragmentSizeMegabytes"`
// 						MaxConcurrentJobs          int    `json:"maxConcurrentJobs"`
// 						OnDemandOnly               bool   `json:"onDemandOnly"`
// 					}{
// 						Name:                       "My Storage",
// 						StorageType:                "disk",
// 						StorageSubType:             "basic",
// 						StorageServerType:          "media",
// 						UseAnyAvailableMediaServer: true,
// 						Accelerator:                false,
// 						InstantAccessEnabled:       true,
// 						IsCloudSTU:                 false,
// 						FreeCapacityBytes:          1024,
// 						TotalCapacityBytes:         2048,
// 						UsedCapacityBytes:          1024,
// 						MaxFragmentSizeMegabytes:   512,
// 						MaxConcurrentJobs:          10,
// 						OnDemandOnly:               false,
// 					},
// 					Relationships: struct {
// 						DiskPool struct {
// 							Links struct {
// 								Related struct {
// 									Href string `json:"href"`
// 								} `json:"related"`
// 							} `json:"links"`
// 							Data struct {
// 								Type string `json:"type"`
// 								ID   string `json:"id"`
// 							} `json:"data"`
// 						} `json:"diskPool"`
// 					}{
// 						DiskPool: struct {
// 							Links struct {
// 								Related struct {
// 									Href string `json:"href"`
// 								} `json:"related"`
// 							} `json:"links"`
// 							Data struct {
// 								Type string `json:"type"`
// 								ID   string `json:"id"`
// 							} `json:"data"`
// 						}{
// 							Links: struct {
// 								Related struct {
// 									Href string `json:"href"`
// 								} `json:"related"`
// 							}{
// 								Related: struct {
// 									Href string `json:"href"`
// 								}{
// 									Href: "https://example.com/api/diskpools/456",
// 								},
// 							},
// 							Data: struct {
// 								Type string `json:"type"`
// 								ID   string `json:"id"`
// 							}{
// 								Type: "diskPool",
// 								ID:   "456",
// 							},
// 						},
// 					},
// 				},
// 			},
// 			wantErr: false,
// 		},
// 		{
// 			name:    "invalid input",
// 			input:   []byte(`invalid`),
// 			want:    nil,
// 			wantErr: true,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			var s Storage
// 			err := s.UnmarshalJSON(tt.input)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("Storage.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(&s, tt.want) {
// 				t.Errorf("Storage.UnmarshalJSON() = %v, want %v", s, tt.want)
// 			}
// 		})
// 	}
// }
