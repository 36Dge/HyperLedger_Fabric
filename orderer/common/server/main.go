package server

import (
	"fmt"
	"os"
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

	conf,err := localcofig.Load()
	if err != nil{
		logger.Error("failed to parse config:",err)
		os.Exit(1)
	}

	initializeLogging()

	prettyPrintStruct(conf)






}