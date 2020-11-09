package localconfig

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

