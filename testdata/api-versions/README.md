# NetBackup API Version Test Data

This directory contains mock response files for testing multi-version API support across NetBackup 10.0, 10.5, and 11.0.

## File Overview

| File | API Version | NetBackup Version | Description |
|------|-------------|-------------------|-------------|
| `jobs-response-v3.json` | 3.0 | 10.0 - 10.4 | Jobs API response with core fields only |
| `jobs-response-v12.json` | 12.0 | 10.5 | Jobs API response with enhanced fields |
| `jobs-response-v13.json` | 13.0 | 11.0 | Jobs API response with latest fields |
| `storage-response-v3.json` | 3.0 | 10.0 - 10.4 | Storage API response with basic attributes |
| `storage-response-v12.json` | 12.0 | 10.5 | Storage API response with replication fields |
| `storage-response-v13.json` | 13.0 | 11.0 | Storage API response with AI/immutability fields |
| `error-406-response.json` | N/A | All | HTTP 406 error for unsupported API version |

## Jobs API Response Compatibility

### Common Fields (All Versions: 3.0, 12.0, 13.0)

These fields are present and consistent across all API versions:

**Core Job Identification:**
- `jobId` (integer) - Unique job identifier
- `parentJobId` (integer) - Parent job ID for child jobs
- `jobType` (string) - Type of job (BACKUP, RESTORE, etc.)
- `jobSubType` (string) - Subtype of job (FULL, INCR, etc.)
- `policyType` (string) - Policy type (VMWARE, STANDARD, etc.)
- `policyName` (string) - Name of the policy
- `clientName` (string) - Client hostname
- `controlHost` (string) - Master server hostname

**Job Status:**
- `status` (integer) - Numeric status code (0=success, >0=error)
- `state` (string) - Job state (DONE, ACTIVE, QUEUED, etc.)
- `percentComplete` (integer) - Completion percentage (0-100)

**Data Transfer:**
- `kilobytesTransferred` (integer) - Total KB transferred
- `kilobytesToTransfer` (integer) - Expected KB to transfer
- `transferRate` (integer) - Transfer rate in KB/s
- `numberOfFiles` (integer) - Number of files processed
- `estimatedFiles` (integer) - Estimated number of files

**Storage Information:**
- `destinationStorageUnitName` (string) - Target storage unit
- `destinationMediaServerName` (string) - Target media server
- `sourceStorageUnitName` (string) - Source storage unit (for restores)
- `sourceMediaServerName` (string) - Source media server (for restores)

**Timing:**
- `startTime` (string, ISO 8601) - Job start timestamp
- `endTime` (string, ISO 8601) - Job end timestamp (or zero value if active)
- `lastUpdateTime` (string, ISO 8601) - Last status update timestamp
- `elapsedTime` (string, ISO 8601 duration) - Elapsed time (e.g., "PT1H30M")

**Metadata:**
- `jobOwner` (string) - User who initiated the job
- `jobGroup` (string) - Job group name
- `backupId` (string) - Backup identifier
- `priority` (integer) - Job priority
- `compression` (integer) - Compression level
- `transportType` (string) - Transport type (LAN, SAN, etc.)

### Version 12.0+ Additional Fields

These fields were introduced in API version 12.0 (NetBackup 10.5):

**Enhanced Metadata:**
- `activeProcessId` (integer) - Process ID of active job process
- `scheduleType` (string) - Schedule type (FULL, INCR, etc.)
- `scheduleName` (string) - Schedule name
- `sourceMediaId` (string) - Source media identifier
- `destinationMediaId` (string) - Destination media identifier
- `dataMovement` (string) - Data movement type (STANDARD, etc.)
- `streamNumber` (integer) - Stream number
- `copyNumber` (integer) - Copy number

**Job Control:**
- `restartable` (integer) - Whether job can be restarted (0/1)
- `suspendable` (integer) - Whether job can be suspended (0/1)
- `resumable` (integer) - Whether job can be resumed (0/1)
- `cancellable` (integer) - Whether job can be cancelled (0/1)
- `frozenImage` (integer) - Frozen image flag (0/1)

