package main

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

func ExportResp(resp *resty.Response, err error) {
	// Explore response object
	if cli.Debug {
		fmt.Println("  Error      :", err)
		fmt.Println("  Status Code:", resp.StatusCode())
		fmt.Println("  Status     :", resp.Status())
		fmt.Println("  Proto      :", resp.Proto())
		fmt.Println("  Time       :", resp.Time())
		fmt.Println("  Received At:", resp.ReceivedAt())
	}
	fmt.Println("  Body       :\n", resp)
	// fmt.Println()

	if cli.Debug {
		// Explore trace info
		fmt.Println("Request Trace Info:")
		ti := resp.Request.TraceInfo()
		fmt.Println("  DNSLookup     :", ti.DNSLookup)
		fmt.Println("  ConnTime      :", ti.ConnTime)
		fmt.Println("  TCPConnTime   :", ti.TCPConnTime)
		fmt.Println("  TLSHandshake  :", ti.TLSHandshake)
		fmt.Println("  ServerTime    :", ti.ServerTime)
		fmt.Println("  ResponseTime  :", ti.ResponseTime)
		fmt.Println("  TotalTime     :", ti.TotalTime)
		fmt.Println("  IsConnReused  :", ti.IsConnReused)
		fmt.Println("  IsConnWasIdle :", ti.IsConnWasIdle)
		fmt.Println("  ConnIdleTime  :", ti.ConnIdleTime)
		fmt.Println("  RequestAttempt:", ti.RequestAttempt)
		// fmt.Println("  RemoteAddr    :", ti.RemoteAddr.String())
	}
}
