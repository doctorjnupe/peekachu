package peekachu

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
)

const (
	MESOS_TASK_FILTER_KEY string = "mesos-task-filter"
	FILTERED_FIELD_NAME   string = "instance"
)

func init() {
	Filters.Register(MESOS_TASK_FILTER_KEY, NewMesosTaskFilter)
}

type MesosTaskFilter struct {
	Client *Client
	Port   int16
}

func NewMesosTaskFilter(client *Client, pk *Peekachu) Filterer {
	return &MesosTaskFilter{
		Client: client,
		Port:   pk.config.MesosTaskResolver.Port,
	}
}

func (r *MesosTaskFilter) hostString() string {
	return fmt.Sprintf("http://%s:%d/getid/", r.Client.PcpClient.Host, r.Port)
}

func (r *MesosTaskFilter) hostStringForId(id string) string {
	return fmt.Sprintf("%s/%s", r.hostString(), id)
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
	dockerId := r.getDockerIdFromContainerId(row[FILTERED_FIELD_NAME].(string))
	if dockerId == "" {
		// the container id was not a docker id, so we filter the row out
		return nil, nil
	}
	resp, err := http.Get(r.hostStringForId(dockerId))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(string(body[:]))
	glog.V(3).Infof(
		"Resolving name for container '%s' on host '%s' to '%s'\n",
		dockerId,
		r.Client.PcpClient.Host,
		name,
	)
	row[FILTERED_FIELD_NAME] = name
	return row, nil
}
