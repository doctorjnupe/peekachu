package peekachu

import (
	"errors"

	"github.com/golang/glog"
)

type Header []string

type Row []interface{}
type RowMap map[string]interface{}

type Table struct {
	Header Header
	Rows   []Row
}

func (t *Table) AddColumn(colname string) {
	t.Header = append(t.Header, colname)
}

func (t *Table) AddRowFromMap(values map[string]interface{}) error {
	row := Row{}

	if len(t.Header) != len(values) {
		return errors.New("More values than columns!")
	}
	for _, col := range t.Header {
		if _, ok := values[col]; !ok {
			glog.Warningf("No data for column '%s' found!", col)
		}
		row = append(row, values[col])
	}
	t.Rows = append(t.Rows, row)
	return nil
}

func (t *Table) RowsAsInterfaceList() [][]interface{} {
	list := [][]interface{}{}

	for _, row := range t.Rows {
		list = append(list, []interface{}(row))
	}

	return list
}
