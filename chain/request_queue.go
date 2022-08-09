package chain

import "github.com/google/uuid"

type NftListReq struct {
	uuid       uuid.UUID
	walletAddr string
	network    string
}

type nftListQueue []*NftListReq

func (q *nftListQueue) Push(x *NftListReq) {
	item := x
	*q = append(*q, item)
}

func (q *nftListQueue) Remove(i int) *NftListReq {
	old := *q
	n := len(old)
	item := old[i]
	old[i] = old[n-1]
	old[n-1] = nil
	*q = old[0 : n-1]
	return item
}

func (q nftListQueue) Len() int { return len(q) }

type NativeTxReq struct {
	uuid       uuid.UUID
	walletAddr string
	blockNum   uint64
}

type nativeTxQueue []*NativeTxReq

func (q *nativeTxQueue) Push(x *NativeTxReq) {
	item := x
	*q = append(*q, item)
}

func (q *nativeTxQueue) Remove(i int) *NativeTxReq {
	old := *q
	n := len(old)
	item := old[i]
	old[i] = old[n-1]
	old[n-1] = nil
	*q = old[0 : n-1]
	return item
}

func (q nativeTxQueue) Len() int { return len(q) }

type ERC20TxReq struct {
	uuid         uuid.UUID
	walletAddr   string
	contractAddr string
	blockNum     uint64
}

type erc20TxQueue []*ERC20TxReq

func (q *erc20TxQueue) Push(x *ERC20TxReq) {
	item := x
	*q = append(*q, item)
}

func (q *erc20TxQueue) Remove(i int) *ERC20TxReq {
	old := *q
	n := len(old)
	item := old[i]
	old[i] = old[n-1]
	old[n-1] = nil
	*q = old[0 : n-1]
	return item
}

func (q erc20TxQueue) Len() int { return len(q) }
