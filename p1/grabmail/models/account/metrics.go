package account

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsCanUsedAccountsGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "grabmail_email_accounts_success_num",
	}, []string{"protocol"})
	metricsFailAccountsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "grabmail_email_accounts_fail_num",
	})
)

func metricsCanUsedAccounts(protocol string, num int) {
	metricsCanUsedAccountsGauge.WithLabelValues(protocol).Set(float64(num))
}

func metricsFailAccounts(num int) {
	metricsFailAccountsGauge.Set(float64(num))
}
