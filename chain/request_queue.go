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
