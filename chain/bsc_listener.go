package chain

import (
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-redis/redis"
	logger "github.com/ipfs/go-log"
	"os"
	"spike-blockchain-server/cache"
	"spike-blockchain-server/game"
)

var log = logger.Logger("chain")

const (
	BNB_BLOCKNUM   = "bnb_blockNum"
	USDC_BLOCKNUM  = "usdc_blockNum"
	SKK_BLOCKNUM   = "skk_blockNum"
	SKS_BLOCKNUM   = "sks_blockNum"
	AUNFT_BLOCKNUM = "aunft_blockNum"
	BLOCK_NUM      = "blockNum"
)

type BscListener struct {
	network string
	manager *Manager
	ec      *ethclient.Client
	rc      *redis.Client
	l       map[TokenType]Listener
}

func NewBscListener(speedyNodeAddress string, targetWalletAddr string) (*BscListener, error) {
	log.Infof("bsc listener start")
	bl := &BscListener{}
	bl.manager = NewManager()

	client, err := ethclient.Dial(speedyNodeAddress)
	if err != nil {
		log.Error("eth client dial err : ", err)
		return nil, err
	}
	chainId, err := client.ChainID(context.Background())
	switch chainId.String() {
	case "56":
		bl.network = "bsc"
	case "97":
		bl.network = "bsc testnet"
	default:
		panic("not expected chainId")
	}

	bl.rc = cache.RedisClient
	bl.ec = client
	erc20Notify := make(chan ERC20Tx, 10)
	erc721Notify := make(chan ERC721Tx, 10)

	usdcChan := make(DataChannel, 10)
	skkChan := make(DataChannel, 10)
	sksChan := make(DataChannel, 10)
	aunftChan := make(DataChannel, 10)
	eb.Subscribe(newBlockTopic, usdcChan)
	eb.Subscribe(newBlockTopic, skkChan)
	eb.Subscribe(newBlockTopic, sksChan)
	eb.Subscribe(newBlockTopic, aunftChan)

	l := make(map[TokenType]Listener)
	l[BNB] = newBNBListener(newBNBTarget(targetWalletAddr), bl.ec, bl.rc, erc20Notify)
	l[USDC] = newERC20Listener(newUSDCTarget(targetWalletAddr), USDCContractAddress, USDC_BLOCKNUM, bl.ec, bl.rc, erc20Notify, usdcChan, getABI(USDCContractAbi))
	l[SKK] = newERC20Listener(newSKKTarget(targetWalletAddr), SKKContractAddress, SKK_BLOCKNUM, bl.ec, bl.rc, erc20Notify, skkChan, getABI(SKKContractAbi))
	l[SKS] = newERC20Listener(newSKSTarget(targetWalletAddr), SKSContractAddress, SKS_BLOCKNUM, bl.ec, bl.rc, erc20Notify, sksChan, getABI(SKSContractAbi))
	l[AUNFT] = newAUNFTListener(newAUNFTTarget(targetWalletAddr), AUNFTContractAddress, AUNFT_BLOCKNUM, bl.ec, bl.rc, erc721Notify, aunftChan, getABI(AUNFTAbi))
	bl.l = l
	spikeTxMgr := newSpikeTxMgr(game.NewKafkaClient(os.Getenv("KAFKA_ADDR")), erc20Notify, erc721Notify)
	go spikeTxMgr.run()
	return bl, nil
}

func (bl *BscListener) Run() {
	for _, listener := range bl.l {
		go func(l Listener) {
			l.run()
		}(listener)
	}

	if bl.rc.Get(BLOCK_NUM).Err() == redis.Nil {
		log.Infof("blockNum is not exist")
		return
	}
	nowBlockNum, err := bl.ec.BlockNumber(context.Background())
	if err != nil {
		log.Error("query now bnb_blockNum err :", err)
		return
	}
	cacheBlockNum, err := bl.rc.Get(BLOCK_NUM).Uint64()
	if err != nil {
		log.Error("query cache bnb_blockNum err : ", err)
		return
	}
	if cacheBlockNum < nowBlockNum-blockConfirmHeight {
		for _, listener := range bl.l {
			go func(l Listener) {
				l.handlePastBlock(cacheBlockNum, nowBlockNum-blockConfirmHeight)
			}(listener)
		}

	}
}
