package memcache

import (
	"sync"
	"time"
)

//连接池
type ConnectionPool struct {
	pool     chan *Connection
	address  string
	maxCnt   int
	totalCnt int
	idleTime time.Duration

	sync.Mutex
}

func open(address string, maxCnt int, initCnt int, idelTime time.Duration) (pool *ConnectionPool) {
	pool = &ConnectionPool{
		pool:     make(chan *Connection, maxCnt),
		address:  address,
		maxCnt:   maxCnt,
		idleTime: idelTime,
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
	for {
		conn, err = this.get()

		if err != nil {
			return nil, err
		}

		if conn.lastActiveTime.Add(this.idleTime).UnixNano() > time.Now().UnixNano() {
			break
		} else {
			this.Release(conn)
		}
	}
	conn.lastActiveTime = time.Now()
	return conn, err
}

func (this *ConnectionPool) get() (conn *Connection, err error) {
	select {
	case conn = <-this.pool:
		return conn, nil
	default:
	}

	this.Lock()
	defer this.Unlock()

	if this.totalCnt >= this.maxCnt {
		//阻塞，直到有可用连接
		conn = <-this.pool
		return conn, nil
	}

	//create new connect
	conn, err = connect(this.address)
	if err != nil {
		return nil, err
	}
	this.totalCnt++

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
