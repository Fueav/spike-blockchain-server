package chain

import (
	"github.com/google/uuid"
	"sync"
	"time"
)

type result struct {
	r   interface{}
	err error
}

type NftListManager struct {
	counter Counter
	nlq     *nftListQueue
	workLk  sync.Mutex
	callRes map[uuid.UUID]chan result
	notify  chan struct{}
}

func NewNftListManager() *NftListManager {
	var counter Counter
	counter.Set(5, time.Second)
	m := &NftListManager{
		counter: counter,
		nlq:     &nftListQueue{},
		callRes: map[uuid.UUID]chan result{},
		notify:  make(chan struct{}),
	}
	go m.RunSched()
	return m
}

func (m *NftListManager) QueryNftList(uuid uuid.UUID, walletAddr, network string) {
	m.nlq.Push(&NftListReq{
		uuid:       uuid,
		walletAddr: walletAddr,
		network:    network,
	})
	m.notify <- struct{}{}
}

func (m *NftListManager) RunSched() {
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:

		case <-m.notify:

		}
		m.handle()
	}
}

func (m *NftListManager) handle() {
	queueLen := m.nlq.Len()
	for i := 0; i < queueLen; i++ {
		if !m.counter.Allow() {
			continue
		}
		go func(sqi int) {
			task := (*m.nlq)[sqi]
			m.nlq.Remove(sqi)
			m.queryNftListByMoralis(task.uuid, task.walletAddr, task.network)
		}(i)
	}
}

func (m *NftListManager) WaitCall(uuid uuid.UUID) (interface{}, error) {
	defer func() {
		m.workLk.Lock()
		defer m.workLk.Unlock()
		delete(m.callRes, uuid)
	}()

	ch, ok := m.callRes[uuid]
	if !ok {
		ch = make(chan result, 1)
		m.workLk.Lock()
		m.callRes[uuid] = ch
		m.workLk.Unlock()
	}

	select {
	case res := <-ch:
		return res.r, res.err
	}
}

func (m *NftListManager) queryNftListByMoralis(uuid uuid.UUID, walletAddr, network string) {
	var res []NftResult
	res = QueryWalletNft("", walletAddr, network, res)
	log.Info("---", len(res))
	m.workLk.Lock()
	defer m.workLk.Unlock()
	m.callRes[uuid] <- result{
		r:   res,
		err: nil,
	}
}

type NativeTxManager struct {
	counter Counter
	ntq     *nativeTxQueue
	workLk  sync.Mutex
	callRes map[uuid.UUID]chan result
	notify  chan struct{}
}

func NewNativeTxManager() *NativeTxManager {
	var counter Counter
	counter.Set(5, time.Second)
	m := &NativeTxManager{
		counter: counter,
		ntq:     &nativeTxQueue{},
		callRes: map[uuid.UUID]chan result{},
		notify:  make(chan struct{}),
	}
	go m.RunSched()
	return m
}

func (m *NativeTxManager) QueryNativeTxRecord(uuid uuid.UUID, walletAddr string, blockNum uint64) {
	m.ntq.Push(&NativeTxReq{
		uuid:       uuid,
		walletAddr: walletAddr,
		blockNum:   blockNum,
	})
	m.notify <- struct{}{}
}

func (m *NativeTxManager) RunSched() {
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:

		case <-m.notify:

		}
		m.handle()
	}
}

func (m *NativeTxManager) handle() {
	queueLen := m.ntq.Len()
	for i := 0; i < queueLen; i++ {
		if !m.counter.Allow() {
			continue
		}
		go func(sqi int) {
			task := (*m.ntq)[sqi]
			m.ntq.Remove(sqi)
			m.queryNativeTxRecordByBscScan(task.uuid, task.walletAddr, task.blockNum)
		}(i)
	}
}

func (m *NativeTxManager) WaitCall(uuid uuid.UUID) (interface{}, error) {
	defer func() {
		m.workLk.Lock()
		defer m.workLk.Unlock()
		delete(m.callRes, uuid)
	}()

	ch, ok := m.callRes[uuid]
	if !ok {
		ch = make(chan result, 1)
		m.workLk.Lock()
		m.callRes[uuid] = ch
		m.workLk.Unlock()
	}

	select {
	case res := <-ch:
		return res.r, res.err
	}
}

func (m *NativeTxManager) queryNativeTxRecordByBscScan(uuid uuid.UUID, walletAddr string, blockNum uint64) {
	res, err := queryNativeTxRecord(walletAddr, blockNum)
	m.workLk.Lock()
	defer m.workLk.Unlock()
	m.callRes[uuid] <- result{
		r:   res,
		err: err,
	}
}

type ERC20TxManager struct {
	counter Counter
	etq     *erc20TxQueue
	workLk  sync.Mutex
	callRes map[uuid.UUID]chan result
	notify  chan struct{}
}

func NewERC20TxManager() *ERC20TxManager {
	var counter Counter
	counter.Set(5, time.Second)
	m := &ERC20TxManager{
		counter: counter,
		etq:     &erc20TxQueue{},
		callRes: map[uuid.UUID]chan result{},
		notify:  make(chan struct{}),
	}
	go m.RunSched()
	return m
}

func (m *ERC20TxManager) QueryERC20TxRecord(uuid uuid.UUID, contractAddr, walletAddr string, blockNum uint64) {
	m.etq.Push(&ERC20TxReq{
		uuid:         uuid,
		walletAddr:   walletAddr,
		contractAddr: contractAddr,
		blockNum:     blockNum,
	})
	m.notify <- struct{}{}
}

func (m *ERC20TxManager) RunSched() {
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:

		case <-m.notify:

		}
		m.handle()
	}
}

func (m *ERC20TxManager) handle() {
	queueLen := m.etq.Len()
	for i := 0; i < queueLen; i++ {
		if !m.counter.Allow() {
			continue
		}
		go func(sqi int) {
			task := (*m.etq)[sqi]
			m.etq.Remove(sqi)
			m.queryErc20TxRecordByBscScan(task.uuid, task.contractAddr, task.walletAddr, task.blockNum)
		}(i)
	}
}

func (m *ERC20TxManager) WaitCall(uuid uuid.UUID) (interface{}, error) {
	defer func() {
		m.workLk.Lock()
		defer m.workLk.Unlock()
		delete(m.callRes, uuid)
	}()

	ch, ok := m.callRes[uuid]
	if !ok {
		ch = make(chan result, 1)
		m.workLk.Lock()
		m.callRes[uuid] = ch
		m.workLk.Unlock()
	}

	select {
	case res := <-ch:
		return res.r, res.err
	}
}

func (m *ERC20TxManager) queryErc20TxRecordByBscScan(uuid uuid.UUID, contractAddr, walletAddr string, blockNum uint64) {
	res, err := queryERC20TxRecord(contractAddr, walletAddr, blockNum)
	m.workLk.Lock()
	defer m.workLk.Unlock()
	m.callRes[uuid] <- result{
		r:   res,
		err: err,
	}
}
