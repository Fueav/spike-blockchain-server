package chain

type Listener interface {
	run()
	handlePastBlock(fromBlock, toBlock uint64)
}
