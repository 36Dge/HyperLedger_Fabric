package broadcast

import "fabricProject/orderer/common/msgprocessor"

var logger = flogging.MustGetLogger("order.common.broadcast")

//go:generate counterfeiter -o mock/channel_support_registrar.go --fake-name ChannelSupportRegistrar . ChannelSupportRegistrar

//channelSupportRegistrar provides a way for the handler to look up the support of the cahnnel
type ChannelSupportRegistrar interface {

	//broadcastchannelsupport returns the message channel header ,whether the message it a config update
	//and the channel resources for the message or an error if the message is not a message which can
	//be processed directly (like config and orderer_transaction message)
	BroadcastChannelSupport(msg *cb.Envelope)(*cb.ChannelHeader,bool,ChannelSupport,error)

}


//go:generate counterfeiter -o mock/channel_support.go --fake-name ChannelSupport . ChannelSupport
//channelsupport provides the backing resourse needed to support broadcast on a channel
type  ChannelSupport interface {
	msgprocessor.Processor
	Consenter
}

//consenter provideds methods to send message through consenus
type Consenter interface {

	//order accepts a message or return an error indicating the cause of failure
	//it ultimately passed through to the consensue.chan interface

	Order(env *cb.Envelope,configSeq uint64) error

	//configure accepts a reconfiguration or returns an error indicating the cause
	//of it ultimately passes throough to the consensus.chain interfaece


	Configure(config *cb.Envelope,configSeq uint64) error

	//waitReady blocks wating for consenter to be ready or accepting new messages.
	// This is useful when consenter needs to temporarily block ingress messages so
	// that in-flight messages can be consumed. It could return error if consenter is
	// in erroneous states. If this blocking behavior is not desired, consenter could
	WaitReady() error

}

//handler is desgined to handle connections from broadcast AB gRPc service

type Handler struct {
	SupportRegistrar ChannelSupport
	Metrics *Metrics
}




//classifyError converts an error type into a status code.
func ClassifyError(err error) cb.Status {
	switch errors.Cause(err) {
	case msgprocessor.ErrChannelDoesNotExist:
		return cb.Status_NOT_FOUND
	case msgprocessor.ErrPermissionDenied:
		return cb.Status_FORBIDDEN
	case msgprocessor.ErrMaintenanceMode:
		return cb.Status_SERVICE_UNAVAILABLE
	default:
		return cb.Status_BAD_REQUEST
	}
}

//over
