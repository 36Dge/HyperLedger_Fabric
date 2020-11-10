package server

import (
	"fabricProject/orderer/common/localconfig"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"time"
)

var logger = flogging.MustGetLogger("orderer.common.server")

//command line flags
var (
	app = kingpin.New("orderer", "Hyperledger Fabric orderer node")

	_       = app.Command("start", "Start the orderer node").Default() // preserved for cli compatibility
	version = app.Command("version", "Show version information")

	clusterTypes = map[string]struct{}{"etcdraft": {}}
)

//main is the entry point of orderer process
func Main() {
	fullCmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	//"version" command
	if fullCmd == version.FullCommand() {
		fmt.Println(metadata.GetVersionInfo())
		return
	}

	conf, err := localcofig.Load()
	if err != nil {
		logger.Error("failed to parse config:", err)
		os.Exit(1)
	}

	initializeLogging()

	prettyPrintStruct(conf)

	cryptoProvider := factory.GetDefault()

	signer, signErr := loadLocalMSP(conf).GetDefaultSigningIdentity()
	if signErr != nil {
		logger.Panicf("Failed to get local MSP identity: %s", signErr)
	}

	opsSystem := newOperationsSystem(conf.Operations, conf.Metrics)
	if err = opsSystem.Start(); err != nil {
		logger.Panicf("failed to start operations subsystem: %s", err)
	}
	defer opsSystem.Stop()
	metricsProvider := opsSystem.Provider
	logObserver := floggingmetrics.NewObserver(metricsProvider)
	flogging.SetObserver(logObserver)

	serverConfig := initializeServerConfig(conf, metricsProvider)
	grpcServer := initializeGrpcServer(conf, serverConfig)
	caMgr := &caManager{
		appRootCAsByChain:     make(map[string][][]byte),
		ordererRootCAsByChain: make(map[string][][]byte),
		clientRootCAs:         serverConfig.SecOpts.ClientRootCAs,
	}

	lf, err := createLedgerFactory(conf, metricsProvider)
	if err != nil {
		logger.Panicf("Failed to create ledger factory: %v", err)
	}

	var bootstrapBlock *cb.Block
	switch conf.General.BootstrapMethod {
	case "file":
		if len(lf.ChannelIDs()) > 0 {
			logger.Info("Not bootstrapping the system channel because of existing channels")
			break
		}

		bootstrapBlock = file.New(conf.General.BootstrapFile).GenesisBlock()
		if err := onboarding.ValidateBootstrapBlock(bootstrapBlock, cryptoProvider); err != nil {
			logger.Panicf("Failed validating bootstrap block: %v", err)
		}

		if bootstrapBlock.Header.Number > 0 {
			logger.Infof("Not bootstrapping the system channel because the bootstrap block number is %d (>0), replication is needed", bootstrapBlock.Header.Number)
			break
		}

		// bootstrapping with a genesis block (i.e. bootstrap block number = 0)
		// generate the system channel with a genesis block.
		logger.Info("Bootstrapping the system channel")
		initializeBootstrapChannel(bootstrapBlock, lf)
	case "none":
		bootstrapBlock = initSystemChannelWithJoinBlock(conf, cryptoProvider, lf)
	default:
		logger.Panicf("Unknown bootstrap method: %s", conf.General.BootstrapMethod)
	}

	// select the highest numbered block among the bootstrap block and the last config block if the system channel.
	sysChanConfigBlock := extractSystemChannel(lf, cryptoProvider)
	clusterBootBlock := selectClusterBootBlock(bootstrapBlock, sysChanConfigBlock)

	// determine whether the orderer is of cluster type
	var isClusterType bool
	if clusterBootBlock == nil {
		logger.Infof("Starting without a system channel")
		isClusterType = true
	} else {
		sysChanID, err := protoutil.GetChannelIDFromBlock(clusterBootBlock)
		if err != nil {
			logger.Panicf("Failed getting channel ID from clusterBootBlock: %s", err)
		}

		consensusTypeName := consensusType(clusterBootBlock, cryptoProvider)
		logger.Infof("Starting with system channel: %s, consensus type: %s", sysChanID, consensusTypeName)
		_, isClusterType = clusterTypes[consensusTypeName]
	}

	// configure following artifacts properly if orderer is of cluster type
	var repInitiator *onboarding.ReplicationInitiator
	clusterServerConfig := serverConfig
	clusterGRPCServer := grpcServer // by default, cluster shares the same grpc server
	var clusterClientConfig comm.ClientConfig
	var clusterDialer *cluster.PredicateDialer

	var reuseGrpcListener bool
	var serversToUpdate []*comm.GRPCServer

	if isClusterType {
		logger.Infof("Setting up cluster")
		clusterClientConfig = initializeClusterClientConfig(conf)
		clusterDialer = &cluster.PredicateDialer{
			Config: clusterClientConfig,
		}

		if reuseGrpcListener = reuseListener(conf); !reuseGrpcListener {
			clusterServerConfig, clusterGRPCServer = configureClusterListener(conf, serverConfig, ioutil.ReadFile)
		}

		// If we have a separate gRPC server for the cluster,
		// we need to update its TLS CA certificate pool.
		serversToUpdate = append(serversToUpdate, clusterGRPCServer)

		// If the orderer has a system channel and is of cluster type, it may have
		// to replicate first.
		if clusterBootBlock != nil {
			// When we are bootstrapping with a clusterBootBlock with number >0,
			// replication will be performed. Only clusters that are equipped with
			// a recent config block (number i.e. >0) can replicate. This will
			// replicate all channels if the clusterBootBlock number > system-channel
			// height (i.e. there is a gap in the ledger).
			repInitiator = onboarding.NewReplicationInitiator(lf, clusterBootBlock, conf, clusterClientConfig.SecOpts, signer, cryptoProvider)
			repInitiator.ReplicateIfNeeded(clusterBootBlock)
			// With BootstrapMethod == "none", the bootstrapBlock comes from a
			// join-block. If it exists, we need to remove the system channel
			// join-block from the filerepo.
			if conf.General.BootstrapMethod == "none" && bootstrapBlock != nil {
				discardSystemChannelJoinBlock(conf, bootstrapBlock)
			}
		}
	}

	identityBytes, err := signer.Serialize()
	if err != nil {
		logger.Panicf("Failed serializing signing identity: %v", err)
	}

	expirationLogger := flogging.MustGetLogger("certmonitor")
	crypto.TrackExpiration(
		serverConfig.SecOpts.UseTLS,
		serverConfig.SecOpts.Certificate,
		[][]byte{clusterClientConfig.SecOpts.Certificate},
		identityBytes,
		expirationLogger.Infof,
		expirationLogger.Warnf, // This can be used to piggyback a metric event in the future
		time.Now(),
		time.AfterFunc)

	// if cluster is reusing client-facing server, then it is already
	// appended to serversToUpdate at this point.
	if grpcServer.MutualTLSRequired() && !reuseGrpcListener {
		serversToUpdate = append(serversToUpdate, grpcServer)
	}

	tlsCallback := func(bundle *channelconfig.Bundle) {
		logger.Debug("Executing callback to update root CAs")
		caMgr.updateTrustedRoots(bundle, serversToUpdate...)
		if isClusterType {
			caMgr.updateClusterDialer(
				clusterDialer,
				clusterClientConfig.SecOpts.ServerRootCAs,
			)
		}
	}

	manager := initializeMultichannelRegistrar(
		clusterBootBlock,
		repInitiator,
		clusterDialer,
		clusterServerConfig,
		clusterGRPCServer,
		conf,
		signer,
		metricsProvider,
		opsSystem,
		lf,
		cryptoProvider,
		tlsCallback,
	)

	adminServer := newAdminServer(conf.Admin)
	adminServer.RegisterHandler(
		channelparticipation.URLBaseV1,
		channelparticipation.NewHTTPHandler(conf.ChannelParticipation, manager),
		conf.Admin.TLS.Enabled,
	)
	if err = adminServer.Start(); err != nil {
		logger.Panicf("failed to start admin server: %s", err)
	}
	defer adminServer.Stop()

	mutualTLS := serverConfig.SecOpts.UseTLS && serverConfig.SecOpts.RequireClientCert
	server := NewServer(
		manager,
		metricsProvider,
		&conf.Debug,
		conf.General.Authentication.TimeWindow,
		mutualTLS,
		conf.General.Authentication.NoExpirationChecks,
	)

	logger.Infof("Starting %s", metadata.GetVersionInfo())
	handleSignals(addPlatformSignals(map[os.Signal]func(){
		syscall.SIGTERM: func() {
			grpcServer.Stop()
			if clusterGRPCServer != grpcServer {
				clusterGRPCServer.Stop()
			}
		},
	}))

	if !reuseGrpcListener && isClusterType {
		logger.Info("Starting cluster listener on", clusterGRPCServer.Address())
		go clusterGRPCServer.Start()
	}

	if conf.General.Profile.Enabled {
		go initializeProfilingService(conf)
	}
	ab.RegisterAtomicBroadcastServer(grpcServer.Server(), server)
	logger.Info("Beginning to serve requests")
	if err := grpcServer.Start(); err != nil {
		logger.Fatalf("Atomic Broadcast gRPC server has terminated while serving requests due to: %v", err)
	}
}

