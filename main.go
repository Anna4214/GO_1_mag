package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	serverURL    = "http://srv.msk01.gigacorp.local/_stats"
	pollInterval = 10 * time.Second
	maxErrors    = 3

	loadAvgThreshold = 30.0
	memoryThreshold  = 80.0
	diskThreshold    = 90.0
	networkThreshold = 90.0
)

type Stats struct {
	LoadAverage      float64
	TotalMemory      int64
	UsedMemory       int64
	TotalDisk        int64
	UsedDisk         int64
	NetworkBandwidth int64
	NetworkUsage     int64
}

func main() {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	errorCount := 0

	// Первый запрос сразу при запуске
	checkServer(&errorCount)

	// Периодический опрос
	for range ticker.C {
		checkServer(&errorCount)
	}
}

func checkServer(errorCount *int) {
	stats, err := fetchStats()
	if err != nil {
		*errorCount++
		if *errorCount >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return
	}

	*errorCount = 0
	checkThresholds(stats)
}

func fetchStats() (*Stats, error) {
	resp, err := http.Get(serverURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseStats(string(body))
}

func parseStats(data string) (*Stats, error) {
	data = strings.TrimSpace(data)
	fields := strings.Split(data, ",")

	if len(fields) != 7 {
		return nil, fmt.Errorf("invalid data format")
	}

	stats := &Stats{}
	var err error

	stats.LoadAverage, err = strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return nil, err
	}

	stats.TotalMemory, err = strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return nil, err
	}

	stats.UsedMemory, err = strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return nil, err
	}

	stats.TotalDisk, err = strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return nil, err
	}

	stats.UsedDisk, err = strconv.ParseInt(fields[4], 10, 64)
	if err != nil {
		return nil, err
	}

	stats.NetworkBandwidth, err = strconv.ParseInt(fields[5], 10, 64)
	if err != nil {
		return nil, err
	}

	stats.NetworkUsage, err = strconv.ParseInt(fields[6], 10, 64)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func checkThresholds(stats *Stats) {
	// Load Average
	if stats.LoadAverage > loadAvgThreshold {
		fmt.Printf("Load Average is too high: %.0f\n", stats.LoadAverage)
	}

	// Memory
	if stats.TotalMemory > 0 {
		memoryPercent := float64(stats.UsedMemory) / float64(stats.TotalMemory) * 100
		if memoryPercent > memoryThreshold {
			fmt.Printf("Memory usage too high: %.0f%%\n", memoryPercent)
		}
	}

	// Disk
	if stats.TotalDisk > 0 {
		diskPercent := float64(stats.UsedDisk) / float64(stats.TotalDisk) * 100
		if diskPercent > diskThreshold {
			freeDiskBytes := stats.TotalDisk - stats.UsedDisk
			freeDiskMB := freeDiskBytes / (1024 * 1024)
			fmt.Printf("Free disk space is too low: %d Mb left\n", freeDiskMB)
		}
	}

	// Network
	if stats.NetworkBandwidth > 0 {
		networkPercent := float64(stats.NetworkUsage) / float64(stats.NetworkBandwidth) * 100
		if networkPercent > networkThreshold {
			availableBandwidthBytes := stats.NetworkBandwidth - stats.NetworkUsage
			availableMbits := float64(availableBandwidthBytes) * 8 / (1024 * 1024)
			fmt.Printf("Network bandwidth usage high: %.2f Mbit/s available\n", availableMbits)
		}
	}
}
