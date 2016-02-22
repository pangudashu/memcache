package memcache

type Memcache struct {
	pool *ConnectionPool
}

var badTryCnt int = 4

func NewMemcache(address string, args ...int) (mem *Memcache, err error) { /*{{{*/
	maxCnt := 128
	initCnt := 0

	switch len(args) {
	case 1:
		maxCnt = args[0]
	case 2:
		maxCnt = args[0]
		initCnt = args[1]
	}

	pool := open(address, maxCnt, initCnt)

	if err != nil {
		return nil, err
	}

	return &Memcache{
		pool: pool,
	}, nil
} /*}}}*/

func (this *Memcache) Get(key string, format ...interface{}) (value interface{}, cas uint64, err error) { /*{{{*/
	var res *response
	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if e != nil {
			if i == badTryCnt-1 {
				return nil, 0, e
			} else {
				continue
			}
		}

		res, err = conn.get(key, format...)

		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
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
	var timeout uint32 = 0

	if len(expire) > 0 {
		timeout = expire[0]
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if e != nil {
			if i == badTryCnt-1 {
				return false, e
			} else {
				continue
			}
		}

		res, err = conn.store(OP_SET, key, value, timeout, 0)

		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Add(key string, value interface{}, expire ...uint32) (res bool, err error) { /*{{{*/
	var timeout uint32 = 0

	if len(expire) > 0 {
		timeout = expire[0]
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if e != nil {
			if i == badTryCnt-1 {
				return false, e
			} else {
				continue
			}
		}

		res, err = conn.store(OP_ADD, key, value, timeout, 0)
		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Replace(key string, value interface{}, args ...uint64) (res bool, err error) { /*{{{*/
	var timeout uint32 = 0
	var cas uint64 = 0

	switch len(args) {
	case 1:
		timeout = uint32(args[0])
	case 2:
		timeout = uint32(args[0])
		cas = args[1]
	}

	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if e != nil {
			if i == badTryCnt-1 {
				return false, e
			} else {
				continue
			}
		}

		res, err = conn.store(OP_REPLACE, key, value, timeout, cas)
		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
			break
		}
	}
	return res, err
} /*}}}*/

func (this *Memcache) Delete(key string, cas ...uint64) (res bool, err error) { /*{{{*/
	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if e != nil {
			if i == badTryCnt-1 {
				return false, e
			} else {
				continue
			}
		}

		res, err = conn.delete(key, cas...)

		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Increment(key string, args ...interface{}) (res bool, err error) { /*{{{*/
	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if e != nil {
			if i == badTryCnt-1 {
				return false, e
			} else {
				continue
			}
		}

		res, err = conn.numberic(OP_INCREMENT, key, args...)

		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Decrement(key string, args ...interface{}) (res bool, err error) { /*{{{*/
	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if err != nil {
			if i == badTryCnt-1 {
				return false, e
			} else {
				continue
			}
		}

		res, err = conn.numberic(OP_DECREMENT, key, args...)

		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Append(key string, value string, cas ...uint64) (res bool, err error) { /*{{{*/
	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if e != nil {
			if i == badTryCnt-1 {
				return false, e
			} else {
				continue
			}
		}

		res, err = conn.appends(OP_APPEND, key, value, cas...)

		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Prepend(key string, value string, cas ...uint64) (res bool, err error) { /*{{{*/
	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if e != nil {
			if i == badTryCnt-1 {
				return false, e
			} else {
				continue
			}
		}

		res, err = conn.appends(OP_PREPEND, key, value, cas...)

		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Flush(delay ...uint32) (res bool, err error) { /*{{{*/
	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if e != nil {
			if i == badTryCnt-1 {
				return false, e
			} else {
				continue
			}
		}

		res, err = conn.flush(delay...)

		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
			break
		}
	}
	return res, err
} /*}}}*/

func (this *Memcache) Noop() (res bool, err error) { /*{{{*/
	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if e != nil {
			if i == badTryCnt-1 {
				return false, e
			} else {
				continue
			}
		}

		res, err = conn.noop()

		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
			break
		}
	}

	return res, err
} /*}}}*/

func (this *Memcache) Version() (v string, err error) { /*{{{*/
	for i := 0; i < badTryCnt; i++ {
		conn, e := this.pool.Get()
		if e != nil {
			if i == badTryCnt-1 {
				return "", e
			} else {
				continue
			}
		}

		v, err = conn.version()

		if err == ErrBadConn {
			this.pool.Release(conn)
		} else {
			this.pool.Put(conn)
			break
		}
	}

	return v, err
} /*}}}*/

func (this *Memcache) Close() {
	this.pool.Close()
}