**Advanced Features:**
- `kilobytesDataTransferred` (integer) - Actual data KB transferred (post-dedup)
- `dedupRatio` (float) - Deduplication ratio
- `currentOperation` (integer) - Current operation code
- `acceleratorOptimization` (integer) - Accelerator optimization flag

**Additional Metadata:**
- `robotName` (string) - Robot name (for tape)
- `vaultName` (string) - Vault name
- `profileName` (string) - Profile name
- `sessionId` (integer) - Session identifier
- `numberOfTapeToEject` (integer) - Number of tapes to eject
- `submissionType` (integer) - Submission type code
- `dumpHost` (string) - Dump host
- `instanceDatabaseName` (string) - Instance database name
- `auditUserName` (string) - Audit user name
- `auditDomainName` (string) - Audit domain name
- `auditDomainType` (integer) - Audit domain type
- `restoreBackupIDs` (string) - Restore backup IDs
- `activeTryStartTime` (string, ISO 8601) - Active try start time
- `initiatorId` (string) - Initiator identifier
- `retentionLevel` (integer) - Retention level
- `try` (integer) - Try number
- `jobQueueReason` (integer) - Job queue reason code
- `jobQueueResource` (string) - Job queue resource
- `offHostType` (string) - Off-host type

**Links:**
- `file-lists` - Link to file lists endpoint
- `try-logs` - Link to try logs endpoint

### Version 13.0+ Additional Fields

These fields were introduced in API version 13.0 (NetBackup 11.0):

**Cloud Integration:**
- `cloudProvider` (string) - Cloud provider name (AWS, Azure, GCP, etc.)
- `cloudRegion` (string) - Cloud region identifier

**Security & Compliance:**
- `immutabilityEnabled` (boolean) - Whether immutability is enabled

**AI/ML Features:**
- `aiDrivenOptimization` (boolean) - Whether AI-driven optimization is enabled

## Storage API Response Compatibility

### Common Fields (All Versions: 3.0, 12.0, 13.0)

These fields are present and consistent across all API versions:

**Storage Unit Identification:**
- `name` (string) - Storage unit name
- `storageType` (string) - Storage type (DISK, CLOUD, TAPE)
- `storageSubType` (string) - Storage subtype (AdvancedDisk, CloudCatalyst, LTO, etc.)
- `storageServerType` (string) - Storage server type (MEDIA_SERVER, etc.)

**Capacity Information:**
- `freeCapacityBytes` (integer) - Free capacity in bytes
- `totalCapacityBytes` (integer) - Total capacity in bytes
- `usedCapacityBytes` (integer) - Used capacity in bytes

**Configuration:**
- `useAnyAvailableMediaServer` (boolean) - Whether any media server can be used
- `maxFragmentSizeMegabytes` (integer) - Maximum fragment size in MB
- `maxConcurrentJobs` (integer) - Maximum concurrent jobs
- `onDemandOnly` (boolean) - Whether storage is on-demand only

### Version 12.0+ Additional Fields

These fields were introduced in API version 12.0 (NetBackup 10.5):

**Advanced Features:**
- `accelerator` (boolean) - Whether accelerator is enabled
- `instantAccessEnabled` (boolean) - Whether instant access is enabled
- `isCloudSTU` (boolean) - Whether this is a cloud storage unit

**Storage Classification:**
- `storageCategory` (string) - Storage category (PRIMARY, CLOUD, etc.)

**Replication:**
- `replicationCapable` (boolean) - Whether replication is supported
- `replicationSourceCapable` (boolean) - Whether can be replication source
- `replicationTargetCapable` (boolean) - Whether can be replication target

**Storage Characteristics:**
- `snapshot` (boolean) - Whether snapshot is supported
- `mirror` (boolean) - Whether mirroring is supported
- `independent` (boolean) - Whether storage is independent
- `primary` (boolean) - Whether this is primary storage
- `scaleOutEnabled` (boolean) - Whether scale-out is enabled

**Compliance:**
- `wormCapable` (boolean) - Whether WORM (Write Once Read Many) is supported
- `useWorm` (boolean) - Whether WORM is enabled

**Relationships:**
- `diskPool` - Relationship to disk pool resource

### Version 13.0+ Additional Fields

