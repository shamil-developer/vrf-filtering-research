package traffic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/shamil-developer/vrf-filtering-research/internal/command"
)

type IperfRunner struct {
	runner *command.Runner
}

type ProgressFunc func(doneSeconds int, totalSeconds int)

type IperfResult struct {
	BitsPerSecondSent     float64
	BitsPerSecondReceived float64
	BytesSent             float64
	BytesReceived         float64
	Retransmits           int
	DurationSeconds       float64
	CPUHostTotal          float64
	CPURemoteTotal        float64
}

type iperfJSON struct {
	End struct {
		SumSent struct {
			Bytes         float64 `json:"bytes"`
			BitsPerSecond float64 `json:"bits_per_second"`
			Retransmits   int     `json:"retransmits"`
			Seconds       float64 `json:"seconds"`
		} `json:"sum_sent"`
		SumReceived struct {
			Bytes         float64 `json:"bytes"`
			BitsPerSecond float64 `json:"bits_per_second"`
			Seconds       float64 `json:"seconds"`
		} `json:"sum_received"`
		CPUUtilizationPercent struct {
			HostTotal   float64 `json:"host_total"`
			RemoteTotal float64 `json:"remote_total"`
		} `json:"cpu_utilization_percent"`
	} `json:"end"`
}

func NewIperfRunner(
	runner *command.Runner,
) *IperfRunner {
	return &IperfRunner{
		runner: runner,
	}
}

func (r *IperfRunner) RunTCP(
	ctx context.Context,
	clientNamespace string,
	serverNamespace string,
	serverIP string,
	port int,
	durationSeconds int,
) (IperfResult, error) {
	return r.RunTCPWithProgress(
		ctx,
		clientNamespace,
		serverNamespace,
		serverIP,
		port,
		durationSeconds,
		nil,
	)
}

func (r *IperfRunner) RunTCPWithProgress(
	ctx context.Context,
	clientNamespace string,
	serverNamespace string,
	serverIP string,
	port int,
	durationSeconds int,
	progress ProgressFunc,
) (IperfResult, error) {
	iperfTimeout := iperfCommandTimeout(durationSeconds)
	iperfCtx, cancelIperf := context.WithTimeout(ctx, iperfTimeout)
	defer cancelIperf()

	serverCtx, cancelServer := context.WithCancel(iperfCtx)
	defer cancelServer()

	serverDone := make(chan error, 1)
	go func() {
		_, err := r.runner.RunInNamespace(
			serverCtx,
			serverNamespace,
			"iperf3",
			"-s",
			"-1",
			"-p",
			strconv.Itoa(port),
		)
		serverDone <- err
	}()

	time.Sleep(500 * time.Millisecond)

	clientDone := make(chan struct{})
	if progress != nil {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for second := 1; second <= durationSeconds; second++ {
				select {
				case <-clientDone:
					return
				case <-iperfCtx.Done():
					return
				case <-ticker.C:
					progress(second, durationSeconds)
				}
			}
		}()
	}

	clientOutput, err := r.runner.RunInNamespace(iperfCtx, clientNamespace, "iperf3", "-c", serverIP, "-p", strconv.Itoa(port), "-t", strconv.Itoa(durationSeconds), "-J")
	close(clientDone)
	if err != nil {
		cancelServer()
		if iperfCtx.Err() != nil {
			return IperfResult{}, fmt.Errorf("run iperf client timed out after %s: %w", iperfTimeout, err)
		}
		return IperfResult{}, fmt.Errorf("run iperf client: %w", err)
	}

	select {
	case err := <-serverDone:
		if err != nil && iperfCtx.Err() == nil {
			return IperfResult{}, fmt.Errorf("run iperf server: %w", err)
		}
	case <-time.After(2 * time.Second):
		cancelServer()
	}

	return parseResult(clientOutput.Stdout)
}

func iperfCommandTimeout(
	durationSeconds int,
) time.Duration {
	graceSeconds := 30
	if durationSeconds > graceSeconds {
		graceSeconds = durationSeconds
	}

	return time.Duration(durationSeconds+graceSeconds) * time.Second
}

func parseResult(
	data string,
) (IperfResult, error) {
	var parsed iperfJSON
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return IperfResult{}, fmt.Errorf("parse iperf json: %w", err)
	}

	return IperfResult{
		BitsPerSecondSent:     parsed.End.SumSent.BitsPerSecond,
		BitsPerSecondReceived: parsed.End.SumReceived.BitsPerSecond,
		BytesSent:             parsed.End.SumSent.Bytes,
		BytesReceived:         parsed.End.SumReceived.Bytes,
		Retransmits:           parsed.End.SumSent.Retransmits,
		DurationSeconds:       parsed.End.SumSent.Seconds,
		CPUHostTotal:          parsed.End.CPUUtilizationPercent.HostTotal,
		CPURemoteTotal:        parsed.End.CPUUtilizationPercent.RemoteTotal,
	}, nil
}
