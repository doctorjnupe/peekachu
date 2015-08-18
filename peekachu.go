package peekachu

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	influx "github.com/influxdb/influxdb/client"
	"github.com/jameskyle/pcp"

	"gopkg.in/redis.v3"
)

type MetricValueResponseMap map[string]pcp.MetricValueResponseList

type ClientCache struct {
	Client                    *pcp.Client
	Metrics                   pcp.MetricList
	MetricValueResponses      MetricValueResponseMap
	PriorMetricValueResponses MetricValueResponseMap
}

func NewClientCache(client *pcp.Client) *ClientCache {
	cache := ClientCache{Client: client}
	cache.MetricValueResponses = make(MetricValueResponseMap)
	cache.PriorMetricValueResponses = make(MetricValueResponseMap)
	return &cache
}

type Peekachu struct {
	Clients  []*ClientCache
	Influxdb *influx.Client
	Redis    *redis.Client
	config   *Config
}

func NewPeekachu(config *Config) (*Peekachu, error) {
	var err error
	pk := Peekachu{}
	pk.config = config
	pk.initRedis()
	err = pk.initInfluxdb()

	if err != nil {
		return nil, err
	}

	pk.Clients = []*ClientCache{}

	return &pk, nil
}

func (pk *Peekachu) initRedis() {

	url := fmt.Sprintf("%s:%d", pk.config.Redis.Host, pk.config.Redis.Port)
	pk.Redis = redis.NewClient(&redis.Options{
		Addr:       url,
		Password:   "",
		DB:         0,
		MaxRetries: 5,
	})
}

func (pk *Peekachu) initInfluxdb() error {
	url := fmt.Sprintf("%s:%d", pk.config.Influxdb.Host, pk.config.Influxdb.Port)
	c, err := influx.NewClient(&influx.ClientConfig{
		Host:     url,
		Username: pk.config.Influxdb.Username,
		Password: pk.config.Influxdb.Password,
		Database: pk.config.Influxdb.Database,
	})
	if err != nil {
		glog.Errorf("Failed to created Influxdb client: %s\n", err)
		return nil
	}
	pk.Influxdb = c

	if pk.config.Influxdb.DisableCompression {
		pk.Influxdb.DisableCompression()
	}

	// check for database existence
	list, err := pk.Influxdb.GetDatabaseList()
	if err != nil {
		return err
	}

	found := false
	for _, item := range list {
		for key, value := range item {
			if key == "name" {
				if value == pk.config.Influxdb.Database {
					glog.Infof(
						"Found Influxdb database '%s', not creating\n",
						pk.config.Influxdb.Database,
					)
					found = true
					break
				}
			}
		}
		if found {
			break
		}
	}

	if !found {
		glog.Infof(
			"Creating influxdb database '%s'\n",
			pk.config.Influxdb.Database,
		)
		pk.Influxdb.CreateDatabase(pk.config.Influxdb.Database)
	}

	return nil
}

func (pk *Peekachu) Write() error {
	glog.Infoln("Accummulating data for write to database...")
	var instanceMap map[string]map[string]interface{}
	payload := []*influx.Series{}

	for _, cache := range pk.Clients {
		for tableName, responses := range cache.MetricValueResponses {
			table := Table{}
			table.AddColumn("time")
			table.AddColumn("instance")
			table.AddColumn("node")
			for _, name := range responses.MetricNames() {
				table.AddColumn(name)
			}

			instanceMap = responses.MetricValueByInstance()

			// Add the node and time values to each instance metric collection
			for instance, _ := range instanceMap {
				instanceMap[instance]["node"] = cache.Client.Host

				/*
					the instanceMap is of a format:
						instanceMap["eth0"]["firstMetricName"] = metricValue
						instanceMap["eth0"]["secondMetricName"] = metricValue
						instanceMap["eth0"]["time"] = timestamp
						instanceMap["eth0"]["node"] = nodename

					We now restructure it to produce an map
					{"instance": eth0, "time": timestamp, "node": nodename, ....}

					Which is added as a row to our table
				*/
				var row map[string]interface{}
				row = instanceMap[instance]
				row["instance"] = instance
				row = pk.applyFilters(cache.Client, tableName, row)
				if row != nil {
					table.AddRowFromMap(row)
				} else {
					glog.Infof("Row for instance %s filtered.\n", instance)
				}
			}
			series := &influx.Series{
				Name:    tableName,
				Columns: table.Header,
				Points:  table.RowsAsInterfaceList(),
			}
			payload = append(payload, series)
		}
	}

	err := pk.Influxdb.WriteSeriesWithTimePrecision(payload, influx.Second)
	if err != nil {
		return err
	}
	glog.Infof("Wrote %d series to database.", len(payload))

	return nil
}

