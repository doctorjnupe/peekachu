package peekachu

import (
	"errors"
	"fmt"

	"github.com/golang/glog"
	"github.com/jameskyle/pcp"
)

var Filters = FilterListType{
	plugins: make(map[string]FilterFactory),
}

type FilterFactory func(*pcp.Client, *Peekachu) Filterer

type FilterListType struct {
	plugins map[string]FilterFactory
}

func (t *FilterListType) Register(key string, factory FilterFactory) {
	glog.Infof("Registering new filter %s\n", key)
	t.plugins[key] = factory
}

func (t *FilterListType) FilterNames() []string {
	result := []string{}
	for key, _ := range t.plugins {
		result = append(result, key)
	}
	return result
}

func (t *FilterListType) GetFilter(
	key string,
	client *pcp.Client,
	pk *Peekachu,
) (Filterer, error) {
	if _, ok := t.plugins[key]; !ok {
		return nil, errors.New(fmt.Sprintf("Unknown plugin requested: %s", key))
	}
	return t.plugins[key](client, pk), nil
}

func (t *FilterListType) Count() int {
	return len(t.plugins)
}

type Filterer interface {
	Filter(tableName string, row RowMap) (RowMap, error)
}
