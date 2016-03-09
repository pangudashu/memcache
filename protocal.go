package memcache

type magic_t uint8

const (
	MAGIC_REQ magic_t = 0x80
	MAGIC_RES magic_t = 0x81
)

type opcode_t uint8

const (
	OP_GET       opcode_t = 0x00
	OP_SET       opcode_t = 0x01
	OP_ADD       opcode_t = 0x02
	OP_REPLACE   opcode_t = 0x03
	OP_DELETE    opcode_t = 0x04
	OP_INCREMENT opcode_t = 0x05
	OP_DECREMENT opcode_t = 0x06
	OP_FLUSH     opcode_t = 0x08
	OP_NOOP      opcode_t = 0x0a
	OP_VERSION   opcode_t = 0x0b
	OP_GETK      opcode_t = 0x0c
	OP_APPEND    opcode_t = 0x0e
	OP_PREPEND   opcode_t = 0x0f
)

type status_t uint16

const (
	STATUS_SUCCESS         status_t = 0x00 //no error
	STATUS_KEY_ENOENT      status_t = 0x01 //Key not found
	STATUS_KEY_EEXISTS     status_t = 0x02 //Key exists
	STATUS_E2BIG           status_t = 0x03 //Value too long
	STATUS_EINVAL          status_t = 0x04 //Invalid arguments
	STATUS_NOT_STORED      status_t = 0x05 //Item not stored
	STATUS_DELTA_BADVAL    status_t = 0x06 //Incr/Decr on non-numberic value
	STATUS_AUTH_ERROR      status_t = 0x20 //
	STATUS_AUTH_CONTINUE   status_t = 0x21 //
	STATUS_UNKNOWN_COMMAND status_t = 0x81 //Unkown opcode
	STATUS_ENOMEM          status_t = 0x82 //Out of memery
)

type type_t uint8

const (
	TYPE_RAW_BYTES type_t = 0x00
)

//自定义flags
type value_type_t uint32

const (
	VALUE_TYPE_INT     value_type_t = 0x00000000 //0 /*change Type 0 => int to sure memcached can right deal Incr/Decr*/
	VALUE_TYPE_BIN     value_type_t = 0x00000001 //1
	VALUE_TYPE_BYTE    value_type_t = 0x00000002 //2
	VALUE_TYPE_INT8    value_type_t = 0x00000004 //4
	VALUE_TYPE_INT16   value_type_t = 0x00000008 //8
	VALUE_TYPE_INT32   value_type_t = 0x00000010 //16
	VALUE_TYPE_INT64   value_type_t = 0x00000020 //32
	VALUE_TYPE_UINT8   value_type_t = 0x00000040 //64
	VALUE_TYPE_UINT16  value_type_t = 0x00000080 //128
	VALUE_TYPE_UINT32  value_type_t = 0x00000100 //256
	VALUE_TYPE_UINT64  value_type_t = 0x00000200 //512
	VALUE_TYPE_FLOAT32 value_type_t = 0x00000400 //1024
	VALUE_TYPE_FLOAT64 value_type_t = 0x00000800 //2048
	VALUE_TYPE_STRING  value_type_t = 0x00001000 //4096
	VALUE_TYPE_BOOL    value_type_t = 0x00002000 //8192
)

//request header
type request_header struct {
	magic    magic_t
	opcode   opcode_t
	keylen   uint16
	extlen   uint8
	datatype type_t
	status   status_t
	bodylen  uint32
	opaque   uint32
	cas      uint64
}

//response header
type response_header struct {
	magic    magic_t
	opcode   opcode_t
	keylen   uint16
	extlen   uint8
	datatype type_t
	status   status_t
	bodylen  uint32
	opaque   uint32
	cas      uint64
}

type response struct {
	header   *response_header
	bodyByte []byte
	body     interface{}
}
