package client

import (
	"bytes"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func encodeRaw(r *bson.Registry, val interface{}) ([]byte, error) {
	var buf bytes.Buffer
	vw := bson.NewDocumentWriter(&buf)
	enc := bson.NewEncoder(vw)
	enc.SetRegistry(r)
	if err := enc.Encode(val); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func bsonUnmarshal(raw []byte, val interface{}) error {
	return decodeRaw(bson.NewRegistry(), raw, val)
}

func decodeRaw(r *bson.Registry, raw []byte, val interface{}) error {
	vr := bson.NewDocumentReader(bytes.NewReader(raw))
	dec := bson.NewDecoder(vr)
	dec.DefaultDocumentM()
	dec.SetRegistry(r)
	return dec.Decode(val)
}

func roundTrip(r *bson.Registry, original interface{}, decoded interface{}) error {
	raw, err := encodeRaw(r, original)
	if err != nil {
		return err
	}
	return decodeRaw(r, raw, decoded)
}
