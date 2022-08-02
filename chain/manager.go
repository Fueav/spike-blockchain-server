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

type Manager struct {
	counter Counter
	nlq     *nftListQueue
	workLk  sync.Mutex
	callRes map[uuid.UUID]chan result
	notify  chan struct{}
}

func NewManager() *Manager {
	var counter Counter
	counter.Set(5, time.Second)
	m := &Manager{
		counter: counter,
		nlq:     &nftListQueue{},
		callRes: map[uuid.UUID]chan result{},
		notify:  make(chan struct{}),
	}
	go m.RunSched()
	return m
}

func (m *Manager) QueryNftList(uuid uuid.UUID, walletAddr, network string) {
	m.nlq.Push(&NftListReq{
		uuid:       uuid,
		walletAddr: walletAddr,
		network:    network,
	})
	m.notify <- struct{}{}
}

func (m *Manager) RunSched() {
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:

		case <-m.notify:

		}
		m.handle()
	}
}

func (m *Manager) handle() {
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

func (m *Manager) WaitCall(uuid uuid.UUID) (interface{}, error) {
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

func (m *Manager) queryNftListByMoralis(uuid uuid.UUID, walletAddr, network string) {
	var res []NftResult
	res = QueryWalletNft("", walletAddr, network, res)
	m.workLk.Lock()
	defer m.workLk.Unlock()
	m.callRes[uuid] <- result{
		r:   res,
		err: nil,
	}
}
