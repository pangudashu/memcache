package memcache

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"math"
)

//float 32/64 -> []byte
func Float32ToByte(float float32) []byte {
	bits := math.Float32bits(float)
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, bits)

	return bytes
}

func ByteToFloat32(bytes []byte) float32 {
	bits := binary.LittleEndian.Uint32(bytes)

	return math.Float32frombits(bits)
}

func Float64ToByte(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)

	return bytes
}

func ByteToFloat64(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)

	return math.Float64frombits(bits)
}

func StructToByte(value interface{}) (b []byte, err error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err = encoder.Encode(value); err != nil {
		return nil, errors.New("encode fail")
	}

	return buf.Bytes(), nil
}

func ByteToStruct(b []byte, value interface{}) (err error) {
	buf := bytes.NewBuffer(b)

	decoder := gob.NewDecoder(buf)
	if err = decoder.Decode(value); err != nil {
		return errors.New("decode fail")
	}

	return nil
}
