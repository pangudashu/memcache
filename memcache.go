package memcache

import (
	"errors"
	"sync"
	"time"
)

type Memcache struct {
	nodes   *Nodes
	manager *serverManager

	sync.RWMutex //保证操作nodes的原子性
}

type serverManager struct {
	serverList      []*Server
	badServerNotice chan bool
	isRmBadServer   bool
}

var (
	badTryCnt       = 4
	defaultMaxConn  = 128
	defaultInitConn = 8
	defaultIdleTime = time.Hour * 2
)

func NewMemcache(server_list []*Server) (mem *Memcache, err error) { /*{{{*/
	if server_list == nil {
		return nil, errors.New("Server is nil or address is empty")
	}

	mem = &Memcache{}

	//create connect pool
	for _, server := range server_list {
		if server == nil || server.Address == "" {
			return nil, errors.New("Server is nil or address is empty")
		}
		if server.MaxConn == 0 {
			server.MaxConn = defaultMaxConn
		}
		if server.InitConn == 0 {
			server.InitConn = defaultInitConn
		}
		if server.IdleTime == 0 {
			server.IdleTime = defaultIdleTime
		}
		server.isActive = true
	}

	mem.manager = &serverManager{
		serverList: server_list,
	}

	//create server hash node
	mem.nodes = createServerNode(server_list)
	return mem, nil
} /*}}}*/

//设置是否移除不可用server
func (this *Memcache) SetRemoveBadServer(option bool) { /*{{{*/
	if option == false {
		return
	}
	this.manager.isRmBadServer = option
	this.manager.badServerNotice = make(chan bool)

	go this.monitorBadServer()
} /*}}}*/

func (this *Memcache) SetTimeout(dial, read, write time.Duration) { /*{{{*/
	dialTimeout = dial
	readTimeout = read
	writeTimeout = write
} /*}}}*/

func (this *Memcache) Get(key string, format ...interface{}) (value interface{}, cas uint64, err error) { /*{{{*/
	this.RLock()
	defer this.RUnlock()

	var res *response
	server := this.nodes.getServerByKey(key)
	if server == nil {
		return nil, 0, ErrNotConn
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := server.pool.Get()
		if e != nil && e == ErrNotConn {
			this.sendBadServerNotice()
			return nil, 0, e
		}

		res, err = conn.get(key, format...)

		if err == ErrBadConn {
			server.pool.Release(conn)
		} else {
			server.pool.Put(conn)
			break
		}
	}

	if res != nil {
		return res.body, res.header.cas, err
	} else {
		return nil, 0, err
	}
} /*}}}*/

