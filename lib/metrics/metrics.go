package metrics

import prom "github.com/prometheus/client_golang/prometheus"

const (
	Namespace = "xuperos"

	SubsystemCommon = "common"
	SubsystemLedger = "ledger"
	SubsystemContract = "contract"
	SubsystemState = "state"
	SubsystemNetwork = "network"

	LabelBCName = "bcname"
	LabelMessageType = "message"
	LabelCallMethod = "method"
	LabelHeight = "height"
	LabelErrorCode = "code"
	LabelContractModuleName = "contract_module"
	LabelContractName = "contract_name"
	LabelContractMethod = "contract_method"
)

// common
var (
	// 函数调用
	CallMethodCounter = prom.NewCounterVec(
		prom.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemCommon,
			Name: "call_method_total",
			Help: "Total number of call method.",
		},
		[]string{LabelBCName, LabelCallMethod, LabelErrorCode})
	CallMethodHistogram = prom.NewHistogramVec(
		prom.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemCommon,
			Name: "call_method_seconds",
			Help: "Histogram of call method cost latency.",
			Buckets: prom.DefBuckets,
		},
		[]string{LabelBCName, LabelCallMethod})
)

// contract
var (
	ContractInvokeCounter = prom.NewCounterVec(
		prom.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemContract,
			Name: "invoke_total",
			Help: "Total number of invoke contract latency.",
		},
		[]string{LabelBCName, LabelContractModuleName, LabelContractName, LabelContractMethod, LabelErrorCode})
	ContractInvokeHistogram = prom.NewHistogramVec(
		prom.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemContract,
			Name: "invoke_seconds",
			Help: "Histogram of invoke contract latency.",
			Buckets: prom.DefBuckets,
		},
		[]string{LabelBCName, LabelContractModuleName, LabelContractName, LabelContractMethod})
)

// ledger
var (
	TxPerBlockHistogram = prom.NewHistogramVec(
		prom.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemLedger,
			Name: "tx_per_block",
			Help: "Histogram of tx_per_block.",
			Buckets: prom.LinearBuckets(0, 500, 10),
		},
		[]string{LabelBCName})
	LedgerConfirmTxCounter = prom.NewCounterVec(
		prom.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemLedger,
			Name: "confirmed_tx_total",
			Help: "Total number of ledger confirmed tx.",
		},
		[]string{LabelBCName})
	LedgerHeightGauge = prom.NewGaugeVec(
		prom.GaugeOpts{
			Namespace: Namespace,
			Subsystem: SubsystemLedger,
			Name: "height_total",
			Help: "Total number of ledger height.",
		},
		[]string{LabelBCName})
	LedgerSwitchBranchCounter = prom.NewCounterVec(
		prom.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemLedger,
			Name: "switch_branch_total",
			Help: "Total number of ledger switch branch.",
		},
		[]string{LabelBCName})
)

// state
var (
	StateUnconfirmedTxGauge = prom.NewGaugeVec(
		prom.GaugeOpts{
			Namespace: Namespace,
			Subsystem: SubsystemState,
			Name: "unconfirmed_tx_gauge",
			Help: "Total number of miner unconfirmed tx.",
		},
		[]string{LabelBCName})
)

// network
var (
	NetworkMsgSendCounter = prom.NewCounterVec(
		prom.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name: "msg_send_total",
			Help: "Total number of P2P send message.",
		},
		[]string{LabelBCName, LabelMessageType})
	NetworkMsgSendBytesCounter = prom.NewCounterVec(
		prom.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name: "msg_send_bytes",
			Help: "Total size of P2P send message.",
		},
		[]string{LabelBCName, LabelMessageType})
	NetworkClientHandlingHistogram = prom.NewHistogramVec(
		prom.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name: "client_handled_seconds",
			Help: "Histogram of response latency (seconds) of P2P handled.",
			Buckets: prom.DefBuckets,
		},
		[]string{LabelBCName, LabelMessageType})


	NetworkMsgReceivedCounter = prom.NewCounterVec(
		prom.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name: "msg_received_total",
			Help: "Total number of P2P received message.",
		},
		[]string{LabelBCName, LabelMessageType})
	NetworkMsgReceivedBytesCounter = prom.NewCounterVec(
		prom.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name: "msg_received_bytes",
			Help: "Total size of P2P received message.",
		},
		[]string{LabelBCName, LabelMessageType})
	NetworkServerHandlingHistogram = prom.NewHistogramVec(
		prom.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name: "server_handled_seconds",
			Help: "Histogram of response latency (seconds) of P2P handled.",
			Buckets: prom.DefBuckets,
		},
		[]string{LabelBCName, LabelMessageType})
)

func RegisterMetrics() {
	// common
	prom.MustRegister(CallMethodCounter)
	prom.MustRegister(CallMethodHistogram)
	// contract
	prom.MustRegister(ContractInvokeCounter)
	prom.MustRegister(ContractInvokeHistogram)
	// ledger
	prom.MustRegister(LedgerConfirmTxCounter)
	prom.MustRegister(TxPerBlockHistogram)
	prom.MustRegister(LedgerSwitchBranchCounter)
	prom.MustRegister(LedgerHeightGauge)
	// state
	prom.MustRegister(StateUnconfirmedTxGauge)
	// network
	prom.MustRegister(NetworkMsgSendCounter)
	prom.MustRegister(NetworkMsgSendBytesCounter)
	prom.MustRegister(NetworkClientHandlingHistogram)
	prom.MustRegister(NetworkMsgReceivedCounter)
	prom.MustRegister(NetworkMsgReceivedBytesCounter)
	prom.MustRegister(NetworkServerHandlingHistogram)
}