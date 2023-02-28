package watchdog

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
)

var (
	// monitoringMutex is used to prevent multiple endpoint from being evaluated at the same time.
	// Without this, conditions using response time may become inaccurate.
	monitoringMutex sync.Mutex

	ctx        context.Context
	cancelFunc context.CancelFunc
)

// Monitor loops over each endpoint and starts a goroutine to monitor each endpoint separately
func Monitor(cfg *config.Config) {
	ctx, cancelFunc = context.WithCancel(context.Background())
	fmt.Println("cfg value in Monitor func: ", cfg)
	for _, endpoint := range cfg.Endpoints {
		if endpoint.IsEnabled() {
			// To prevent multiple requests from running at the same time, we'll wait for a little before each iteration
			time.Sleep(777 * time.Millisecond)
			go monitor(endpoint, cfg.Alerting, cfg.Maintenance, cfg.DisableMonitoringLock, cfg.Metrics, cfg.Debug, ctx)
		}
	}
}

// monitor a single endpoint in a loop
func monitor(endpoint *core.Endpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, disableMonitoringLock, enabledMetrics, debug bool, ctx context.Context) {
	// Run it immediately on start
	fmt.Println("cfg value in small monitor func: ")
	execute(endpoint, alertingConfig, maintenanceConfig, disableMonitoringLock, enabledMetrics, debug)
	// Loop for the next executions
	for {
		select {
		case <-ctx.Done():
			log.Printf("[watchdog][monitor] Canceling current execution of group=%s; endpoint=%s", endpoint.Group, endpoint.Name)
			return
		case <-time.After(endpoint.Interval):
			fmt.Println("This is where the execute is called")
			execute(endpoint, alertingConfig, maintenanceConfig, disableMonitoringLock, enabledMetrics, debug)
			fmt.Println("So we return from the execute")
		}
	}
}

func execute(endpoint *core.Endpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, disableMonitoringLock, enabledMetrics, debug bool) {
	fmt.Println("We are checking the execute function")
	if !disableMonitoringLock {
		// By placing the lock here, we prevent multiple endpoints from being monitored at the exact same time, which
		// could cause performance issues and return inaccurate results
		monitoringMutex.Lock()
	}

	if debug {
		log.Printf("[watchdog][execute] Monitoring group=%s; endpoint=%s", endpoint.Group, endpoint.Name)
	}
	result := endpoint.EvaluateHealth()
	if enabledMetrics {
		metrics.PublishMetricsForEndpoint(endpoint, result)
	}
	UpdateEndpointStatuses(endpoint, result)
	log.Printf(
		"[watchdog][execute] Monitored group=%s; endpoint=%s; success=%v; errors=%d; duration=%s",
		endpoint.Group,
		endpoint.Name,
		result.Success,
		len(result.Errors),
		result.Duration.Round(time.Millisecond),
	)
	if !maintenanceConfig.IsUnderMaintenance() {
		// TODO: Consider moving this after the monitoring lock is unlocked? I mean, how much noise can a single alerting provider cause...
		fmt.Println("After this HandleAlerting will be called")
		fmt.Println("****this is the result****", result)
		HandleAlerting(endpoint, result, alertingConfig, debug)
		fmt.Println("We have returned from the function")
	} else if debug {
		fmt.Println("Not handling alerting because currently in the maintenance window")
		log.Println("[watchdog][execute] Not handling alerting because currently in the maintenance window")
	}
	if debug {
		fmt.Println("Waiting for interval before monitoring group endpoint again")
		log.Printf("[watchdog][execute] Waiting for interval=%s before monitoring group=%s endpoint=%s again", endpoint.Interval, endpoint.Group, endpoint.Name)
	}
	if !disableMonitoringLock {
		fmt.Println("The monitoring lock is not disabled")
		monitoringMutex.Unlock()
	}
}

// UpdateEndpointStatuses updates the slice of endpoint statuses
func UpdateEndpointStatuses(endpoint *core.Endpoint, result *core.Result) {
	fmt.Println("So we are in updateEndPointStatuses")
	if err := store.Get().Insert(endpoint, result); err != nil {
		log.Println("[watchdog][UpdateEndpointStatuses] Failed to insert data in storage:", err.Error())
	}
}

// Shutdown stops monitoring all endpoints
func Shutdown() {
	cancelFunc()
}
