package memcache

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type Connection struct {
	c              net.Conn
	buffered       bufio.ReadWriter
	lastActiveTime time.Time
}

var (
	dialTimeout  time.Duration
	writeTimeout time.Duration
	readTimeout  time.Duration
)

func connect(address string) (conn *Connection, err error) { /*{{{*/
	var network string
	if strings.Contains(address, "/") {
		network = "unix"
	} else {
		network = "tcp"
	}
	var nc net.Conn

	if dialTimeout > 0 {
		nc, err = net.DialTimeout(network, address, dialTimeout)
	} else {
		nc, err = net.Dial(network, address)
	}
	if err != nil {
		return nil, ErrNotConn
	}
	return newConnection(nc), nil
} /*}}}*/

func newConnection(c net.Conn) *Connection { /*{{{*/
	return &Connection{
		c: c,
		buffered: *bufio.NewReadWriter(
			bufio.NewReader(c),
			bufio.NewWriter(c),
		),
		lastActiveTime: time.Now(),
	}
} /*}}}*/

func (this *Connection) readResponse() (*response, error) { /*{{{*/
	b := make([]byte, 24)

	if readTimeout > 0 {
		this.c.SetReadDeadline(time.Now().Add(readTimeout))
	}

	if _, err := this.buffered.Read(b); err != nil {
		//if err == io.EOF {
		return nil, ErrBadConn
		//} else {
		//	return nil, err
		//}
	}

	response_header := this.parseHeader(b)

	if response_header.magic != MAGIC_RES {
		return nil, errors.New("invalid magic")
	}

	res := &response{header: response_header}

	if response_header.bodylen > 0 {
		if readTimeout > 0 {
			this.c.SetReadDeadline(time.Now().Add(readTimeout))
		}

		res.bodyByte = make([]byte, response_header.bodylen)
		io.ReadFull(this.buffered, res.bodyByte)
	}

	return res, nil
} /*}}}*/

func (this *Connection) flushBufferToServer() error { /*{{{*/
	if writeTimeout > 0 {
		this.c.SetWriteDeadline(time.Now().Add(writeTimeout))
	}
	return this.buffered.Flush()
} /*}}}*/

func (this *Connection) get(key string, format ...interface{}) (res *response, err error) { /*{{{*/
	header := &request_header{
		magic:    MAGIC_REQ,
		opcode:   OP_GET,
		keylen:   uint16(len(key)),
		extlen:   0x00,
		datatype: TYPE_RAW_BYTES,
		status:   0x00,
		bodylen:  uint32(len(key)),
		opaque:   0x00,
		cas:      0x00,
	}
	if err := this.writeHeader(header); err != nil {
		return nil, err
	}

	this.buffered.WriteString(key)

	if err := this.flushBufferToServer(); err != nil {
		return nil, ErrBadConn
	}

	resp, err := this.readResponse()
	if err != nil {
		return resp, err
	}

	if err := this.checkResponseError(resp.header.status); err != nil {
		return resp, err
	}

	if resp.header.bodylen > 0 {
		flags := binary.BigEndian.Uint32(resp.bodyByte[:resp.header.extlen])
		res_value, err := this.formatValueFromByte(value_type_t(flags), resp.bodyByte[resp.header.extlen:], format...)
		if err != nil {
			res_value = nil
		}

		resp.body = res_value
		return resp, err
	} else {
		return resp, errors.New("unkown error")
	}
} /*}}}*/

func (this *Connection) delete(key string, cas ...uint64) (res bool, err error) { /*{{{*/
	var set_cas uint64 = 0
	if len(cas) > 0 {
		set_cas = cas[0]
	}
	header := &request_header{
		magic:    MAGIC_REQ,
		opcode:   OP_DELETE,
		keylen:   uint16(len(key)),
		extlen:   0x00,
		datatype: TYPE_RAW_BYTES,
		status:   0x00,
		bodylen:  uint32(len(key)),
		opaque:   0x00,
		cas:      set_cas,
	}

	if err := this.writeHeader(header); err != nil {
		return false, err
	}
	this.buffered.Write([]byte(key))

	if err := this.flushBufferToServer(); err != nil {
		return false, ErrBadConn
	}

	resp, err := this.readResponse()
	if err != nil {
		return false, err
	}

	if err := this.checkResponseError(resp.header.status); err != nil {
		return false, err
	}

	return true, nil
} /*}}}*/

