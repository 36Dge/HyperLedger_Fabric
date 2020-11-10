package localconfig

import "time"

var logger = flogging.MustGetLogger("localconfig")

//toplevel dircetly corresponds to the orderconfig yaml.
type TopLevel struct {
	General              General
	FileLedger           FileLedger
	Kafka                Kafka
	Debug                Debug
	Consensus            interface{}
	Operations           Operations
	Metrics              Metrics
	ChannelParticipation ChannelParticipation
	Admin                Admin
}

//general contains config which should be commom among all order types.

type Generanl struct {
	ListenAddress     string
	ListenPort        uint16
	TLS               TLS
	Cluster           Cluster
	Keepalive         Keepalive
	ConnectionTimeout time.Duration
	GenesisMethod     string // For compatibility only, will be replaced by BootstrapMethod
	GenesisFile       string // For compatibility only, will be replaced by BootstrapFile
	BootstrapMethod   string
	BootstrapFile     string
	Profile           Profile
	LocalMSPDir       string
	LocalMSPID        string
	BCCSP             *bccsp.FactoryOpts
	Authentication    Authentication
}

type Cluster struct {
	ListenAddress                        string
	ListenPort                           uint16
	ServerCertificate                    string
	ServerPrivateKey                     string
	ClientCertificate                    string
	ClientPrivateKey                     string
	RootCAs                              []string
	DialTimeout                          time.Duration
	RPCTimeout                           time.Duration
	ReplicationBufferSize                int
	ReplicationPullTimeout               time.Duration
	ReplicationRetryTimeout              time.Duration
	ReplicationBackgroundRefreshInterval time.Duration
	ReplicationMaxRetries                int
	SendBufferSize                       int
	CertExpirationWarningThreshold       time.Duration
	TLSHandshakeTimeShift                time.Duration
}

//keepalive contains configuration for grpc servers.
type Keepalive struct {
	ServerMinInterval time.Duration
	ServerInterval    time.Duration
	ServerTimeout     time.Duration
}

//tls contains configuration for TLS connections.
type TLS struct {
	Enabled               bool
	PrivateKey            string
	Certificate           string
	RootCAs               []string
	ClientAuthRequired    bool
	ClientRootCAs         []string
	TLSHandshakeTimeShift time.Duration
}

type SASLPlain struct {
	Enable   bool
	User     string
	Password string
}

//authentication contains configuration parameters related to authenticating
//client message

type Authentication struct {
	TimeWindow         time.Duration
	NoExpirationChecks bool
}

//profile contains configuration for go pprof profiling
type Profile struct {
	Enable  bool
	Address string
}

//fileledger contains configuration for the file-based ledger.
type FileLedger struct {
	Location string
	Prefix   string //for compatibility only.the setting is no longer supported.
}

//kafka contains configuration for hte kafka-based orderer
type Kafka struct {
	Retry     Retry
	Verbose   bool
	Version   sarama.KafkaVersion
	TLS       TLS
	SASLPlain SASLPlain
	Topic     Topic
}

//retry contains configuration to retries and timeout when the
//connection to the kakfa culuter cannot be established ,or when
//metadata requests neeed to be repeated (because the cluster is
//the middle of a leader election ).
type Retry struct {
	ShortTnterval   time.Duration
	ShortTotal      time.Duration
	LongInterval    time.Duration
	LongTotal       time.Duration
	NetworkTimeouts NetworkTimeouts
	Metadata        Metadata
	Producer        Producer
	Consumer        Consumer
}

//newworktimeout contains the socket timeouts for network requests to the
//kafka cluster.
type NetworkTimeouts struct {
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

//metadata contains configuration for the metadata requests to the kafka
//cluster.
type Metadata struct {
	RetryMax     int
	RetryBackoff time.Duration
}

// Producer contains configuration for the producer's retries when failing to
// post a message to a Kafka partition.
type Producer struct {
	RetryMax     int
	RetryBackoff time.Duration
}

// Consumer contains configuration for the consumer's retries when failing to
// read from a Kafa partition.
type Consumer struct {
	RetryBackoff time.Duration
}

//the topic contains the setting to use when creating kafka topics
type Topic struct {
	ReplicationFactor int16
}

//debug contains configuration for the order,s dubug parameters
type Debug struct {
	BroadcastTraceDir string
	DeliverTraceDir   string
}

//operations configuration the operation endpoint for the orderer.
type Operations struct {
	ListenAddress string
	TLS           TLS
}

//metrics configures the metrics provider for the orderer.
type Metrics struct {
	Provider string
	Statsd   Statsd
}

//statsd provides the configuration required to emit statsd metrics form the orders
type Statsd struct {
	Network      string
	Address      string
	WireInterval time.Duration
	Prefix       string
}

//admin configures the admin endpoint for the order.
type Admin struct {
	ListenAddress string
	TLS TLS
}


// ChannelParticipation provides the channel participation API configuration for the orderer.
// Channel participation uses the same ListenAddress and TLS settings of the Operations service.
type ChannelParticipation struct {
	Enabled            bool
	MaxRequestBodySize uint32
}



