func (this *Memcache) Set(key string, value interface{}, expire ...uint32) (res bool, err error) { /*{{{*/
	this.RLock()
	defer this.RUnlock()
	var timeout uint32 = 0

	if len(expire) > 0 {
		timeout = expire[0]
	}
	server := this.nodes.getServerByKey(key)
	if server == nil {
		return false, ErrNotConn
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := server.pool.Get()
		if e != nil && e == ErrNotConn {
			this.sendBadServerNotice()
			return false, e
		}

		res, err = conn.store(OP_SET, key, value, timeout, 0)

		if err == ErrBadConn {
			server.pool.Release(conn)
		} else {
			server.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Add(key string, value interface{}, expire ...uint32) (res bool, err error) { /*{{{*/
	this.RLock()
	defer this.RUnlock()
	var timeout uint32 = 0

	if len(expire) > 0 {
		timeout = expire[0]
	}
	server := this.nodes.getServerByKey(key)
	if server == nil {
		return false, ErrNotConn
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := server.pool.Get()
		if e != nil && e == ErrNotConn {
			this.sendBadServerNotice()
			return false, e
		}

		res, err = conn.store(OP_ADD, key, value, timeout, 0)
		if err == ErrBadConn {
			server.pool.Release(conn)
		} else {
			server.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Replace(key string, value interface{}, args ...uint64) (res bool, err error) { /*{{{*/
	this.RLock()
	defer this.RUnlock()
	var timeout uint32 = 0
	var cas uint64 = 0

	switch len(args) {
	case 1:
		timeout = uint32(args[0])
	case 2:
		timeout = uint32(args[0])
		cas = args[1]
	}
	server := this.nodes.getServerByKey(key)
	if server == nil {
		return false, ErrNotConn
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := server.pool.Get()
		if e != nil && e == ErrNotConn {
			this.sendBadServerNotice()
			return false, e
		}

		res, err = conn.store(OP_REPLACE, key, value, timeout, cas)
		if err == ErrBadConn {
			server.pool.Release(conn)
		} else {
			server.pool.Put(conn)
			break
		}
	}
	return res, err
} /*}}}*/

func (this *Memcache) Delete(key string, cas ...uint64) (res bool, err error) { /*{{{*/
	this.RLock()
	defer this.RUnlock()
	server := this.nodes.getServerByKey(key)
	if server == nil {
		return false, ErrNotConn
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := server.pool.Get()
		if e != nil && e == ErrNotConn {
			this.sendBadServerNotice()
			return false, e
		}

		res, err = conn.delete(key, cas...)

		if err == ErrBadConn {
			server.pool.Release(conn)
		} else {
			server.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Increment(key string, args ...interface{}) (res bool, err error) { /*{{{*/
	this.RLock()
	defer this.RUnlock()
	server := this.nodes.getServerByKey(key)
	if server == nil {
		return false, ErrNotConn
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := server.pool.Get()
		if e != nil && e == ErrNotConn {
			this.sendBadServerNotice()
			return false, e
		}

		res, err = conn.numberic(OP_INCREMENT, key, args...)

		if err == ErrBadConn {
			server.pool.Release(conn)
		} else {
			server.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Decrement(key string, args ...interface{}) (res bool, err error) { /*{{{*/
	this.RLock()
	defer this.RUnlock()
	server := this.nodes.getServerByKey(key)
	if server == nil {
		return false, ErrNotConn
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := server.pool.Get()
		if e != nil && e == ErrNotConn {
			this.sendBadServerNotice()
			return false, e
		}

		res, err = conn.numberic(OP_DECREMENT, key, args...)

		if err == ErrBadConn {
			server.pool.Release(conn)
		} else {
			server.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Append(key string, value string, cas ...uint64) (res bool, err error) { /*{{{*/
	this.RLock()
	defer this.RUnlock()
	server := this.nodes.getServerByKey(key)
	if server == nil {
		return false, ErrNotConn
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := server.pool.Get()
		if e != nil && e == ErrNotConn {
			this.sendBadServerNotice()
			return false, e
		}

		res, err = conn.appends(OP_APPEND, key, value, cas...)

		if err == ErrBadConn {
			server.pool.Release(conn)
		} else {
			server.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Prepend(key string, value string, cas ...uint64) (res bool, err error) { /*{{{*/
	this.RLock()
	defer this.RUnlock()
	server := this.nodes.getServerByKey(key)
	if server == nil {
		return false, ErrNotConn
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := server.pool.Get()
		if e != nil && e == ErrNotConn {
			this.sendBadServerNotice()
			return false, e
		}

		res, err = conn.appends(OP_PREPEND, key, value, cas...)

		if err == ErrBadConn {
			server.pool.Release(conn)
		} else {
			server.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Flush(server *Server, delay ...uint32) (res bool, err error) { /*{{{*/
	for i := 0; i < badTryCnt; i++ {
		conn, e := server.pool.Get()
		if e != nil && e == ErrNotConn {
			this.sendBadServerNotice()
			return false, e
		}

		res, err = conn.flush(delay...)

		if err == ErrBadConn {
			server.pool.Release(conn)
		} else {
			server.pool.Put(conn)
			break
		}
	}
	return res, err
} /*}}}*/

func (this *Memcache) Version(server *Server) (v string, err error) { /*{{{*/
	for i := 0; i < badTryCnt; i++ {
		conn, e := server.pool.Get()
		if e != nil && e == ErrNotConn {
			this.sendBadServerNotice()
			return "", e
		}

		v, err = conn.version()

		if err == ErrBadConn {
			server.pool.Release(conn)
		} else {
			server.pool.Put(conn)
			break
		}
	}

	return v, err
} /*}}}*/

func (this *Memcache) sendBadServerNotice() { /*{{{*/
	if this.manager.isRmBadServer == false {
		return
	}

	select {
	case this.manager.badServerNotice <- true:
	default:
	}
} /*}}}*/

func (this *Memcache) monitorBadServer() { /*{{{*/
	for {
		select {
		case <-this.manager.badServerNotice:
			this.doDealBadServer()
		case <-time.After(time.Second * 120):
			this.doDealBadServer()
		}
	}
} /*}}}*/

func (this *Memcache) doDealBadServer() { /*{{{*/
	var res map[*Server]chan bool
	var isReload bool = false

	res = make(map[*Server]chan bool, len(this.manager.serverList))
	for _, s := range this.manager.serverList {
		res[s] = make(chan bool, 1)
		go this.checkServerActive(s, res[s])
	}

	for s, ch := range res {
		switch <-ch {
		case true:
			if s.isActive == false {
				s.pool.Close()
				s.pool = nil
				isReload = true
			}
			s.isActive = true
		case false:
			if s.isActive == true {
				isReload = true
			}
			s.isActive = false
		}

	}

	if isReload == false {
		return
	}
	//有server状态发生变化，重新生成node
	new_server_list := make([]*Server, 0)
	for _, s := range this.manager.serverList {
		if s.isActive == true {
			new_server_list = append(new_server_list, s)
		}
	}
	//create server hash node
	new_nodes := createServerNode(new_server_list)

	this.Lock()
	this.nodes = new_nodes
	this.Unlock()
} /*}}}*/

func (this *Memcache) checkServerActive(server *Server, ch chan bool) { /*{{{*/
	conn, e := server.pool.Get()
	if e != nil && e == ErrNotConn {
		//can't connect to server
		ch <- false
		return
	}

	res, err := conn.noop()

	if err == ErrBadConn {
		server.pool.Release(conn)
	} else {
		server.pool.Put(conn)
	}

	if err == nil && res == true {
		ch <- true
		return
	}

	//noop failed ,then try dial server
	if new_connection, err := connect(server.Address); err == ErrNotConn {
		ch <- false
	} else {
		ch <- true
		new_connection.Close()
	}
} /*}}}*/

func (this *Memcache) Close() {
	for _, s := range this.manager.serverList {
		s.pool.Close()
	}
}
