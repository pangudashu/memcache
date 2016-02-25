package memcache

import (
	"sync"
)

//连接池
type ConnectionPool struct {
	pool     chan *Connection
	address  string
	maxCnt   int
	totalCnt int

	sync.Mutex
}

func open(address string, maxCnt int, initCnt int) (pool *ConnectionPool) {
	pool = &ConnectionPool{
		pool:    make(chan *Connection, maxCnt),
		address: address,
		maxCnt:  maxCnt,
	}

	for i := 0; i < initCnt; i++ {
		conn, err := connect(address)
		if err != nil {
			continue
		}
		pool.totalCnt++
		pool.pool <- conn
	}
	return pool
}

func (this *ConnectionPool) Get() (conn *Connection, err error) {
	select {
	case conn = <-this.pool:
		return conn, nil
	default:
	}

	this.Lock()

	if this.totalCnt >= this.maxCnt {
		//阻塞，直到有可用连接
		conn = <-this.pool
		this.Unlock()
		return conn, nil
	}

	//create new connect
	conn, err = connect(this.address)
	if err != nil {
		this.Unlock()
		return nil, err
	}
	this.totalCnt++
	this.Unlock()

	return conn, nil
}

func (this *ConnectionPool) Put(conn *Connection) {
	if conn == nil {
		return
	}

	this.pool <- conn
}

func (this *ConnectionPool) Release(conn *Connection) {
	conn.Close()
	this.Lock()
	this.totalCnt--
	this.Unlock()
}

//clear pool
func (this *ConnectionPool) Close() {
	for i := 0; i < len(this.pool); i++ {
		conn := <-this.pool
		conn.Close()
	}
}
