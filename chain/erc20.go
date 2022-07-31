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

type USDCTarget struct {
	txAddress string
}

func newUSDCTarget(address string) *USDCTarget {
	return &USDCTarget{
		txAddress: address,
	}
}

func (t *USDCTarget) Accept(fromAddr, toAddr string) (bool, uint64) {
	if strings.ToLower(t.txAddress) == strings.ToLower(toAddr) {
		return true, USDC_RECHARGE
	}

	if strings.ToLower(t.txAddress) == strings.ToLower(fromAddr) {
		return true, USDC_WITHDRAW
	}

	return false, NOT_EXIST
}

type SKSTarget struct {
	txAddress string
}

func newSKSTarget(address string) *SKSTarget {
	return &SKSTarget{
		txAddress: address,
	}
}

func (t *SKSTarget) Accept(fromAddr, toAddr string) (bool, uint64) {
	if strings.ToLower(t.txAddress) == strings.ToLower(toAddr) {
		return true, SKS_RECHARGE
	}

	if strings.ToLower(t.txAddress) == strings.ToLower(fromAddr) {
		return true, SKS_WITHDRAW
	}

	return false, NOT_EXIST
}

type SKKTarget struct {
	txAddress string
}

func newSKKTarget(address string) *SKKTarget {
	return &SKKTarget{
		txAddress: address,
	}
}

func (t *SKKTarget) Accept(fromAddr, toAddr string) (bool, uint64) {
	if strings.ToLower(t.txAddress) == strings.ToLower(toAddr) {
		return true, SKK_RECHARGE
	}

	if strings.ToLower(t.txAddress) == strings.ToLower(fromAddr) {
		return true, SKK_WITHDRAW
	}

	return false, NOT_EXIST
}

type ERC20Listener struct {
	TxFilter
	contractAddr   string
	cacheBlockNum  string
	erc20Notify    chan ERC20Tx
	newBlockNotify DataChannel
	ec             *ethclient.Client
	rc             *redis.Client
	abi            abi.ABI
}

func newERC20Listener(filter TxFilter, contractAddr string, cacheBlockNum string, ec *ethclient.Client, rc *redis.Client, erc20Notify chan ERC20Tx, newBlockNotify DataChannel, abi abi.ABI) *ERC20Listener {
	return &ERC20Listener{
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

func (el *ERC20Listener) run() {
	go el.NewEventFilter(el.contractAddr)
}

func (el *ERC20Listener) handlePastBlock(fromBlock, toBlock uint64) {
	go el.PastEventFilter(el.contractAddr, fromBlock, toBlock)
}

func (el *ERC20Listener) NewEventFilter(contractAddr string) error {
	for {
		select {
		case de := <-el.newBlockNotify:
			height := de.Data.(*big.Int).Uint64()
			el.PastEventFilter(contractAddr, height, height)
		}
	}
}

func (el *ERC20Listener) PastEventFilter(contractAddr string, fromBlockNum, toBlockNum uint64) error {
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
		case eventSignHash(TransferTopic):
			intr, err := el.abi.Events["Transfer"].Inputs.Unpack(logEvent.Data)
			if err != nil {
				log.Error("erc20 data unpack err : ", err)
				break
			}
			fromAddr := common.HexToAddress(logEvent.Topics[1].Hex()).String()
			toAddr := common.HexToAddress(logEvent.Topics[2].Hex()).String()
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
				Amount:  intr[0].(*big.Int).String(),
			}
		}
	}
	return err
}
