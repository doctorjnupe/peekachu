package peekachu

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
)

type Resolver struct {
	Host string
	Port int16
}

func NewResolver(host string, port int16) *Resolver {
	return &Resolver{Host: host, Port: port}
}

func (r *Resolver) HostString() string {
	return fmt.Sprintf("http://%s:%d/getid/", r.Host, r.Port)
}

func (r *Resolver) HostStringForId(id string) string {
	return fmt.Sprintf("%s/%s", r.HostString(), id)
}

func (r *Resolver) getDockerIdFromContainerId(id string) string {
	result := ""
	if strings.Contains(id, "docker") && len(id) >= 8 {
		i := strings.LastIndex(id, "/")
		result = id[i+1:]
	}

	return result
}

func (r *Resolver) Resolve(containerId string) (string, error) {
	dockerId := r.getDockerIdFromContainerId(containerId)
	if dockerId == "" {
		// the container id was not a docker id, so we filter it out
		return "", nil
	}
	glog.Infof("Resolving name for container %s on host %s\n", dockerId, r.Host)
	resp, err := http.Get(r.HostStringForId(dockerId))

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(body[:]))
	glog.Infof("Resolved %s to %s\n", dockerId, name)
	return name, nil
}
