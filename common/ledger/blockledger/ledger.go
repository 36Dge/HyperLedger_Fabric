package blockledger

//factory retrieves or create new ledger by channelId
type Factory interface {
	//getorcreate gets an existing ledger(it it exists)or creates it if does not
	GetOrCreate(channelID string) (ReadWriter,error)

	//remove removes an existing ledger
	Remove(channelID string) error

	//channelIDS returns the channel IDS the factory is awary of
	ChannelIDs() []string

	//close release all resources acquireed by the factory
	Close()
}


// Iterator is useful for a chain Reader to stream blocks as they are created
type Iterator interface {
	// Next blocks until there is a new block available, or returns an error if
	// the next block is no longer retrievable
	Next() (*cb.Block, cb.Status)
	// Close releases resources acquired by the Iterator
	Close()
}

// Reader allows the caller to inspect the ledger
type Reader interface {
	// Iterator returns an Iterator, as specified by an ab.SeekInfo message, and
	// its starting block number
	Iterator(startType *ab.SeekPosition) (Iterator, uint64)
	// Height returns the number of blocks on the ledger
	Height() uint64
}

// Writer allows the caller to modify the ledger
type Writer interface {
	// Append a new block to the ledger
	Append(block *cb.Block) error
}

// ReadWriter encapsulates the read/write functions of the ledger
type ReadWriter interface {
	Reader
	Writer
}

//over
