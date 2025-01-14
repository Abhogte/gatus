package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/controller"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
)

func main() {
	cfg, err := loadConfiguration()
	if err != nil {
		panic(err)
	}
	fmt.Println("cfg value in main func :", cfg)
	initializeStorage(cfg)
	start(cfg)
	// Wait for termination signal
	log.Println("Hello we are in main")
	signalChannel := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	log.Println("Hello we are in miain")
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChannel
		log.Println("Received termination signal, attempting to gracefully shut down")
		stop()
		save()
		done <- true
	}()
	<-done
	log.Print("All the logs shut down")
	log.Println("Shutting down")

}

func start(cfg *config.Config) {
	fmt.Println("We are in start")
	fmt.Println(cfg.Endpoints[len(cfg.Endpoints)-1].Group)
	fmt.Println(cfg.Endpoints[len(cfg.Endpoints)-1].Name)
	fmt.Println("cfg value in start func : ", cfg)
	go controller.Handle(cfg)
	fmt.Println("We are in start function")
	watchdog.Monitor(cfg)
	go listenToConfigurationFileChanges(cfg)
}

func stop() {
	watchdog.Shutdown()
	controller.Shutdown()
}

func save() {
	if err := store.Get().Save(); err != nil {
		log.Println("Failed to save storage provider:", err.Error())
	}
}

func loadConfiguration() (*config.Config, error) {

	// GATUS_CONFIG_PATH := "C:\\Users\abhogte\\Deskop\\GatusWork\\AdityaGatus\\config.yaml"
	// configPath := os.Getenv("GATUS_CONFIG_PATH")
	configPath := os.Getenv("GATUS_CONFIG_PATH")
	fmt.Println("****************", configPath)
	// Backwards compatibility
	if len(configPath) == 0 {
		if configPath = os.Getenv("GATUS_CONFIG_FILE"); len(configPath) > 0 {
			log.Println("WARNING: GATUS_CONFIG_FILE is deprecated. Please use GATUS_CONFIG_PATH instead.")
		}
	}
	// return config.LoadConfiguration(configPath)
	return config.LoadConfiguration(configPath)
}

// initializeStorage initializes the storage provider
//
// Q: "TwiN, why are you putting this here? Wouldn't it make more sense to have this in the config?!"
// A: Yes. Yes it would make more sense to have it in the config package. But I don't want to import
// the massive SQL dependencies just because I want to import the config, so here we are.
func initializeStorage(cfg *config.Config) {
	err := store.Initialize(cfg.Storage)
	if err != nil {
		panic(err)
	}
	// Remove all EndpointStatus that represent endpoints which no longer exist in the configuration
	var keys []string
	for _, endpoint := range cfg.Endpoints {
		keys = append(keys, endpoint.Key())
	}
	numberOfEndpointStatusesDeleted := store.Get().DeleteAllEndpointStatusesNotInKeys(keys)
	if numberOfEndpointStatusesDeleted > 0 {
		log.Printf("[main][initializeStorage] Deleted %d endpoint statuses because their matching endpoints no longer existed", numberOfEndpointStatusesDeleted)
	}
}

func listenToConfigurationFileChanges(cfg *config.Config) {
	fmt.Println("listenToConfigurationFileChanges")
	for {
		time.Sleep(30 * time.Second)
		if cfg.HasLoadedConfigurationBeenModified() {
			log.Println("[main][listenToConfigurationFileChanges] Configuration file has been modified")
			stop()
			time.Sleep(time.Second) // Wait a bit to make sure everything is done.
			save()
			updatedConfig, err := loadConfiguration()
			if err != nil {
				if cfg.SkipInvalidConfigUpdate {
					log.Println("[main][listenToConfigurationFileChanges] Failed to load new configuration:", err.Error())
					log.Println("[main][listenToConfigurationFileChanges] The configuration file was updated, but it is not valid. The old configuration will continue being used.")
					// Update the last file modification time to avoid trying to process the same invalid configuration again
					cfg.UpdateLastFileModTime()
					continue
				} else {
					panic(err)
				}
			}
			initializeStorage(updatedConfig)
			start(updatedConfig)
			return
		}
	}
}
