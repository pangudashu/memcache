package memcache

import "errors"

//connection error
var (
	ErrBadConn = errors.New("Connect closed")
	ErrNotConn = errors.New("Can't connect to server")
)

//memcached server returned error
var (
	ErrNotFound     = errors.New("Key not found")
	ErrKeyExists    = errors.New("Key exists")
	ErrBig          = errors.New("Value too long")
	ErrInval        = errors.New("Invalid arguments")
	ErrNotStord     = errors.New("Item not stored")
	ErrDeltaBadVal  = errors.New("Increment/Decrement on non-numberic value")
	ErrAuthError    = errors.New("Auth error")
	ErrAuthContinue = errors.New("Auth continue")
	ErrCmd          = errors.New("Unkown commond")
	ErrMem          = errors.New("Out of memery")
	ErrUnkown       = errors.New("Unkown error")
)

var (
	ErrInvalValue  = errors.New("Unkown value type")
	ErrInvalFormat = errors.New("Invalid format struct")
	ErrNoFormat    = errors.New("Format struct empty")
)
