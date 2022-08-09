package chain

import (
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-redis/redis"
	logger "github.com/ipfs/go-log"
	"spike-blockchain-server/cache"
	"spike-blockchain-server/constants"
	"spike-blockchain-server/game"
)

var log = logger.Logger("chain")

const (
	BNB_BLOCKNUM        = "bnb_blockNum"
	GAME_VAULT_BLOCKNUM = "vault_blockNum"
	SKK_BLOCKNUM        = "skk_blockNum"
	SKS_BLOCKNUM        = "sks_blockNum"
	AUNFT_BLOCKNUM      = "aunft_blockNum"
	BLOCK_NUM           = "blockNum"
)

type BscListener struct {
	network   string
	nlManager *NftListManager
	ntManager *NativeTxManager
	etManager *ERC20TxManager
	ec        *ethclient.Client
	rc        *redis.Client
	l         map[TokenType]Listener
}

func NewBscListener(speedyNodeAddress string, targetWalletAddr string) (*BscListener, error) {
	log.Infof("bsc listener start")
	bl := &BscListener{}
	bl.nlManager = NewNftListManager()
	bl.ntManager = NewNativeTxManager()
	bl.etManager = NewERC20TxManager()

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

	vaultChan := make(DataChannel, 10)
	skkChan := make(DataChannel, 10)
	sksChan := make(DataChannel, 10)
	aunftChan := make(DataChannel, 10)
	eb.Subscribe(newBlockTopic, vaultChan)
	eb.Subscribe(newBlockTopic, skkChan)
	eb.Subscribe(newBlockTopic, sksChan)
	eb.Subscribe(newBlockTopic, aunftChan)

	l := make(map[TokenType]Listener)
	l[BNB] = newBNBListener(newBNBTarget(targetWalletAddr), bl.ec, bl.rc, erc20Notify)
	l[gameVault] = newGameVaultListener(newGameVaultTarget(targetWalletAddr), constants.GAME_VAULT_ADDRESS, GAME_VAULT_BLOCKNUM, bl.ec, bl.rc, erc20Notify, vaultChan, getABI(GameVaultABI))
	l[governanceToken] = newERC20Listener(newSKKTarget(targetWalletAddr), constants.GOVERNANCE_TOKEN_ADDRESS, SKK_BLOCKNUM, bl.ec, bl.rc, erc20Notify, skkChan, getABI(GovernanceTokenABI))
	l[gameToken] = newERC20Listener(newSKSTarget(targetWalletAddr), constants.GAME_TOKEN_ADDRESS, SKS_BLOCKNUM, bl.ec, bl.rc, erc20Notify, sksChan, getABI(GameTokenABI))
	l[gameNft] = newAUNFTListener(newAUNFTTarget(targetWalletAddr), constants.GAME_NFT_ADDRESS, AUNFT_BLOCKNUM, bl.ec, bl.rc, erc721Notify, aunftChan, getABI(GameNftABI))
	bl.l = l
	spikeTxMgr := newSpikeTxMgr(game.NewKafkaClient(constants.KAFKA_ADDR), erc20Notify, erc721Notify)
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
