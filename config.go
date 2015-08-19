package peekachu

const (
	NETWORK_TABLE_KEY   string = "network"
	MACHINE_TABLE_KEY   string = "machine"
	CONTAINER_TABLE_KEY string = "containers"
	SCHEMA_FILTERS_KEY  string = "schema-filters"
	METRIC_FILTERS_KEY  string = "metric-filters"
)

type Config struct {
	Peekachu struct {
		Timeout           int
		MaxRetries        int
		RetriesDelay      int `json:"retries-delay"`
		RateLimit         int `json:"rate-limit"`
		ClientRefreshRate int `json:"client-refrehs-rate"`
	}
	MesosTaskResolver struct {
		Port int16
	} `json:"mesos-task-resolver"`
	Redis struct {
		Host  string
		Port  uint16
		Nodes string
	}
	Influxdb struct {
		Host               string
		Port               uint16
		Username           string
		Password           string
		DisableCompression bool `json:"disable-compression"`
		Database           string
		Schema             map[string][]string
		SchemaFilters      map[string][]string `json:"schema-filters"`
		MetricFilters      map[string][]string `json:"metric-filters"`
	}
	Mesos struct {
		Port uint16
	}
	PCP struct {
		Port               uint16
		HostSpec           string `json:"hostspec"`
		ContextPollTimeout int32  `json:"context-polltimeout"`
	}
}

func NewConfig() *Config {
	config := &Config{}
	config.Peekachu.Timeout = 300
	config.Peekachu.MaxRetries = 30
	config.Peekachu.RetriesDelay = 10
	config.Peekachu.RateLimit = 5
	config.Peekachu.ClientRefreshRate = 4
	config.MesosTaskResolver.Port = 31300
	config.Redis.Host = "172.31.2.11"
	config.Redis.Port = 31600
	config.Redis.Nodes = "charmander:nodes"
	config.Influxdb.Host = "172.31.2.11"
	config.Influxdb.Port = 31410
	config.Influxdb.Username = "root"
	config.Influxdb.Password = "root"
	config.Influxdb.DisableCompression = true
	config.Influxdb.Database = "charmander-dc"
	config.Influxdb.Schema = make(map[string][]string)
	config.Influxdb.Schema[NETWORK_TABLE_KEY] = []string{
		"network.interface.in.bytes",
		"network.interface.out.bytes",
		"network.interface.out.drops",
		"network.interface.in.drops",
	}
	config.Influxdb.Schema[MACHINE_TABLE_KEY] = []string{
		"kernel.all.cpu.user",
		"kernel.all.cpu.sys",
		"mem.util.used",
	}
	config.Influxdb.Schema[CONTAINER_TABLE_KEY] = []string{
		"cgroup.cpuacct.stat.user",
		"cgroup.cpuacct.stat.system",
		"cgroup.memory.usage",
	}

	config.Influxdb.SchemaFilters = make(map[string][]string)
	config.Influxdb.SchemaFilters[MESOS_TASK_FILTER_KEY] = []string{
		CONTAINER_TABLE_KEY,
	}
	config.Influxdb.MetricFilters = make(map[string][]string)
	config.Influxdb.MetricFilters[DERIVATIVE_FILTER_KEY] = []string{
		"cgroup.memory.usage",
	}
	config.Mesos.Port = 31300
	config.PCP.Port = 44323
	config.PCP.HostSpec = "localhost"
	config.PCP.ContextPollTimeout = 12

	return config
}
