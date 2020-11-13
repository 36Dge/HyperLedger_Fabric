package msgprocessor


// ChannelConfigTemplator can be used to generate config templates.
type ChannelConfigTemplator interface {
	// NewChannelConfig creates a new template configuration manager.
	NewChannelConfig(env *cb.Envelope) (channelconfig.Resources, error)
}

// MetadataValidator can be used to validate updates to the consensus-specific metadata.
type MetadataValidator interface {
	ValidateConsensusMetadata(oldOrdererConfig, newOrdererConfig channelconfig.Orderer, newChannel bool) error
}

// SystemChannel implements the Processor interface for the system channel.
type SystemChannel struct {
	*StandardChannel
	templator ChannelConfigTemplator
}
