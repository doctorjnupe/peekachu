package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/jameskyle/peekachu"
)

var configFile string
var config = &peekachu.Config{}

func init() {
	//need to switch ip when deploying for prod vs local dev
	//var redisHost = flag.String("source_redis_host", "127.0.0.1:6379" "Redis IP Address:Port")
	flag.StringVar(&configFile, "config", "", "Data Collector Config File")
	flag.Parse()
	if configFile == "" {
		config.Redis.Host = "172.31.2.11"
		config.Redis.Port = 31600
		config.Influxdb.Username = "root"
		config.Influxdb.Password = "root"
		config.Influxdb.Host = "172.31.2.11"
		config.Influxdb.Port = 31410
		config.Influxdb.Database = "charmander-dc"
		config.Interval = 5
		config.Peekachu.Timeout = 300
		config.Peekachu.MaxRetries = 5
	} else {
		file, err := ioutil.ReadFile(configFile)
		if err != nil {
			glog.Errorf("Error reading config file: %s", err)
			os.Exit(1)
		}
		err = json.Unmarshal(file, &config)
		if err != nil {
			glog.Errorf("Error reading config file: %s", err)
			os.Exit(1)
		}
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

	for i := 0; i < 30; i++ {
		pk.RefreshMetricValues()
		pk.Write()
		time.Sleep(10 * time.Second)
	}
}
