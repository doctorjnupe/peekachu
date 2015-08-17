pkeyackage peekachu

import (
	"errors"
	"fmt"

	"github.com/jameskyle/pcp"
)

type FilterFactory func(*pcp.Client, *Peekachu) Filterer

type FilterListType struct {
	plugins map[string]FilterFactory
}

func (t *FilterListType) Register(key string, factory FilterFactory) {
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
	if _, ok := t.plugins[key]; ok {
		return nil, errors.New(fmt.Sprintf("Unknown plugin requested: %s", key))
	}
	return t.plugins[key](client, pk), nil
}

var Filters = FilterListType{}

type Filterer interface {
	Filter(tableName string, row RowMap) (RowMap, error)
}
