package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/doctorjnupe/peekachu"
)

var configFile string
var config *peekachu.Config

func init() {
	//need to switch ip when deploying for prod vs local dev
	//var redisHost = flag.String("source_redis_host", "127.0.0.1:6379" "Redis IP Address:Port")
	config = peekachu.NewConfig()

	flag.StringVar(&configFile, "config", "", "Data Collector Config File")
	flag.Parse()

	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		glog.Errorf("Error reading config file: %s", err)
		os.Exit(1)
	}
	err = json.Unmarshal(file, config)
	if err != nil {
		glog.Errorf("Error reading config file: %s", err)
		os.Exit(1)
	}
}

func main() {
	glog.Infoln("Peekachu Data Collector Initialization...")
	glog.Infof("Loaded %d filters..\n", peekachu.Filters.Count())
	pk, err := peekachu.NewPeekachu(config)
	if err != nil {
		glog.Fatalf("Failed to create Pickachu: %s\n", err)
	}
	defer pk.Close()
	// Wait for redis, die after timeout or max retries exceeded
	pk.RedisConnectOrDie()
	pk.RefreshClients()

	throttle := time.Tick(pk.RateLimit())
	count := 1

	for {
		if count%pk.ClientRefreshRate() == 0 {
			pk.RefreshClients()
			count = 0
		}
		pk.RefreshMetricValues()
		pk.Write()
		<-throttle
		count += 1
	}
}
