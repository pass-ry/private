package grab

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	grabMetricsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "grabmail_email_grab_success_num",
	})
)

func addGrabSuccessNum(num int) {
	grabMetricsCounter.Add(float64(num))
}
