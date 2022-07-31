package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"golang.org/x/xerrors"
	"math"
	"math/big"
	"os"
	"sort"
	"spike-blockchain-server/chain/contract"
	"spike-blockchain-server/constants"
	"spike-blockchain-server/serializer"
	"strconv"
	"strings"
	"time"
)

const (
	duration      = 10 * time.Minute
	nftTypeSuffix = "_nftType"
)

type NftType struct {
	Name   string `json:"name"`
	Amount int    `json:"amount"`
}

type CacheData struct {
	Type        string                 `json:"type"`
	GameId      string                 `json:"gameId"`
	BlockNumber string                 `json:"block_number"`
	TokenId     string                 `json:"token_id"`
	Description string                 `json:"description"`
	SpikeInfo   SpikeInfo              `json:"spike_info"`
	Attributes  map[string]interface{} `json:"attributes"`
}
type CacheDataList struct {
	CD []CacheData `json:"cache_data"`
}

type txQueryService struct {
	TxHash string `json:"txHash"`
}

type metadataService struct {
	TokenId string `json:"tokenId"`
	Address string `json:"address"`
}

type nftTypeService struct {
	WalletAddress string `json:"walletAddress"`
	NetWork       string `json:"network"`
}

type nftMetadataService struct {
	WalletAddress string `json:"walletAddress"`
	NetWork       string `json:"network"`
	Type          string `json:"type"`
	Page          int    `json:"page"`
	PageSize      int    `json:"page_size"`
}

type Metadata struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Image       string    `json:"image"`
	ExternalUrl string    `json:"external_url"`
	SpikeInfo   SpikeInfo `json:"spike_info"`
	Attribute   []Attr    `json:"attributes"`
}

type SpikeInfo struct {
	Version string `json:"version"`
	Tp      string `json:"type"`
	Avatar  string `json:"avatar"`
}

type Attr struct {
	TraitType string      `json:"trait_type"`
	Value     interface{} `json:"value"`
}

type NftResults struct {
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Cursor   string      `json:"cursor"`
	Results  []NftResult `json:"result"`
}

type NftResult struct {
	TokenId     string `json:"token_id"`
	BlockNumber string `json:"block_number"`
	TokenUri    string `json:"token_uri"`
	Metadata    string `json:"metadata"`
}

func (bl *BscListener) QueryTxIsPendingByHash(c *gin.Context) {
	var service txQueryService
	if err := c.ShouldBind(&service); err == nil {
		log.Infof("tx: %s", service.TxHash)
		res := bl.queryTxIsPendingByHash(service.TxHash)
		c.JSON(200, res)
	} else {
		c.JSON(500, err.Error())
	}
}

func (bl *BscListener) QueryTxStatusByHash(c *gin.Context) {
	var service txQueryService
	if err := c.ShouldBind(&service); err == nil {
		log.Infof("tx: %s", service.TxHash)
		res := bl.queryTxStatusByHash(service.TxHash)
		c.JSON(200, res)
	} else {
		c.JSON(500, err.Error())
	}
}

func (bl *BscListener) queryTxIsPendingByHash(txHash string) serializer.Response {
	_, isPending, err := bl.ec.TransactionByHash(context.Background(), common.HexToHash(txHash))
	code := 200
	if err != nil {
		code = 500
		return serializer.Response{
			Code:  code,
			Data:  isPending,
			Error: err.Error(),
		}
	}
	return serializer.Response{
		Code: code,
		Data: isPending,
	}
}

func (bl *BscListener) queryTxStatusByHash(txHash string) serializer.Response {
	receipt, err := bl.ec.TransactionReceipt(context.Background(), common.HexToHash(txHash))
	code := 200
	if err != nil {
		code = 500
		return serializer.Response{
			Code:  code,
			Error: err.Error(),
		}
	}
	return serializer.Response{
		Code: code,
		Data: receipt.Status,
	}
}

func (bl *BscListener) QueryNftMetadata(c *gin.Context) {
	var service metadataService

	if err := c.ShouldBind(&service); err == nil {
		tokenId, err := strconv.Atoi(service.TokenId)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		res := bl.queryNftMetadata(int64(tokenId), service.Address)
		c.JSON(200, res)
	} else {
		c.JSON(500, err.Error())
	}
}