func (this *Connection) numberic(opcode opcode_t, key string, args ...interface{}) (res bool, err error) { /*{{{*/
	var delta = 1
	var cas = 0

	switch len(args) {
	case 1:
		delta = args[0].(int)
	case 2:
		delta = args[0].(int)
		cas = args[1].(int)
	}

	header := &request_header{
		magic:    MAGIC_REQ,
		opcode:   opcode,
		keylen:   uint16(len(key)),
		extlen:   0x14,
		datatype: TYPE_RAW_BYTES,
		status:   0x00,
		bodylen:  uint32(len(key) + 0x14),
		opaque:   0x00,
		cas:      uint64(cas),
	}
	extra_byte := make([]byte, 0x14)
	binary.BigEndian.PutUint64(extra_byte[0:8], uint64(delta))
	binary.BigEndian.PutUint64(extra_byte[8:16], 0x0000000000000000 /*uint64(initial)*/)
	binary.BigEndian.PutUint32(extra_byte[16:20], 0x00000000 /*uint32(expiration) If the expiration value is all one-bits (0xffffffff), the operation will fail with NOT_FOUND*/)

	if err := this.writeHeader(header); err != nil {
		return false, err
	}

	this.buffered.Write(extra_byte)
	this.buffered.Write([]byte(key))

	if err := this.flushBufferToServer(); err != nil {
		return false, ErrBadConn
	}

	resp, err := this.readResponse()
	if err != nil {
		return false, err
	}

	if err := this.checkResponseError(resp.header.status); err != nil {
		return false, err
	}

	return true, nil
} /*}}}*/

func (this *Connection) store(opcode opcode_t, key string, value interface{}, timeout uint32, cas uint64) (res bool, err error) { /*{{{*/
	val, data_type := this.getValueTypeByte(value)
	if val == nil {
		return false, ErrInvalValue
	}

	header := &request_header{
		magic:    MAGIC_REQ,
		opcode:   opcode,
		keylen:   uint16(len(key)),
		extlen:   0x08,
		datatype: TYPE_RAW_BYTES,
		status:   0x00,
		bodylen:  uint32(len(key) + 0x08 + len(val)),
		opaque:   0x00,
		cas:      cas, //If the Data Version Check (CAS) is nonzero, the requested operation MUST only succeed if the item exists and has a CAS value identical to the provided value.
	}

	if err := this.writeHeader(header); err != nil {
		return false, err
	}

	extra_byte := make([]byte, 8)
	binary.BigEndian.PutUint32(extra_byte[0:4], uint32(data_type)) //uint32 flags
	binary.BigEndian.PutUint32(extra_byte[4:8], timeout)           //uint32 expiration

	this.buffered.Write(extra_byte)
	this.buffered.Write([]byte(key))
	this.buffered.Write(val)

	if err := this.flushBufferToServer(); err != nil {
		return false, ErrBadConn
	}

	resp, err := this.readResponse()
	if err != nil {
		return false, err
	}

	if err := this.checkResponseError(resp.header.status); err != nil {
		return false, err
	}

	return true, nil
} /*}}}*/

func (this *Connection) appends(opcode opcode_t, key string, value string, cas ...uint64) (res bool, err error) { /*{{{*/
	var set_cas uint64 = 0
	if len(cas) > 0 {
		set_cas = cas[0]
	}

	header := &request_header{
		magic:    MAGIC_REQ,
		opcode:   opcode,
		keylen:   uint16(len(key)),
		extlen:   0x00,
		datatype: TYPE_RAW_BYTES,
		status:   0x00,
		bodylen:  uint32(len(key) + len(value)),
		opaque:   0x00,
		cas:      set_cas,
	}

	if err := this.writeHeader(header); err != nil {
		return false, err
	}
	this.buffered.Write([]byte(key + value))

	if err := this.flushBufferToServer(); err != nil {
		return false, ErrBadConn
	}

	resp, err := this.readResponse()
	if err != nil {
		return false, err
	}

	if err := this.checkResponseError(resp.header.status); err != nil {
		return false, err
	}

	return true, nil
} /*}}}*/

func (this *Connection) flush(delay ...uint32) (res bool, err error) { /*{{{*/
	var set_delay uint32 = 0
	if len(delay) > 0 {
		set_delay = delay[0]
	}

	header := &request_header{
		magic:    MAGIC_REQ,
		opcode:   OP_FLUSH,
		keylen:   0x00,
		extlen:   0x04,
		datatype: TYPE_RAW_BYTES,
		status:   0x00,
		bodylen:  0x00000004,
		opaque:   0x00,
		cas:      0x00,
	}

	if err := this.writeHeader(header); err != nil {
		return false, err
	}

	extra_byte := make([]byte, 4)
	binary.BigEndian.PutUint32(extra_byte, set_delay)

	this.buffered.Write(extra_byte)

	if err := this.flushBufferToServer(); err != nil {
		return false, ErrBadConn
	}

	resp, err := this.readResponse()
	if err != nil {
		return false, err
	}

	if err := this.checkResponseError(resp.header.status); err != nil {
		return false, err
	}

	return true, nil
} /*}}}*/

