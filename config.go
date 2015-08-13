package peekachu

type Config struct {
	Interval int16
	Peekachu  struct {
		Timeout      int
		MaxRetries   int
		RetriesDelay int `json:"retries-delay"`
	}
	Redis struct {
		Host        string
		Port        uint16
		Nodes       string
		DialTimeout int `json:"dial-timeout"`
	}
	Influxdb struct {
		Host               string
		Port               uint16
		Username           string
		Password           string
		DisableCompression bool `json:"disable-compression"`
		Database           string
		Schema             map[string][]string
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