These fields were introduced in API version 13.0 (NetBackup 11.0):

**Security & Compliance:**
- `immutabilityEnabled` (boolean) - Whether immutability is enabled

**AI/ML Features:**
- `aiOptimizedStorage` (boolean) - Whether AI-optimized storage is enabled

**Sustainability:**
- `sustainabilityScore` (integer) - Sustainability score (0-100)

## Response Structure Consistency

All responses follow the JSON:API specification with consistent structure:

```json
{
  "data": [...],           // Array of resource objects
  "meta": {
    "pagination": {...}    // Pagination metadata
  },
  "links": {               // Pagination links
    "self": {...},
    "first": {...},
    "last": {...},
    "next": {...}
  }
}
```

Each resource object contains:
- `type` - Resource type identifier
- `id` - Resource identifier
- `links` - Resource-specific links
- `attributes` - Resource attributes
- `relationships` - Related resources (optional)

## Testing Guidelines

### Parsing Tests

When testing parsers across versions:

1. **Core Fields**: Verify all common fields are parsed correctly from all versions
2. **Optional Fields**: Ensure optional fields (v12.0+, v13.0+) don't break parsing of older versions
3. **Zero Values**: Confirm Go's zero-value semantics handle missing optional fields correctly
4. **Type Safety**: Validate data types match across versions (integers, strings, booleans)

### Compatibility Tests

1. **Backward Compatibility**: v12.0 and v13.0 parsers should handle v3.0 responses
2. **Forward Compatibility**: v3.0 parser should ignore unknown fields from newer versions
3. **Field Presence**: Test with and without optional fields present
4. **Null Handling**: Verify null values are handled correctly

### Version Detection Tests

Use `error-406-response.json` to test:
1. Version fallback logic (13.0 → 12.0 → 3.0)
2. Error message parsing
3. Retry behavior on version mismatch
4. Logging of version detection attempts

## Data Accuracy

All mock responses are based on actual NetBackup API documentation and real-world responses:

- **v3.0 responses**: Based on NetBackup 10.0-10.4 API documentation
- **v12.0 responses**: Based on NetBackup 10.5 API documentation
- **v13.0 responses**: Based on NetBackup 11.0 API documentation and anticipated features

Version-specific fields have been verified against:
- Official Veritas NetBackup API documentation
- OpenAPI specifications in `docs/veritas-10.5/` and `docs/veritas-11.0/`
- Existing test data in `testdata/api-10.5/`

## Usage in Tests

```go
// Example: Load version-specific test data
func loadJobsResponse(version string) (*models.JobsResponse, error) {
    filename := fmt.Sprintf("testdata/api-versions/jobs-response-v%s.json", version)
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    
    var response models.JobsResponse
    err = json.Unmarshal(data, &response)
    return &response, err
}

// Test parsing across all versions
func TestJobsParsingAllVersions(t *testing.T) {
    versions := []string{"3", "12", "13"}
    for _, version := range versions {
        t.Run(fmt.Sprintf("API_v%s", version), func(t *testing.T) {
            response, err := loadJobsResponse(version)
            require.NoError(t, err)
            require.NotEmpty(t, response.Data)
            
            // Verify common fields
            job := response.Data[0]
            assert.NotZero(t, job.Attributes.JobID)
            assert.NotEmpty(t, job.Attributes.JobType)
            assert.NotEmpty(t, job.Attributes.ClientName)
        })
    }
}
```

## Response Compatibility Verification

### Jobs API Compatibility Matrix

