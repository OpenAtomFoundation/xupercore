package metrics

import prom "github.com/prometheus/client_golang/prometheus"

const (
	Namespace = "xuperos"

	SubsystemNetwork = "network"
	SubsystemEngine = "engine"
	SubsystemMiner = "miner"
	SubsystemContract = "contract"
	SubsystemLedger = "ledger"
	SubsystemState = "state"
	SubsystemTimer = "timer"

	LabelBCName = "bcname"
	LabelContractModuleName = "contract_module"
	LabelContractName = "contract_name"
	LabelContractMethod = "contract_method"
	LabelContractCode = "contract_code"

	LabelLockType = "lock"
	LabelLockState = "state"

	LabelMessageType = "message"

	LabelTimerMark = "mark"
	LabelCallMethod = "method"
)

// common
var (
	// 锁
	LockCounter = prom.NewCounterVec(
		prom.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemEngine,
			Name: "lock_total",
			Help: "Total number of lock.",
		},
		[]string{LabelBCName, LabelLockType})
	// 函数调用
	CallMethodCounter = prom.NewCounterVec(
		prom.CounterOpts{
			Namespace: Namespace,
			Subsystem: LabelCallMethod,
			Name: "call_total",
			Help: "Total number of call method.",
		},
		[]string{LabelBCName, LabelCallMethod})
	CallMethodHistogram = prom.NewHistogramVec(
		prom.HistogramOpts{
			Namespace: Namespace,
			Subsystem: LabelCallMethod,
			Name: "cost_seconds",
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
		[]string{LabelBCName, LabelContractModuleName, LabelContractName, LabelContractMethod, LabelContractCode})
	ContractInvokeHistogram = prom.NewHistogramVec(
		prom.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemContract,
			Name: "invoke_seconds",
			Help: "Histogram of invoke contract latency.",
		},
		[]string{LabelBCName, LabelContractModuleName, LabelContractName, LabelContractMethod})
)

// ledger
var (
	LedgerConfirmTxCounter = prom.NewCounterVec(
		prom.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemLedger,
			Name: "confirmed_tx_total",
			Help: "Total number of ledger confirmed tx.",
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
	prom.MustRegister(LockCounter)
	// contract
	prom.MustRegister(ContractInvokeCounter)
	prom.MustRegister(ContractInvokeHistogram)
	// ledger
	prom.MustRegister(LedgerConfirmTxCounter)
	prom.MustRegister(LedgerSwitchBranchCounter)
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