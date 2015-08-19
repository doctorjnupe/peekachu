package peekachu

import "github.com/golang/glog"

const (
	DERIVATIVE_FILTER_KEY string = "derivative"
	INSTANCE_KEY          string = "instance"
	TIME_KEY              string = "time"
)

func init() {
	Filters.Register(DERIVATIVE_FILTER_KEY, NewDerivativeFilter)
}

type DerivativeFilter struct {
	Client *Client
	pk     *Peekachu
}

func NewDerivativeFilter(client *Client, pk *Peekachu) Filterer {
	return &DerivativeFilter{
		Client: client,
		pk:     pk,
	}
}

func (d *DerivativeFilter) filteredMetrics() []string {
	return d.pk.config.Influxdb.MetricFilters[DERIVATIVE_FILTER_KEY]
}

func (d *DerivativeFilter) calcDerivative(v1, v2 int64, t1, t2 int64) int64 {
	glog.Infof("Calculating: (%d - %d) / (%d - %d)\n", v2, v1, t2, t1)
	var result int64
	vdelta := v2 - v1
	tdelta := t2 - t1

	if tdelta == 0 {
		msg := "Current time is equal to previous time. "
		msg += "Cannot calculate derivative."
		glog.Errorln(msg)
		result = 0
	} else {
		result = vdelta / tdelta
	}

	return result
}

func (d *DerivativeFilter) Filter(tableName string, row RowMap) (RowMap, error) {
	glog.V(3).Infof("Processing row for instance %s\n", row[INSTANCE_KEY])
	for _, key := range d.filteredMetrics() {
		glog.V(3).Infof("Looking for key %s\n", key)
		if value, ok := row[key]; ok {
			glog.V(3).Infof("Found key %s\n", key)
			// Row has metric we need to filter, get previous responses for this table
			previousInstanceMap := d.Client.PriorMetricValueResponses[tableName].MetricValueByInstance()
			// now get values for instance
			if instanceValues, ok := previousInstanceMap[row[INSTANCE_KEY].(string)]; ok {
				if previousValue, ok := instanceValues[key]; ok {
					derivative := d.calcDerivative(
						int64(value.(float64)),
						int64(previousValue.(float64)),
						int64(row[TIME_KEY].(uint64)),
						int64(instanceValues[TIME_KEY].(uint64)),
					)
					row[key] = derivative
					glog.Infof("New Row: %v\n", row)
				}
			}
		}
	}
	return row, nil
}