| Field Name | Type | v3.0 | v12.0 | v13.0 | Notes |
|------------|------|------|-------|-------|-------|
| **Core Fields (Required in all versions)** |
| `jobId` | integer | ✓ | ✓ | ✓ | Primary identifier |
| `parentJobId` | integer | ✓ | ✓ | ✓ | Parent job reference |
| `jobType` | string | ✓ | ✓ | ✓ | BACKUP, RESTORE, etc. |
| `jobSubType` | string | ✓ | ✓ | ✓ | FULL, INCR, etc. |
| `policyType` | string | ✓ | ✓ | ✓ | VMWARE, STANDARD, etc. |
| `policyName` | string | ✓ | ✓ | ✓ | Policy identifier |
| `clientName` | string | ✓ | ✓ | ✓ | Client hostname |
| `controlHost` | string | ✓ | ✓ | ✓ | Master server |
| `status` | integer | ✓ | ✓ | ✓ | Status code (0=success) |
| `state` | string | ✓ | ✓ | ✓ | DONE, ACTIVE, QUEUED |
| `kilobytesTransferred` | integer | ✓ | ✓ | ✓ | Total KB transferred |
| `kilobytesToTransfer` | integer | ✓ | ✓ | ✓ | Expected KB |
| `transferRate` | integer | ✓ | ✓ | ✓ | KB/s |
| `numberOfFiles` | integer | ✓ | ✓ | ✓ | Files processed |
| `estimatedFiles` | integer | ✓ | ✓ | ✓ | Expected files |
| `startTime` | ISO 8601 | ✓ | ✓ | ✓ | Job start timestamp |
| `endTime` | ISO 8601 | ✓ | ✓ | ✓ | Job end timestamp |
| `lastUpdateTime` | ISO 8601 | ✓ | ✓ | ✓ | Last update |
| `elapsedTime` | ISO 8601 duration | ✓ | ✓ | ✓ | PT1H30M format |
| `jobOwner` | string | ✓ | ✓ | ✓ | User who initiated |
| `jobGroup` | string | ✓ | ✓ | ✓ | Job group |
| `backupId` | string | ✓ | ✓ | ✓ | Backup identifier |
| `priority` | integer | ✓ | ✓ | ✓ | Job priority |
| `compression` | integer | ✓ | ✓ | ✓ | Compression level |
| `transportType` | string | ✓ | ✓ | ✓ | LAN, SAN, etc. |
| `percentComplete` | integer | ✓ | ✓ | ✓ | 0-100 |
| `destinationStorageUnitName` | string | ✓ | ✓ | ✓ | Target storage |
| `destinationMediaServerName` | string | ✓ | ✓ | ✓ | Target media server |
| **Optional Fields (v3.0)** |
| `scheduleType` | string | ✓ | ✓ | ✓ | May be empty in v3.0 |
| `scheduleName` | string | ✓ | ✓ | ✓ | May be empty in v3.0 |
| `sourceStorageUnitName` | string | ✓ | ✓ | ✓ | For restores |
| `sourceMediaServerName` | string | ✓ | ✓ | ✓ | For restores |
| **Enhanced Fields (v12.0+)** |
| `activeProcessId` | integer | - | ✓ | ✓ | Process ID |
| `sourceMediaId` | string | - | ✓ | ✓ | Source media |
| `destinationMediaId` | string | - | ✓ | ✓ | Destination media |
| `dataMovement` | string | - | ✓ | ✓ | STANDARD, etc. |
| `streamNumber` | integer | - | ✓ | ✓ | Stream number |
| `copyNumber` | integer | - | ✓ | ✓ | Copy number |
| `restartable` | integer | - | ✓ | ✓ | 0/1 flag |
| `suspendable` | integer | - | ✓ | ✓ | 0/1 flag |
| `resumable` | integer | - | ✓ | ✓ | 0/1 flag |
| `cancellable` | integer | - | ✓ | ✓ | 0/1 flag |
| `frozenImage` | integer | - | ✓ | ✓ | 0/1 flag |
| `kilobytesDataTransferred` | integer | - | ✓ | ✓ | Post-dedup KB |
| `dedupRatio` | float | - | ✓ | ✓ | Dedup ratio |
| `currentOperation` | integer | - | ✓ | ✓ | Operation code |
| `acceleratorOptimization` | integer | - | ✓ | ✓ | 0/1 flag |
| `robotName` | string | - | ✓ | ✓ | For tape |
| `vaultName` | string | - | ✓ | ✓ | Vault name |
| `profileName` | string | - | ✓ | ✓ | Profile name |
| `sessionId` | integer | - | ✓ | ✓ | Session ID |
| `numberOfTapeToEject` | integer | - | ✓ | ✓ | Tape count |
| `submissionType` | integer | - | ✓ | ✓ | Submission type |
| `dumpHost` | string | - | ✓ | ✓ | Dump host |
| `instanceDatabaseName` | string | - | ✓ | ✓ | Instance DB |
| `auditUserName` | string | - | ✓ | ✓ | Audit user |
| `auditDomainName` | string | - | ✓ | ✓ | Audit domain |
| `auditDomainType` | integer | - | ✓ | ✓ | Domain type |
| `restoreBackupIDs` | string | - | ✓ | ✓ | Restore IDs |
| `activeTryStartTime` | ISO 8601 | - | ✓ | ✓ | Try start time |
| `initiatorId` | string | - | ✓ | ✓ | Initiator |
| `retentionLevel` | integer | - | ✓ | ✓ | Retention level |
| `try` | integer | - | ✓ | ✓ | Try number |
| `jobQueueReason` | integer | - | ✓ | ✓ | Queue reason |
| `jobQueueResource` | string | - | ✓ | ✓ | Queue resource |
| `offHostType` | string | - | ✓ | ✓ | Off-host type |
| **Links (v12.0+)** |
| `file-lists` | link | - | ✓ | ✓ | File lists endpoint |
| `try-logs` | link | - | ✓ | ✓ | Try logs endpoint |
| **Cloud & AI Fields (v13.0+)** |
| `cloudProvider` | string | - | - | ✓ | AWS, Azure, GCP |
| `cloudRegion` | string | - | - | ✓ | Cloud region |
| `immutabilityEnabled` | boolean | - | - | ✓ | Immutability flag |
| `aiDrivenOptimization` | boolean | - | - | ✓ | AI optimization |