func (pk *Peekachu) applyFilters(
	client *pcp.Client,
	tableName string,
	row RowMap,
) RowMap {
	for _, filterName := range Filters.FilterNames() {
		if tables, ok := pk.config.Influxdb.SchemaFilters[filterName]; ok {
			for _, table := range tables {
				if table == tableName {
					filterer, err := Filters.GetFilter(filterName, client, pk)

					if err != nil {
						msg := "Error retrieving %s filter: %s\n"
						glog.Errorf(msg, filterName, err)
						glog.Warning("Filter will not be applied!")
						break
					}

					filteredRow, err := filterer.Filter(tableName, row)

					if err != nil {
						msg := "Error applying %s filter: %s\n"
						glog.Errorf(msg, filterName, err)
					} else {
						row = filteredRow
					}

					if row == nil {
						// if row is nil, then the row has been filtered out
						// and we don't need to apply anymore filters
						return nil
					}
				}
			}
		}
	}
	return row
}

func (pk *Peekachu) startTimeout() *time.Timer {
	duration := time.Second * time.Duration(pk.config.Peekachu.Timeout)
	timer := time.NewTimer(duration)

	go func() {
		<-timer.C
		glog.Fatalf("Operation timed out: Reddis Ping Test")
	}()

	return timer
}

func (pk *Peekachu) retriesDelay() {
	duration := time.Second * time.Duration(pk.config.Peekachu.RetriesDelay)
	time.Sleep(duration)
}

func (pk *Peekachu) RedisConnectOrDie() {
	timer := pk.startTimeout()
	retries := 1

	glog.Infoln("Pinging Redis service...")

	for {
		glog.Infof("Attempt %d of %d...\n", retries, pk.config.Peekachu.MaxRetries)

		if pong, ok := pk.Redis.Ping().Result(); ok == nil {
			glog.Infof("Redis says %s. Success!\n", pong)
			timer.Stop()
			break
		} else {
			glog.Infof("Ping failed with error: %s\n", ok)
			retries += 1
			pk.retriesDelay()
		}
		if retries >= pk.config.Peekachu.MaxRetries {
			glog.Fatalf("Max Retries exceeded for Redis Ping test.")
		}
	}
}

func (pk *Peekachu) AllNodes() string {
	return fmt.Sprintf("%s:*", pk.config.Redis.Nodes)
}

func (pk *Peekachu) GetNodes() []string {
	glog.Infof("Performing nodes query: %s\n", pk.AllNodes())
	nodes := []string{}

	results, err := pk.Redis.Keys(pk.AllNodes()).Result()

	if err == redis.Nil {
		glog.Errorf("%s key does not exist!", pk.AllNodes())
	}

	for _, result := range results {
		split := strings.Split(result, ":")
		nodes = append(nodes, split[len(split)-1])
	}

	return nodes
}

func (pk *Peekachu) RefreshClients() {
	pk.Clients = []*ClientCache{}
	nodes := pk.GetNodes()

	timer := pk.startTimeout()
	retries := 1

	for {
		if retries >= pk.config.Peekachu.MaxRetries {
			glog.Fatalf("Max Retries exceeded for Node query.")
		}

		if len(nodes) == 0 {
			glog.Info("Waiting for nodes to come online....")
			pk.retriesDelay()
			retries += 1
			nodes = pk.GetNodes()
		} else {
			glog.Infof("Found %d nodes: %v\n", len(nodes), nodes)
			timer.Stop()
			break
		}
	}

	for _, node := range nodes {
		context := pcp.NewContext("", pk.config.PCP.HostSpec)
		context.PollTimeout = pk.config.PCP.ContextPollTimeout
		client := pcp.NewClient(node, pk.config.PCP.Port, context)
		client.RefreshContext()
		mquery := pcp.NewMetricQuery("")
		metrics, err := client.Metrics(mquery)
		if err != nil {
			glog.Errorf("Error fetching metrics for client: %s", err)
		}
		cache := NewClientCache(client)
		cache.Metrics = metrics
		pk.Clients = append(pk.Clients, cache)
	}
}

func (pk *Peekachu) refreshMetricValuesForClient(
	cache *ClientCache,
	table string,
	names []string,
) {

	glog.Infof("Fetching metric values for host %s...", cache.Client.Host)
	query := pcp.NewMetricValueQuery(names, []string{})

	if resp, err := cache.Client.MetricValues(query); err != nil {
		msg := "Failed to retrieve metric values from host %s : %s\n"
		glog.Errorf(msg, cache.Client.Host, err)
	} else {
		cache.PriorMetricValueResponses = cache.MetricValueResponses

		cache.MetricValueResponses[table] = append(
			cache.MetricValueResponses[table],
			resp,
		)

		for _, value := range resp.Values {
			// FIXME: probably ought to name MetricValue.Name to
			//		  MetricValue.MetricName for clarity
			metric := cache.Metrics.FindMetricByName(value.MetricName)
			indom, err := cache.Client.GetIndomForMetric(metric)
			if err != nil {
				glog.Errorf("Failed to get indom for metric: %s\n", err)
			}
			value.UpdateInstanceNames(indom)
		}
	}
}

func (pk *Peekachu) RefreshMetricValues() {
	for table, metricNames := range pk.config.Influxdb.Schema {
		for _, cache := range pk.Clients {
			pk.refreshMetricValuesForClient(cache, table, metricNames)
		}
	}
}

func (pk *Peekachu) Close() {
	pk.Redis.Close()
}