func (this *Connection) noop() (res bool, err error) { /*{{{*/
	header := &request_header{
		magic:    MAGIC_REQ,
		opcode:   OP_NOOP,
		keylen:   0x00,
		extlen:   0x00,
		datatype: TYPE_RAW_BYTES,
		status:   0x00,
		bodylen:  0x00000000,
		opaque:   0x00,
		cas:      0x00,
	}

	if err := this.writeHeader(header); err != nil {
		return false, err
	}

	if err := this.flushBufferToServer(); err != nil {
		return false, ErrBadConn
	}

	resp, err := this.readResponse()
	if err != nil {
		return false, err
	}

	if err := this.checkResponseError(resp.header.status); err != nil {
		return false, err
	}

	return true, nil
} /*}}}*/

func (this *Connection) version() (v string, err error) { /*{{{*/
	header := &request_header{
		magic:    MAGIC_REQ,
		opcode:   OP_VERSION,
		keylen:   0x00,
		extlen:   0x00,
		datatype: TYPE_RAW_BYTES,
		status:   0x00,
		bodylen:  0x00000000,
		opaque:   0x00,
		cas:      0x00,
	}

	if err := this.writeHeader(header); err != nil {
		return "", err
	}

	if err := this.flushBufferToServer(); err != nil {
		return "", ErrBadConn
	}

	resp, err := this.readResponse()
	if err != nil {
		return "", err
	}

	if err := this.checkResponseError(resp.header.status); err != nil {
		return "", err
	}

	if resp.header.bodylen > 0 {
		return string(resp.bodyByte), nil
	} else {
		return "", nil
	}
} /*}}}*/

//check server returned status
func (this *Connection) checkResponseError(status status_t) (err error) { /*{{{*/
	switch status {
	case STATUS_SUCCESS:
		return nil
	case STATUS_KEY_ENOENT:
		return ErrNotFound
	case STATUS_KEY_EEXISTS:
		return ErrKeyExists
	case STATUS_E2BIG:
		return ErrBig
	case STATUS_EINVAL:
		return ErrInval
	case STATUS_NOT_STORED:
		return ErrNotStord
	case STATUS_DELTA_BADVAL:
		return ErrDeltaBadVal
	case STATUS_AUTH_ERROR:
		return ErrAuthError
	case STATUS_AUTH_CONTINUE:
		return ErrAuthContinue
	case STATUS_UNKNOWN_COMMAND:
		return ErrCmd
	case STATUS_ENOMEM:
		return ErrMem
	default:
		return ErrUnkown
	}
} /*}}}*/

func (this *Connection) parseHeader(b []byte) *response_header { /*{{{*/
	return &response_header{
		magic:    magic_t(b[0]),
		opcode:   opcode_t(b[1]),
		keylen:   uint16(binary.BigEndian.Uint16(b[2:4])),
		extlen:   uint8(b[4]),
		datatype: type_t(b[5]),
		status:   status_t(binary.BigEndian.Uint16(b[6:8])),
		bodylen:  uint32(binary.BigEndian.Uint32(b[8:12])),
		opaque:   uint32(binary.BigEndian.Uint32(b[12:16])),
		cas:      uint64(binary.BigEndian.Uint64(b[16:24])),
	}
} /*}}}*/

func (this *Connection) writeHeader(header *request_header) error { /*{{{*/
	bin_buf := make([]byte, 24)

	bin_buf[0] = byte(header.magic)
	bin_buf[1] = byte(header.opcode)

	binary.BigEndian.PutUint16(bin_buf[2:4], header.keylen)
	bin_buf[4] = byte(header.extlen)
	bin_buf[5] = byte(header.datatype)
	binary.BigEndian.PutUint16(bin_buf[6:8], uint16(header.status))
	binary.BigEndian.PutUint32(bin_buf[8:12], header.bodylen)
	binary.BigEndian.PutUint32(bin_buf[12:16], header.opaque)
	binary.BigEndian.PutUint64(bin_buf[16:24], header.cas)

	len, err := this.buffered.Write(bin_buf)

	if err != nil {
		return err
	}

	if len < 24 {
		return errors.New("write request header error")
	}

	return nil
} /*}}}*/