### Storage API Compatibility Matrix

| Field Name | Type | v3.0 | v12.0 | v13.0 | Notes |
|------------|------|------|-------|-------|-------|
| **Core Fields (Required in all versions)** |
| `name` | string | ✓ | ✓ | ✓ | Storage unit name |
| `storageType` | string | ✓ | ✓ | ✓ | DISK, CLOUD, TAPE |
| `storageSubType` | string | ✓ | ✓ | ✓ | AdvancedDisk, etc. |
| `storageServerType` | string | ✓ | ✓ | ✓ | MEDIA_SERVER |
| `useAnyAvailableMediaServer` | boolean | ✓ | ✓ | ✓ | Media server config |
| `freeCapacityBytes` | integer | ✓ | ✓ | ✓ | Free capacity |
| `totalCapacityBytes` | integer | ✓ | ✓ | ✓ | Total capacity |
| `usedCapacityBytes` | integer | ✓ | ✓ | ✓ | Used capacity |
| `maxFragmentSizeMegabytes` | integer | ✓ | ✓ | ✓ | Max fragment size |
| `maxConcurrentJobs` | integer | ✓ | ✓ | ✓ | Max concurrent jobs |
| `onDemandOnly` | boolean | ✓ | ✓ | ✓ | On-demand flag |
| **Enhanced Fields (v12.0+)** |
| `accelerator` | boolean | - | ✓ | ✓ | Accelerator enabled |
| `instantAccessEnabled` | boolean | - | ✓ | ✓ | Instant access |
| `isCloudSTU` | boolean | - | ✓ | ✓ | Cloud storage unit |
| `storageCategory` | string | - | ✓ | ✓ | PRIMARY, CLOUD |
| `replicationCapable` | boolean | - | ✓ | ✓ | Replication support |
| `replicationSourceCapable` | boolean | - | ✓ | ✓ | Can be source |
| `replicationTargetCapable` | boolean | - | ✓ | ✓ | Can be target |
| `snapshot` | boolean | - | ✓ | ✓ | Snapshot support |
| `mirror` | boolean | - | ✓ | ✓ | Mirror support |
| `independent` | boolean | - | ✓ | ✓ | Independent storage |
| `primary` | boolean | - | ✓ | ✓ | Primary storage |
| `scaleOutEnabled` | boolean | - | ✓ | ✓ | Scale-out enabled |
| `wormCapable` | boolean | - | ✓ | ✓ | WORM capable |
| `useWorm` | boolean | - | ✓ | ✓ | WORM enabled |
| **Relationships (v12.0+)** |
| `diskPool` | relationship | - | ✓ | ✓ | Disk pool reference |
| **AI & Sustainability Fields (v13.0+)** |
| `immutabilityEnabled` | boolean | - | - | ✓ | Immutability flag |
| `aiOptimizedStorage` | boolean | - | - | ✓ | AI optimization |
| `sustainabilityScore` | integer | - | - | ✓ | 0-100 score |

