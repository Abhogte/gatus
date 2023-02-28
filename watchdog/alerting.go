package watchdog

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/core"

	"github.com/google/uuid"
)

// HandleAlerting takes care of alerts to resolve and alerts to trigger based on result success or failure
func HandleAlerting(endpoint *core.Endpoint, result *core.Result, alertingConfig *alerting.Config, debug bool) {
	fmt.Println("We are in Handle Alerting function")

	if alertingConfig != nil {
		fmt.Println("Are we checking this result now")
		if result.Success {
			fmt.Println("This is the result success")
			fmt.Println("This is a alerting result", result)
			fmt.Println("This is alerting endpoing", endpoint)
			fmt.Println("This is alerting config", alertingConfig)
			handleAlertsToResolve(endpoint, result, alertingConfig, debug)
		} else {
			fmt.Println("This is the result failure")
			handleAlertsToTrigger(endpoint, result, alertingConfig, debug)
		}
	}
	// alertingConfig.AlertConfigVal = 2
	// if alertingConfig == nil {
	// 	fmt.Println("This is alerting config which is: ")
	// 	return
	// }
}

func handleAlertsToTrigger(endpoint *core.Endpoint, result *core.Result, alertingConfig *alerting.Config, debug bool) {
	fmt.Println("Currently in handleAlertsToTrigger method")
	// var res = result
	// var altConfig = alertingConfig
	endpoint.NumberOfSuccessesInARow = 0
	endpoint.NumberOfFailuresInARow++
	fmt.Println(endpoint.TraceId)
	for _, endpointAlert := range endpoint.Alerts {
		fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&", endpoint.NumberOfFailuresInARow)
		// If the alert hasn't been triggered, move to the next one
		// endpointAlert.FailureThreshold = 0
		if !endpointAlert.IsEnabled() || endpointAlert.FailureThreshold > endpoint.NumberOfFailuresInARow {
			fmt.Println("This is handleAlertsToTrigger method first If")
			continue
		}
		fmt.Println("Alert Config Val", endpointAlert.Type)
		// fmt.Println("**************", endpointAlert.Description)
		if endpointAlert.Triggered {
			fmt.Println("This is handleAlertsToTrigger method second If")
			if debug {
				fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
				log.Printf("[watchdog][handleAlertsToTrigger] alert for endpoint=%s with description='%s' has already been TRIGGERED, skipping, %s", endpoint.Name, endpointAlert.GetDescription(), endpoint.TraceId)
			}
			continue
		}
		fmt.Println("this is in for", endpoint.TraceId)
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(endpointAlert.Type)
		fmt.Println("Did the above line executed")
		if alertProvider != nil {
			log.Printf("[watchdog][handleAlertsToTrigger] Sending %s alert because alert for endpoint=%s with description='%s' has been TRIGGERED %s", endpointAlert.Type, endpoint.Name, endpointAlert.GetDescription(), endpoint.TraceId)
			fmt.Println("If alert provide is not equal to null", endpoint.TraceId)
			var err error
			// sliceResult := (*res).Hostname
			// var sliceAlertingConfig = string(*altConfig[:4])
			// uniqueIdentifier := sliceResult + sliceAlertingConfig

			if os.Getenv("MOCK_ALERT_PROVIDER") == "true" {
				if os.Getenv("MOCK_ALERT_PROVIDER_ERROR") == "true" {
					err = errors.New("error")
					// var uniqId = 0

				}
			} else {
				err = alertProvider.Send(endpoint, endpointAlert, result, false)
			}
			if err != nil {
				log.Printf("[watchdog][handleAlertsToTrigger] Failed to send an alert for endpoint=%s: %s, %s", endpoint.Name, err.Error(), endpoint.TraceId)
			} else {
				endpointAlert.Triggered = true
			}
		} else {
			log.Printf("[watchdog][handleAlertsToResolve] Not sending alert of type=%s despite being TRIGGERED, because the provider wasn't configured properly", endpointAlert.Type)
		}
		fmt.Println("We are at the end of the function")
	}
}

func handleAlertsToResolve(endpoint *core.Endpoint, result *core.Result, alertingConfig *alerting.Config, debug bool) {
	fmt.Println("Are we handling the alerts")
	endpoint.NumberOfSuccessesInARow++
	fmt.Println("This is handleAlertsToResolve", endpoint.TraceId)
	for _, endpointAlert := range endpoint.Alerts {
		fmt.Println("in the for loop", endpoint.TraceId)
		if !endpointAlert.IsEnabled() || !endpointAlert.Triggered || endpointAlert.SuccessThreshold > endpoint.NumberOfSuccessesInARow {
			fmt.Println("First If")
			continue
		}
		// Even if the alert provider returns an error, we still set the alert's Triggered variable to false.
		// Further explanation can be found on Alert's Triggered field.
		endpointAlert.Triggered = false
		if !endpointAlert.IsSendingOnResolved() {
			fmt.Println("Second If")
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(endpointAlert.Type)
		if alertProvider != nil {
			fmt.Println("This is the sending alert in handleAlertsToResolve", endpoint.TraceId)
			log.Printf("[watchdog][handleAlertsToResolve] Sending %s alert because alert for endpoint=%s with description='%s' has been RESOLVED", endpointAlert.Type, endpoint.Name, endpointAlert.GetDescription())
			err := alertProvider.Send(endpoint, endpointAlert, result, true)
			if err != nil {
				fmt.Println("This is tge failed alert in handleAlertsToResolve", endpoint.TraceId)
				log.Printf("[watchdog][handleAlertsToResolve] Failed to send an alert for endpoint=%s: %s, %s", endpoint.Name, err.Error(), endpoint.TraceId)
			}
		} else {
			log.Printf("[watchdog][handleAlertsToResolve] Not sending alert of type=%s despite being RESOLVED, because the provider wasn't configured properly", endpointAlert.Type)
		}
	}
	endpoint.NumberOfFailuresInARow = 0
}
func generatingUniqueId() string {
	id := uuid.New()
	fmt.Println("Generated UUID:")
	fmt.Println(id.String())
	return id.String()
}
