package mail

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsPushTotalCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "grabmail_email_push_success_num",
	})
)

func metricsPushTotal() {
	metricsPushTotalCounter.Inc()
}
