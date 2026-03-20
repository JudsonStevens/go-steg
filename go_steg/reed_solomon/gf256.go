package reed_solomon

import "sync"

const gfPoly = 0x11D

var (
	expTable [512]byte
	logTable [256]byte
	initOnce sync.Once
)

func initTables() {
	initOnce.Do(func() {
		x := 1
		for i := 0; i < 255; i++ {
			expTable[i] = byte(x)
			logTable[x] = byte(i)
			x <<= 1
			if x >= 256 {
				x ^= gfPoly
			}
		}
		for i := 255; i < 512; i++ {
			expTable[i] = expTable[i-255]
		}
	})
}

func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return expTable[int(logTable[a])+int(logTable[b])]
}

func gfInv(a byte) byte {
	if a == 0 {
		return 0
	}
	return expTable[255-int(logTable[a])]
}

func gfDiv(a, b byte) byte {
	if b == 0 {
		panic("reed_solomon: division by zero in GF(256)")
	}
	if a == 0 {
		return 0
	}
	return expTable[(int(logTable[a])-int(logTable[b])+255)%255]
}