func (this *Connection) getValueTypeByte(value interface{}) (body_bin []byte, value_type value_type_t) { /*{{{*/
	switch v := value.(type) {
	case []byte:
		value_type = VALUE_TYPE_BYTE
		body_bin = v
	case int: //转为字符串处理
		value_type = VALUE_TYPE_INT
		s := strconv.Itoa(v)
		body_bin = []byte(s)
	case int8:
		value_type = VALUE_TYPE_INT8
		body_bin = make([]byte, 1)
		body_bin[0] = byte(v)
	case int16:
		value_type = VALUE_TYPE_INT16
		body_bin = make([]byte, 2)
		binary.LittleEndian.PutUint16(body_bin, uint16(v))
	case int32:
		value_type = VALUE_TYPE_INT32
		body_bin = make([]byte, 4)
		binary.LittleEndian.PutUint32(body_bin, uint32(v))
	case int64:
		value_type = VALUE_TYPE_INT64
		body_bin = make([]byte, 8)
		binary.LittleEndian.PutUint64(body_bin, uint64(v))
	case uint8:
		value_type = VALUE_TYPE_UINT8
		body_bin = make([]byte, 1)
		body_bin[0] = byte(v)
	case uint16:
		value_type = VALUE_TYPE_UINT16
		body_bin = make([]byte, 2)
		binary.LittleEndian.PutUint16(body_bin, v)
	case uint32:
		value_type = VALUE_TYPE_UINT32
		body_bin = make([]byte, 4)
		binary.LittleEndian.PutUint32(body_bin, v)
	case uint64:
		value_type = VALUE_TYPE_UINT64
		body_bin = make([]byte, 8)
		binary.LittleEndian.PutUint64(body_bin, v)
	case float32:
		value_type = VALUE_TYPE_FLOAT32
		body_bin = Float32ToByte(v)
	case float64:
		value_type = VALUE_TYPE_FLOAT64
		body_bin = Float64ToByte(v)
	case string:
		value_type = VALUE_TYPE_STRING
		body_bin = []byte(v)
	case bool:
		value_type = VALUE_TYPE_BOOL
		body_bin = make([]byte, 1)
		if value.(bool) {
			body_bin[0] = uint8(1)
		} else {
			body_bin[0] = uint8(0)
		}
	default: //其它数据类型：map、struct等统一尝试转byte (性能不高)
		value_type = VALUE_TYPE_BIN
		b, err := StructToByte(value)

		if err != nil {
			return nil, value_type
		}

		body_bin = b
	}

	return body_bin, value_type
} /*}}}*/

func (this *Connection) formatValueFromByte(value_type value_type_t, data []byte, format ...interface{}) (value interface{}, err error) { /*{{{*/
	switch value_type {
	case VALUE_TYPE_BYTE:
		value = data
	case VALUE_TYPE_INT:
		data = bytes.Trim(data, " ")
		s := string(data)
		value, err = strconv.Atoi(s)
	case VALUE_TYPE_INT8:
		value = int8(data[0])
	case VALUE_TYPE_INT16:
		value = int16(binary.LittleEndian.Uint16(data))
	case VALUE_TYPE_INT32:
		value = int32(binary.LittleEndian.Uint32(data))
	case VALUE_TYPE_INT64:
		value = int64(binary.LittleEndian.Uint64(data))
	case VALUE_TYPE_UINT8:
		value = uint8(data[0])
	case VALUE_TYPE_UINT16:
		value = binary.LittleEndian.Uint16(data)
	case VALUE_TYPE_UINT32:
		value = binary.LittleEndian.Uint32(data)
	case VALUE_TYPE_UINT64:
		value = binary.LittleEndian.Uint64(data)
	case VALUE_TYPE_FLOAT32:
		value = ByteToFloat32(data)
	case VALUE_TYPE_FLOAT64:
		value = ByteToFloat64(data)
	case VALUE_TYPE_STRING:
		value = string(data)
	case VALUE_TYPE_BOOL:
		if uint8(data[0]) == 1 {
			value = true
		} else {
			value = false
		}
	default:
		if len(format) == 0 {
			err = ErrNoFormat
		} else {
			if e := ByteToStruct(data, format[0]); e != nil {
				err = ErrInvalFormat
			}

		}
	}

	return value, err
} /*}}}*/

func (this *Connection) Close() { /*{{{**/
	if this.c != nil {
		this.c.Close()
		this.c = nil
	}
} /*}}}*/