// Searches whether there is a join block for a system channel, and if there is, and it is a genesis block,
// initializes the ledger with it. Returns the join-block if it finds one.
func initSystemChannelWithJoinBlock(
	config *localconfig.TopLevel,
	cryptoProvider bccsp.BCCSP,
	lf blockledger.Factory,
) (bootstrapBlock *cb.Block) {
	if !config.ChannelParticipation.Enabled {
		return nil
	}

	joinBlockFileRepo, err := multichannel.InitJoinBlockFileRepo(config)
	if err != nil {
		logger.Panicf("Failed initializing join-block file repo: %v", err)
	}

	joinBlockFiles, err := joinBlockFileRepo.List()
	if err != nil {
		logger.Panicf("Failed listing join-block file repo: %v", err)
	}

	var systemChannelID string
	for _, fileName := range joinBlockFiles {
		channelName := joinBlockFileRepo.FileToBaseName(fileName)
		blockBytes, err := joinBlockFileRepo.Read(channelName)
		if err != nil {
			logger.Panicf("Failed reading join-block for channel '%s', error: %v", channelName, err)
		}
		block, err := protoutil.UnmarshalBlock(blockBytes)
		if err != nil {
			logger.Panicf("Failed unmarshalling join-block for channel '%s', error: %v", channelName, err)
		}
		if err = onboarding.ValidateBootstrapBlock(block, cryptoProvider); err == nil {
			bootstrapBlock = block
			systemChannelID = channelName
			break
		}
	}

	if bootstrapBlock == nil {
		logger.Debug("No join-block was found for the system channel")
		return nil
	}

	if bootstrapBlock.Header.Number == 0 {
		initializeBootstrapChannel(bootstrapBlock, lf)
	}

	logger.Infof("Join-block was found for the system channel: %s, number: %d", systemChannelID, bootstrapBlock.Header.Number)
	return bootstrapBlock

}



