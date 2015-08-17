package peekachu

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/jameskyle/pcp"
)

const (
	MESOS_TASK_FILTER_KEY string = "mesos-task-resolver"
)

func init() {
	Filters.Register(MESOS_TASK_FILTER_KEY, NewMesosTaskFilter)
}

type MesosTaskFilter struct {
	Client *pcp.Client
	Port   int16
}

func NewMesosTaskFilter(client *pcp.Client, pk *Peekachu) Filterer {
	return &MesosTaskFilter{
		Client: client,
		Port:   pk.config.MesosTaskResolver.Port,
	}
}

func (r *MesosTaskFilter) HostString() string {
	return fmt.Sprintf("http://%s:%d/getid/", r.Client.Host, r.Port)
}

func (r *MesosTaskFilter) HostStringForId(id string) string {
	return fmt.Sprintf("%s/%s", r.HostString(), id)
}

func (r *MesosTaskFilter) getDockerIdFromContainerId(id string) string {
	result := ""
	if strings.Contains(id, "docker") && len(id) >= 8 {
		i := strings.LastIndex(id, "/")
		result = id[i+1:]
	}

	return result
}

func (r *MesosTaskFilter) Filter(tableName string, row RowMap) (RowMap, error) {
	dockerId := r.getDockerIdFromContainerId(row["instance"].(string))
	if dockerId == "" {
		// the container id was not a docker id, so we filter the row out
		return nil, nil
	}
	glog.Infof("Resolving name for container %s on host %s\n", dockerId, r.Client.Host)
	resp, err := http.Get(r.HostStringForId(dockerId))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(string(body[:]))
	row["interface"] = name
	return row, nil
}