func (bl *BscListener) queryNftMetadata(tokenId int64, address string) serializer.Response {
	aunft, err := contract.NewAunft(common.HexToAddress(AUNFTContractAddress), bl.ec)
	if err != nil {
		log.Error("new auNft err : ", err)
		return serializer.Response{
			Code:  500,
			Error: err.Error(),
		}
	}
	uri, err := aunft.TokenURI(nil, big.NewInt(tokenId))
	if err != nil {
		log.Error("query tokenUri err : ", err)
		return serializer.Response{
			Code:  500,
			Error: err.Error(),
		}
	}
	owner, err := aunft.OwnerOf(nil, big.NewInt(tokenId))
	if err != nil {
		return serializer.Response{
			Code:  500,
			Error: err.Error(),
		}
	}
	if strings.ToLower(owner.String()) != strings.ToLower(address) {
		return serializer.Response{
			Code:  500,
			Error: xerrors.New("tokenId, nft not match").Error(),
		}
	}

	client := resty.New()

	resp, err := client.R().Get(uri)
	log.Infof("query nft tokenId : %d, uri : %s", tokenId, uri)
	if err != nil {
		return serializer.Response{
			Code:  500,
			Error: err.Error(),
		}
	}

	if resp.IsError() {
		return serializer.Response{
			Code:  500,
			Error: resp.String(),
		}
	}
	var m Metadata
	err = json.Unmarshal(resp.Body(), &m)
	if err != nil {
		return serializer.Response{
			Code:  500,
			Error: err.Error(),
		}
	}
	metadata, err := json.Marshal(m)
	if err != nil {
		return serializer.Response{
			Code:  500,
			Error: err.Error(),
		}
	}
	log.Infof(string(metadata))
	return serializer.Response{
		Code: 200,
		Data: string(metadata),
	}
}

func (bl *BscListener) QueryNftListByType(c *gin.Context) {
	var service nftMetadataService

	if err := c.ShouldBind(&service); err == nil {
		if service.NetWork == "" || service.WalletAddress == "" || service.Type == "" {
			if err != nil {
				c.JSON(500, xerrors.New("param can not be null").Error())
				return
			}
		}
		res := bl.queryNftListByType(service.WalletAddress, service.NetWork, service.Type, int64(service.Page), int64(service.PageSize))
		c.JSON(200, res)
	} else {
		c.JSON(500, err.Error())
	}
}

func (bl *BscListener) queryNftListByType(addr, network, tp string, page, pageSize int64) serializer.Response {
	log.Infof("page : %d, pageSize : %d", page, pageSize)
	if result := bl.GetJson(network + addr + tp); result == "" {
		_, err := bl.queryNftFromMoralis(addr, network)
		if err != nil {
			return serializer.Response{
				Code:  500,
				Error: err.Error(),
			}
		}
	}
	nftjson := bl.GetJson(network + addr + tp)
	var cdList CacheDataList
	err := json.Unmarshal([]byte(nftjson), &cdList)
	dataList := cdList.CD
	sort.Slice(dataList, func(i, j int) bool {
		return dataList[i].BlockNumber < dataList[j].BlockNumber
	})
	start, end := SlicePage(page, pageSize, int64(len(dataList)))
	dataPage := dataList[start:end]
	if err != nil {
		return serializer.Response{
			Code: 500,
			Data: err.Error(),
		}
	}
	return serializer.Response{
		Code: 200,
		Data: dataPage,
	}
}

func (bl *BscListener) QueryWalletAddrNft(c *gin.Context) {
	var service nftTypeService

	if err := c.ShouldBind(&service); err == nil {
		if service.NetWork == "" || service.WalletAddress == "" {
			if err != nil {
				c.JSON(500, xerrors.New("param can not be null").Error())
				return
			}
		}
		res := bl.queryWalletAddrNft(service.WalletAddress, service.NetWork)
		c.JSON(200, res)
	} else {
		c.JSON(500, err.Error())
	}
}

func (bl *BscListener) queryWalletAddrNft(addr string, network string) serializer.Response {
	if t := bl.GetJson(network + addr + nftTypeSuffix); t != "" {
		var nftType []NftType
		err := json.Unmarshal([]byte(t), &nftType)
		if err != nil {
			return serializer.Response{
				Code:  500,
				Error: err.Error(),
			}
		}
		return serializer.Response{
			Code: 200,
			Data: nftType,
		}
	}
	nftType, err := bl.queryNftFromMoralis(addr, network)
	if err != nil {
		return serializer.Response{
			Code:  500,
			Error: err.Error(),
		}
	}
	return serializer.Response{
		Code: 200,
		Data: nftType,
	}
}

