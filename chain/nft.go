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

const emptyAddress = "0x0000000000000000000000000000000000000000"

type AUNFTTarget struct {
	txAddress string
}

func newAUNFTTarget(address string) *AUNFTTarget {
	return &AUNFTTarget{
		txAddress: address,
	}
}

func (t *AUNFTTarget) Accept(fromAddr, toAddr string) (bool, uint64) {
	if strings.ToLower(emptyAddress) == strings.ToLower(fromAddr) {
		return true, AUNFT_TRANSFER
	}

	if strings.ToLower(t.txAddress) == strings.ToLower(toAddr) {
		return true, AUNFT_IMPORT
	}
	return true, AUNFT_TRANSFER
}

type AUNFTListener struct {
	TxFilter
	contractAddr   string
	cacheBlockNum  string
	erc721Notify   chan ERC721Tx
	newBlockNotify DataChannel
	ec             *ethclient.Client
	rc             *redis.Client
	abi            abi.ABI
}

func newAUNFTListener(filter TxFilter, contractAddr string, cacheBlockNum string, ec *ethclient.Client, rc *redis.Client, erc721Notify chan ERC721Tx, newBlockNotify DataChannel, abi abi.ABI) *AUNFTListener {
	return &AUNFTListener{
		filter,
		contractAddr,
		cacheBlockNum,
		erc721Notify,
		newBlockNotify,
		ec,
		rc,
		abi,
	}
}

func (al *AUNFTListener) run() {
	go al.NewEventFilter()
}

func (al *AUNFTListener) handlePastBlock(fromBlock, toBlock uint64) {
	go al.PastEventFilter(fromBlock, toBlock)
}

func (al *AUNFTListener) NewEventFilter() error {
	for {
		select {
		case de := <-al.newBlockNotify:
			height := de.Data.(*big.Int).Uint64()
			al.PastEventFilter(height, height)
		}
	}
}

func (al *AUNFTListener) PastEventFilter(fromBlockNum, toBlockNum uint64) error {
	log.Infof("aunft past event filter, fromBlock : %d, toBlock : %d ", fromBlockNum, toBlockNum)
	ethClient := al.ec
	contractAddress := common.HexToAddress(al.contractAddr)

	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		FromBlock: big.NewInt(int64(fromBlockNum)),
		ToBlock:   big.NewInt(int64(toBlockNum)),
	}

	sub, err := ethClient.FilterLogs(context.Background(), query)
	if err != nil {
		log.Error("nft subscribe event log, err : ", err)
		return err
	}
	for _, l := range sub {
		switch l.Topics[0].String() {
		case EventSignHash(TransferTopic):
			var status uint64
			recp, err := al.ec.TransactionReceipt(context.Background(), l.TxHash)
			status = recp.Status
			if err != nil {
				log.Error("nft TransactionReceipt err : ", err)
				status = 0
			}
			block, err := al.ec.BlockByNumber(context.Background(), big.NewInt(int64(l.BlockNumber)))
			if err != nil {
				status = 0
			}

			fromAddr := common.HexToAddress(l.Topics[1].Hex()).String()
			toAddr := common.HexToAddress(l.Topics[2].Hex()).String()
			_, txType := al.Accept(fromAddr, toAddr)
			al.erc721Notify <- ERC721Tx{
				From:    fromAddr,
				To:      toAddr,
				TxType:  txType,
				TxHash:  l.TxHash.Hex(),
				Status:  status,
				PayTime: int64(block.Time() * 1000),
				TokenId: l.Topics[3].Big().Uint64(),
			}
		}
	}
	return nil
}