func loadLocalMSP(conf *localconfig.TopLevel) msp.MSP {
	// MUST call GetLocalMspConfig first, so that default BCCSP is properly
	// initialized prior to LoadByType.
	mspConfig, err := msp.GetLocalMspConfig(conf.General.LocalMSPDir, conf.General.BCCSP, conf.General.LocalMSPID)
	if err != nil {
		logger.Panicf("Failed to get local msp config: %v", err)
	}

	typ := msp.ProviderTypeToString(msp.FABRIC)
	opts, found := msp.Options[typ]
	if !found {
		logger.Panicf("MSP option for type %s is not found", typ)
	}

	localmsp, err := msp.New(opts, factory.GetDefault())
	if err != nil {
		logger.Panicf("Failed to load local MSP: %v", err)
	}

	if err = localmsp.Setup(mspConfig); err != nil {
		logger.Panicf("Failed to setup local msp with config: %v", err)
	}

	return localmsp
}

func initializeLogging() {
	loggingSpec := os.Getenv("FABRIC_LOGGING_SPEC")
	loggingFormat := os.Getenv("FABRIC_LOGGING_FORMAT")
	flogging.Init(flogging.Config{
		Format:  loggingFormat,
		Writer:  os.Stderr,
		LogSpec: loggingSpec,
	})
}
