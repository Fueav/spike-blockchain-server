package main

import (
	logger "github.com/ipfs/go-log"
	"spike-blockchain-server/chain"
	"spike-blockchain-server/config"
	"spike-blockchain-server/constants"
	"spike-blockchain-server/server"
)

func main() {
	logger.SetLogLevel("*", "INFO")
	config.Init()
	bscClient, err := chain.NewBscListener(constants.MORALIS_SPEEDY_NODE, constants.GAME_VAULT_ADDRESS)
	if err != nil {
		//log
		return
	}
	bscClient.Run()

	r := server.NewRouter(bscClient)
	r.Run(":3000")
}
