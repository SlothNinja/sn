package sn

import (
	"bytes"
	"encoding/gob"
)

func Encode(src interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)

	if err := enc.Encode(src); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Decode(dst interface{}, value []byte) error {
	buf := bytes.NewBuffer(value)
	dec := gob.NewDecoder(buf)
	return dec.Decode(dst)
}
