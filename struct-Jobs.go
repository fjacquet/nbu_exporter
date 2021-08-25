package main

import "time"

type Jobs struct {
	Data []struct {
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
			FileLists struct {
				Href string `json:"href"`
			} `json:"file-lists"`
			TryLogs struct {
				Href string `json:"href"`
			} `json:"try-logs"`
		} `json:"links"`
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			JobID                      int       `json:"jobId"`
			ParentJobID                int       `json:"parentJobId"`
			ActiveProcessID            int       `json:"activeProcessId"`
			JobType                    string    `json:"jobType"`
			JobSubType                 string    `json:"jobSubType"`
			PolicyType                 string    `json:"policyType"`
			PolicyName                 string    `json:"policyName"`
			ScheduleType               string    `json:"scheduleType"`
			ScheduleName               string    `json:"scheduleName"`
			ClientName                 string    `json:"clientName"`
			ControlHost                string    `json:"controlHost"`
			JobOwner                   string    `json:"jobOwner"`
			JobGroup                   string    `json:"jobGroup"`
			BackupID                   string    `json:"backupId"`
			SourceMediaID              string    `json:"sourceMediaId"`
			SourceStorageUnitName      string    `json:"sourceStorageUnitName"`
			SourceMediaServerName      string    `json:"sourceMediaServerName"`
			DestinationMediaID         string    `json:"destinationMediaId"`
			DestinationStorageUnitName string    `json:"destinationStorageUnitName"`
			DestinationMediaServerName string    `json:"destinationMediaServerName"`
			DataMovement               string    `json:"dataMovement"`
			StreamNumber               int       `json:"streamNumber"`
			CopyNumber                 int       `json:"copyNumber"`
			Priority                   int       `json:"priority"`
			Compression                int       `json:"compression"`
			Status                     int       `json:"status"`
			State                      string    `json:"state"`
			NumberOfFiles              int       `json:"numberOfFiles"`
			EstimatedFiles             int       `json:"estimatedFiles"`
			KilobytesTransferred       int       `json:"kilobytesTransferred"`
			KilobytesToTransfer        int       `json:"kilobytesToTransfer"`
			TransferRate               int       `json:"transferRate"`
			PercentComplete            int       `json:"percentComplete"`
			Restartable                int       `json:"restartable"`
			Suspendable                int       `json:"suspendable"`
			Resumable                  int       `json:"resumable"`
			FrozenImage                int       `json:"frozenImage"`
			TransportType              string    `json:"transportType"`
			DedupRatio                 float64   `json:"dedupRatio"`
			CurrentOperation           int       `json:"currentOperation"`
			RobotName                  string    `json:"robotName"`
			VaultName                  string    `json:"vaultName"`
			ProfileName                string    `json:"profileName"`
			SessionID                  int       `json:"sessionId"`
			NumberOfTapeToEject        int       `json:"numberOfTapeToEject"`
			SubmissionType             int       `json:"submissionType"`
			AcceleratorOptimization    int       `json:"acceleratorOptimization"`
			DumpHost                   string    `json:"dumpHost"`
			InstanceDatabaseName       string    `json:"instanceDatabaseName"`
			AuditUserName              string    `json:"auditUserName"`
			AuditDomainName            string    `json:"auditDomainName"`
			AuditDomainType            int       `json:"auditDomainType"`
			RestoreBackupIDs           string    `json:"restoreBackupIDs"`
			StartTime                  time.Time `json:"startTime"`
			EndTime                    time.Time `json:"endTime"`
			ActiveTryStartTime         time.Time `json:"activeTryStartTime"`
			LastUpdateTime             time.Time `json:"lastUpdateTime"`
			InitiatorID                string    `json:"initiatorId"`
			RetentionLevel             int       `json:"retentionLevel"`
			Try                        int       `json:"try"`
			Cancellable                int       `json:"cancellable"`
			JobQueueReason             int       `json:"jobQueueReason"`
			JobQueueResource           string    `json:"jobQueueResource"`
			KilobytesDataTransferred   int       `json:"kilobytesDataTransferred"`
			ElapsedTime                string    `json:"elapsedTime"`
			OffHostType                string    `json:"offHostType"`
		} `json:"attributes"`
	} `json:"data"`
	Meta struct {
		Pagination struct {
			Next   int `json:"next"`
			Pages  int `json:"pages"`
			Offset int `json:"offset"`
			Last   int `json:"last"`
			Limit  int `json:"limit"`
			Count  int `json:"count"`
			Page   int `json:"page"`
			First  int `json:"first"`
		} `json:"pagination"`
	} `json:"meta"`
	Links struct {
		Next struct {
			Href string `json:"href"`
		} `json:"next"`
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		Last struct {
			Href string `json:"href"`
		} `json:"last"`
		First struct {
			Href string `json:"href"`
		} `json:"first"`
	} `json:"links"`
}