func (bl *BscListener) queryNftFromMoralis(addr string, network string) ([]NftType, error) {
	var nr []NftResult
	nr = queryWalletNft("", addr, network, nr)
	nr = bl.convertNftResult(nr)
	dataList := parseMetadata(nr)
	dataMap := parseCacheData(dataList)
	var nftType []NftType
	for k, _ := range dataMap {
		nftType = append(nftType, NftType{
			Name:   k,
			Amount: len(dataMap[k]),
		})
		var cdList CacheDataList
		cdList.CD = dataMap[k]
		cacheByte, err := json.Marshal(cdList)
		if err != nil {
			break
		}
		bl.SetJson(network+addr+k, string(cacheByte), duration)
	}
	nftTypeByte, err := json.Marshal(nftType)
	if err != nil {
		return nil, err
	}
	bl.SetJson(network+addr+nftTypeSuffix, string(nftTypeByte), duration)
	return nftType, err
}

func (bl *BscListener) SetJson(key string, value string, duration time.Duration) {
	bl.rc.Set(key, value, duration)
}

func (bl *BscListener) GetJson(key string) string {
	if bl.rc.Get(key).Err() == redis.Nil {
		return ""
	}
	return bl.rc.Get(key).Val()
}

func (bl *BscListener) convertNftResult(res []NftResult) []NftResult {
	aunft, err := contract.NewAunft(common.HexToAddress(AUNFTContractAddress), bl.ec)
	if err != nil {
		log.Error("new auNft err : ", err)
		return res
	}
	for k, v := range res {
		if v.TokenUri == "" {
			tokenId, err := strconv.Atoi(v.TokenId)
			if err != nil {
				log.Errorf("string %s convert int err : %v", v.TokenId, err)
				break
			}
			uri, err := aunft.TokenURI(nil, big.NewInt(int64(tokenId)))
			if err != nil {
				log.Error("query tokenUri err : ", err)
				break
			}
			client := resty.New()
			resp, err := client.R().Get(uri)
			log.Infof("query nft tokenId : %d, uri : %s", tokenId, uri)
			var m Metadata
			err = json.Unmarshal(resp.Body(), &m)
			if err != nil {
				break
			}
			metadata, err := json.Marshal(m)
			if err != nil {
				break
			}
			res[k].TokenUri = uri
			res[k].Metadata = string(metadata)
		}
	}
	return res
}

func parseCacheData(cds []CacheData) map[string][]CacheData {
	dataMap := make(map[string][]CacheData)
	for _, v := range cds {
		if _, have := dataMap[v.Type]; have {
			dataMap[v.Type] = append(dataMap[v.Type], v)
		} else {
			var cd []CacheData
			cd = append(cd, v)
			dataMap[v.Type] = cd
		}
	}
	return dataMap
}

func parseMetadata(nr []NftResult) []CacheData {
	var dataList []CacheData
	for _, v := range nr {
		var cd CacheData
		cd.TokenId = v.TokenId
		cd.BlockNumber = v.BlockNumber
		var m Metadata
		err := json.Unmarshal([]byte(v.Metadata), &m)
		if err != nil {
			log.Error("json unmarshal err : ", err)
			break
		}
		split := strings.Split(m.Name, " ")
		if len(split) != 2 {
			log.Error("pass------")
			break
		}
		cd.Type = split[0]
		cd.GameId = split[1]
		cd.Description = m.Description
		cd.SpikeInfo = m.SpikeInfo
		attrMap := make(map[string]interface{})
		for _, attr := range m.Attribute {
			attrMap[attr.TraitType] = attr.Value
		}
		cd.Attributes = attrMap
		dataList = append(dataList, cd)
	}
	return dataList
}

func queryWalletNft(cursor, walletAddr, network string, res []NftResult) []NftResult {
	client := resty.New()
	resp, _ := client.R().
		SetHeader("Accept", "application/json").
		SetHeader("x-api-key", os.Getenv("MORALIS_KEY")).
		Get(getUrl(AUNFTContractAddress, walletAddr, network, cursor))

	var nrs NftResults
	err := json.Unmarshal(resp.Body(), &nrs)
	if err != nil {
		log.Error("json unmarshal err : ", err)
		return res
	}
	res = append(res, nrs.Results...)
	if nrs.Page*nrs.PageSize >= nrs.Total {
		return res
	}
	res = queryWalletNft(nrs.Cursor, walletAddr, network, res)
	return res
}

func getUrl(contractAddr, walletAddr, network, cursor string) string {
	return fmt.Sprintf("%s%s/nft/%s?chain=%s&cursor=%s", constants.MORALIS_API, walletAddr, contractAddr, network, cursor)
}

func SlicePage(page, pageSize, nums int64) (sliceStart, sliceEnd int64) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > nums {
		return 0, nums
	}
	pageCount := int64(math.Ceil(float64(nums) / float64(pageSize)))
	if page > pageCount {
		return 0, 0
	}
	sliceStart = (page - 1) * pageSize
	sliceEnd = sliceStart + pageSize

	if sliceEnd > nums {
		sliceEnd = nums
	}
	return sliceStart, sliceEnd
}
