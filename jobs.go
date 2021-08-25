package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

func getJob(jobsSize map[string]float64, jobsCount map[string]float64, jobsStatusCount map[string]float64, offset int) int {

	var j Jobs

	endTime := time.Now()
	duration, errMsg := time.ParseDuration("-" + Cfg.Server.ScrappingInterval)
	if errMsg != nil {
		PanicLoggerStr("incorrect interval " + Cfg.Server.ScrappingInterval)
	}
	startTime := endTime.Add(duration)
	url := nbuRoot + "/admin/jobs"

	// Create a Resty Client
	Client := resty.New()
	Client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	Client.SetTimeout(1 * time.Minute)

	var query string = "page[limit]=1&page[offset]=" + strconv.Itoa(offset) + "&sort=jobId&filter=endTime%20gt%20" + convertTimeToNBUDate(startTime.UTC())
	resp, _ := Client.R().SetQueryString(query).
		SetHeader("Accept", Cfg.NbuServer.ContentType).
		SetHeader("Authorization", Cfg.NbuServer.APIKey).
		Get(url)

	unmarshalError := Client.JSONUnmarshal(resp.Body(), &j)

	if unmarshalError != nil {
		fmt.Println(unmarshalError)
		os.Exit(1)
	}
	var pages int = j.Meta.Pagination.Pages
	if pages > 0 {
		//	[]string{"action", "policy_type", "status"}, nil),
		jobType := j.Data[0].Attributes.JobType
		policyType := j.Data[0].Attributes.PolicyType
		jobStatus := j.Data[0].Attributes.Status
		jobSize := j.Data[0].Attributes.KilobytesTransferred * 1024

		key := jobType + "|" + policyType + "|" + strconv.Itoa(jobStatus)
		key2 := jobType + "|" + strconv.Itoa(jobStatus)

		jobsCount[key] = jobsCount[key] + float64(1)
		if jobsStatusCount[key2] == 0 {
			jobsStatusCount[key2] = 1
		} else {
			jobsStatusCount[key2] = jobsStatusCount[key2] + 1
		}

		if jobsSize[key] == 0 {
			jobsSize[key] = float64(jobSize)
		} else {
			jobsSize[key] = jobsSize[key] + float64(jobSize)
		}
	}

	if j.Meta.Pagination.Offset == j.Meta.Pagination.Last {
		return -1
	} else {
		return int(j.Meta.Pagination.Next)
	}

}

func getJobs(jobsSize map[string]float64, jobsCount map[string]float64, jobsStatusCount map[string]float64) {

	var offset int = 0

	for offset != -1 {
		offset = getJob(jobsSize, jobsCount, jobsStatusCount, offset)
	}

}