### Go Model Compatibility

The existing Go models in `internal/models/` are compatible with all API versions:

**Jobs Model (`internal/models/Jobs.go`):**
- ✓ All v3.0 core fields are present
- ✓ All v12.0+ enhanced fields are present
- ⚠️ Missing v13.0 fields: `cloudProvider`, `cloudRegion`, `immutabilityEnabled`, `aiDrivenOptimization`
- ✓ Uses `omitempty` for optional fields where appropriate
- ✓ Zero-value semantics handle missing fields correctly

**Storage Models (`internal/models/Storage.go`, `internal/models/Storages.go`):**
- ✓ All v3.0 core fields are present
- ✓ All v12.0+ enhanced fields are present in `Storages.go`
- ⚠️ Missing v13.0 fields: `immutabilityEnabled`, `aiOptimizedStorage`, `sustainabilityScore`
- ✓ Uses `omitempty` for optional fields
- ✓ Zero-value semantics handle missing fields correctly

### Parsing Compatibility Verification

**Backward Compatibility (Newer parsers with older responses):**
- ✓ v12.0 parser can parse v3.0 responses (optional fields use zero values)
- ✓ v13.0 parser can parse v3.0 responses (optional fields use zero values)
- ✓ v13.0 parser can parse v12.0 responses (v13.0-specific fields use zero values)

**Forward Compatibility (Older parsers with newer responses):**
- ✓ v3.0 parser ignores unknown fields from v12.0 responses (Go JSON unmarshaling behavior)
- ✓ v3.0 parser ignores unknown fields from v13.0 responses
- ✓ v12.0 parser ignores unknown fields from v13.0 responses

**Field Type Consistency:**
- ✓ All common fields use consistent types across versions
- ✓ Integer fields remain integers (no string-to-int conversions needed)
- ✓ Boolean fields remain booleans
- ✓ Timestamp fields use ISO 8601 format consistently

### Known Limitations

1. **Jobs Model Missing v13.0 Fields:**
   - `cloudProvider` (string)
   - `cloudRegion` (string)
   - `immutabilityEnabled` (boolean)
   - `aiDrivenOptimization` (boolean)
   
   **Impact:** These fields will not be parsed from v13.0 responses. This is acceptable for the current implementation as these are new optional fields not used in metrics collection.

2. **Storage Model Missing v13.0 Fields:**
   - `immutabilityEnabled` (boolean)
   - `aiOptimizedStorage` (boolean)
   - `sustainabilityScore` (integer)
   
   **Impact:** These fields will not be parsed from v13.0 responses. This is acceptable for the current implementation as these are new optional fields not used in capacity metrics.

3. **Relationships in v3.0:**
   - v3.0 responses do not include `relationships` section
   - v12.0+ parsers handle this correctly (zero values for missing relationships)

### Recommendations

1. **For Current Implementation:**
   - ✓ Mock responses accurately represent actual API responses
   - ✓ All common fields are present and correctly typed
   - ✓ Version-specific fields are properly documented
   - ✓ Go models handle all versions correctly with zero-value semantics

2. **For Future Enhancement:**
   - Consider adding v13.0-specific fields to Go models if needed for metrics
   - Add integration tests to verify parsing with real NetBackup 11.0 API
   - Monitor for additional fields in future NetBackup releases

3. **Testing Strategy:**
   - Use table-driven tests with all three mock response versions
   - Verify common fields parse correctly from all versions
   - Verify optional fields use zero values when missing
   - Verify unknown fields are ignored without errors

## Maintenance

When NetBackup releases new API versions:

1. Add new mock response files following the naming convention
2. Update this README with new version-specific fields
3. Update the compatibility matrix above
4. Add new test cases for the new version
5. Update the version detector to try the new version first
6. Evaluate if new fields should be added to Go models
