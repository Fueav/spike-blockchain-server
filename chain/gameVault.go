package chain

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-redis/redis"
	"math/big"
	"strings"
)

type GameVaultTarget struct {
	txAddress string
}

func newGameVaultTarget(address string) *GameVaultTarget {
	return &GameVaultTarget{
		txAddress: address,
	}
}

func (t *GameVaultTarget) Accept(fromAddr, toAddr string) (bool, uint64) {
	if strings.ToLower(t.txAddress) == strings.ToLower(fromAddr) {
		return true, BNB_WITHDRAW
	}
	return false, NOT_EXIST
}

type GameVaultListener struct {
	TxFilter
	contractAddr   string
	cacheBlockNum  string
	erc20Notify    chan ERC20Tx
	newBlockNotify DataChannel
	ec             *ethclient.Client
	rc             *redis.Client
	abi            abi.ABI
}

func newGameVaultListener(filter TxFilter, contractAddr string, cacheBlockNum string, ec *ethclient.Client, rc *redis.Client, erc20Notify chan ERC20Tx, newBlockNotify DataChannel, abi abi.ABI) *GameVaultListener {
	return &GameVaultListener{
		filter,
		contractAddr,
		cacheBlockNum,
		erc20Notify,
		newBlockNotify,
		ec,
		rc,
		abi,
	}
}

func (el *GameVaultListener) run() {
	go el.NewEventFilter(el.contractAddr)
}

func (el *GameVaultListener) handlePastBlock(fromBlock, toBlock uint64) {
	go el.PastEventFilter(el.contractAddr, fromBlock, toBlock)
}

func (el *GameVaultListener) NewEventFilter(contractAddr string) error {
	for {
		select {
		case de := <-el.newBlockNotify:
			height := de.Data.(*big.Int).Uint64()
			el.PastEventFilter(contractAddr, height, height)
		}
	}
}

func (el *GameVaultListener) PastEventFilter(contractAddr string, fromBlockNum, toBlockNum uint64) error {
	log.Infof("erc20 past event filter, type : %s, fromBlock : %d, toBlock : %d ", el.cacheBlockNum, fromBlockNum, toBlockNum)
	ethClient := el.ec
	contractAddress := common.HexToAddress(contractAddr)

	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		FromBlock: big.NewInt(int64(fromBlockNum)),
		ToBlock:   big.NewInt(int64(toBlockNum)),
	}

	sub, err := ethClient.FilterLogs(context.Background(), query)
	if err != nil {
		log.Error("erc20 subscribe err : ", err)
		return err
	}
	for _, logEvent := range sub {
		switch logEvent.Topics[0].String() {
		case EventSignHash(WITHRAWALTOPIC):
			intr, err := el.abi.Events["Withdraw"].Inputs.Unpack(logEvent.Data)
			if err != nil {
				log.Error("game vault data unpack err : ", err)
				break
			}
			if intr[0].(common.Address).String() != emptyAddress {
				continue
			}
			fromAddr := intr[1].(common.Address).String()
			toAddr := intr[2].(common.Address).String()
			accept, txType := el.Accept(fromAddr, toAddr)
			if !accept {
				continue
			}
			var status uint64
			recp, err := el.ec.TransactionReceipt(context.Background(), logEvent.TxHash)
			status = recp.Status
			if err != nil {
				status = 0
			}
			block, err := el.ec.BlockByNumber(context.Background(), big.NewInt(int64(logEvent.BlockNumber)))
			if err != nil {
				status = 0
			}
			el.erc20Notify <- ERC20Tx{
				From:    fromAddr,
				To:      toAddr,
				TxType:  txType,
				TxHash:  logEvent.TxHash.Hex(),
				Status:  status,
				PayTime: int64(block.Time() * 1000),
				Amount:  intr[3].(*big.Int).String(),
			}
		}
	}
	return err
}
