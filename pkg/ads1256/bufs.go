package ads1256

import "sync"

var (
	threeBytes = &sync.Pool{New: func() interface{} { return make([]byte, 3) }}
	oneByte    = &sync.Pool{New: func() interface{} { return make([]byte, 1) }}
)

func get3Bytes() []byte {
	return threeBytes.Get().([]byte)
}

func put3Bytes(b []byte) {
	b[0], b[1], b[2] = 0, 0, 0
	threeBytes.Put(b)
}

func get1Byte() []byte {
	return oneByte.Get().([]byte)
}

func put1Byte(b []byte) {
	b[0] = 0
	oneByte.Put(b)
}
