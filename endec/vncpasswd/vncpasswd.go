package vncpasswd

import (
	"encoding/hex"
	"strings"
)

var _SP1 = [64]uint32{
	0x01010400, 0x00000000, 0x00010000, 0x01010404,
	0x01010004, 0x00010404, 0x00000004, 0x00010000,
	0x00000400, 0x01010400, 0x01010404, 0x00000400,
	0x01000404, 0x01010004, 0x01000000, 0x00000004,
	0x00000404, 0x01000400, 0x01000400, 0x00010400,
	0x00010400, 0x01010000, 0x01010000, 0x01000404,
	0x00010004, 0x01000004, 0x01000004, 0x00010004,
	0x00000000, 0x00000404, 0x00010404, 0x01000000,
	0x00010000, 0x01010404, 0x00000004, 0x01010000,
	0x01010400, 0x01000000, 0x01000000, 0x00000400,
	0x01010004, 0x00010000, 0x00010400, 0x01000004,
	0x00000400, 0x00000004, 0x01000404, 0x00010404,
	0x01010404, 0x00010004, 0x01010000, 0x01000404,
	0x01000004, 0x00000404, 0x00010404, 0x01010400,
	0x00000404, 0x01000400, 0x01000400, 0x00000000,
	0x00010004, 0x00010400, 0x00000000, 0x01010004}

var _SP2 = [64]uint32{
	0x80108020, 0x80008000, 0x00008000, 0x00108020,
	0x00100000, 0x00000020, 0x80100020, 0x80008020,
	0x80000020, 0x80108020, 0x80108000, 0x80000000,
	0x80008000, 0x00100000, 0x00000020, 0x80100020,
	0x00108000, 0x00100020, 0x80008020, 0x00000000,
	0x80000000, 0x00008000, 0x00108020, 0x80100000,
	0x00100020, 0x80000020, 0x00000000, 0x00108000,
	0x00008020, 0x80108000, 0x80100000, 0x00008020,
	0x00000000, 0x00108020, 0x80100020, 0x00100000,
	0x80008020, 0x80100000, 0x80108000, 0x00008000,
	0x80100000, 0x80008000, 0x00000020, 0x80108020,
	0x00108020, 0x00000020, 0x00008000, 0x80000000,
	0x00008020, 0x80108000, 0x00100000, 0x80000020,
	0x00100020, 0x80008020, 0x80000020, 0x00100020,
	0x00108000, 0x00000000, 0x80008000, 0x00008020,
	0x80000000, 0x80100020, 0x80108020, 0x00108000}

var _SP3 = [64]uint32{
	0x00000208, 0x08020200, 0x00000000, 0x08020008,
	0x08000200, 0x00000000, 0x00020208, 0x08000200,
	0x00020008, 0x08000008, 0x08000008, 0x00020000,
	0x08020208, 0x00020008, 0x08020000, 0x00000208,
	0x08000000, 0x00000008, 0x08020200, 0x00000200,
	0x00020200, 0x08020000, 0x08020008, 0x00020208,
	0x08000208, 0x00020200, 0x00020000, 0x08000208,
	0x00000008, 0x08020208, 0x00000200, 0x08000000,
	0x08020200, 0x08000000, 0x00020008, 0x00000208,
	0x00020000, 0x08020200, 0x08000200, 0x00000000,
	0x00000200, 0x00020008, 0x08020208, 0x08000200,
	0x08000008, 0x00000200, 0x00000000, 0x08020008,
	0x08000208, 0x00020000, 0x08000000, 0x08020208,
	0x00000008, 0x00020208, 0x00020200, 0x08000008,
	0x08020000, 0x08000208, 0x00000208, 0x08020000,
	0x00020208, 0x00000008, 0x08020008, 0x00020200}

var _SP4 = [64]uint32{
	0x00802001, 0x00002081, 0x00002081, 0x00000080,
	0x00802080, 0x00800081, 0x00800001, 0x00002001,
	0x00000000, 0x00802000, 0x00802000, 0x00802081,
	0x00000081, 0x00000000, 0x00800080, 0x00800001,
	0x00000001, 0x00002000, 0x00800000, 0x00802001,
	0x00000080, 0x00800000, 0x00002001, 0x00002080,
	0x00800081, 0x00000001, 0x00002080, 0x00800080,
	0x00002000, 0x00802080, 0x00802081, 0x00000081,
	0x00800080, 0x00800001, 0x00802000, 0x00802081,
	0x00000081, 0x00000000, 0x00000000, 0x00802000,
	0x00002080, 0x00800080, 0x00800081, 0x00000001,
	0x00802001, 0x00002081, 0x00002081, 0x00000080,
	0x00802081, 0x00000081, 0x00000001, 0x00002000,
	0x00800001, 0x00002001, 0x00802080, 0x00800081,
	0x00002001, 0x00002080, 0x00800000, 0x00802001,
	0x00000080, 0x00800000, 0x00002000, 0x00802080}

var _SP5 = [64]uint32{
	0x00000100, 0x02080100, 0x02080000, 0x42000100,
	0x00080000, 0x00000100, 0x40000000, 0x02080000,
	0x40080100, 0x00080000, 0x02000100, 0x40080100,
	0x42000100, 0x42080000, 0x00080100, 0x40000000,
	0x02000000, 0x40080000, 0x40080000, 0x00000000,
	0x40000100, 0x42080100, 0x42080100, 0x02000100,
	0x42080000, 0x40000100, 0x00000000, 0x42000000,
	0x02080100, 0x02000000, 0x42000000, 0x00080100,
	0x00080000, 0x42000100, 0x00000100, 0x02000000,
	0x40000000, 0x02080000, 0x42000100, 0x40080100,
	0x02000100, 0x40000000, 0x42080000, 0x02080100,
	0x40080100, 0x00000100, 0x02000000, 0x42080000,
	0x42080100, 0x00080100, 0x42000000, 0x42080100,
	0x02080000, 0x00000000, 0x40080000, 0x42000000,
	0x00080100, 0x02000100, 0x40000100, 0x00080000,
	0x00000000, 0x40080000, 0x02080100, 0x40000100}

var _SP6 = [64]uint32{
	0x20000010, 0x20400000, 0x00004000, 0x20404010,
	0x20400000, 0x00000010, 0x20404010, 0x00400000,
	0x20004000, 0x00404010, 0x00400000, 0x20000010,
	0x00400010, 0x20004000, 0x20000000, 0x00004010,
	0x00000000, 0x00400010, 0x20004010, 0x00004000,
	0x00404000, 0x20004010, 0x00000010, 0x20400010,
	0x20400010, 0x00000000, 0x00404010, 0x20404000,
	0x00004010, 0x00404000, 0x20404000, 0x20000000,
	0x20004000, 0x00000010, 0x20400010, 0x00404000,
	0x20404010, 0x00400000, 0x00004010, 0x20000010,
	0x00400000, 0x20004000, 0x20000000, 0x00004010,
	0x20000010, 0x20404010, 0x00404000, 0x20400000,
	0x00404010, 0x20404000, 0x00000000, 0x20400010,
	0x00000010, 0x00004000, 0x20400000, 0x00404010,
	0x00004000, 0x00400010, 0x20004010, 0x00000000,
	0x20404000, 0x20000000, 0x00400010, 0x20004010}

var _SP7 = [64]uint32{
	0x00200000, 0x04200002, 0x04000802, 0x00000000,
	0x00000800, 0x04000802, 0x00200802, 0x04200800,
	0x04200802, 0x00200000, 0x00000000, 0x04000002,
	0x00000002, 0x04000000, 0x04200002, 0x00000802,
	0x04000800, 0x00200802, 0x00200002, 0x04000800,
	0x04000002, 0x04200000, 0x04200800, 0x00200002,
	0x04200000, 0x00000800, 0x00000802, 0x04200802,
	0x00200800, 0x00000002, 0x04000000, 0x00200800,
	0x04000000, 0x00200800, 0x00200000, 0x04000802,
	0x04000802, 0x04200002, 0x04200002, 0x00000002,
	0x00200002, 0x04000000, 0x04000800, 0x00200000,
	0x04200800, 0x00000802, 0x00200802, 0x04200800,
	0x00000802, 0x04000002, 0x04200802, 0x04200000,
	0x00200800, 0x00000000, 0x00000002, 0x04200802,
	0x00000000, 0x00200802, 0x04200000, 0x00000800,
	0x04000002, 0x04000800, 0x00000800, 0x00200002}

var _SP8 = [64]uint32{
	0x10001040, 0x00001000, 0x00040000, 0x10041040,
	0x10000000, 0x10001040, 0x00000040, 0x10000000,
	0x00040040, 0x10040000, 0x10041040, 0x00041000,
	0x10041000, 0x00041040, 0x00001000, 0x00000040,
	0x10040000, 0x10000040, 0x10001000, 0x00001040,
	0x00041000, 0x00040040, 0x10040040, 0x10041000,
	0x00001040, 0x00000000, 0x00000000, 0x10040040,
	0x10000040, 0x10001000, 0x00041040, 0x00040000,
	0x00041040, 0x00040000, 0x10041000, 0x00001000,
	0x00000040, 0x10040040, 0x00001000, 0x00041040,
	0x10001000, 0x00000040, 0x10000040, 0x10040000,
	0x10040040, 0x10000000, 0x00040000, 0x10001040,
	0x00000000, 0x10041040, 0x00040040, 0x10000040,
	0x10040000, 0x10001000, 0x10001040, 0x00000000,
	0x10041040, 0x00041000, 0x00041000, 0x00001040,
			0x00001040, 0x00040040, 0x10000000, 0x10041000}
var _EN0 int = 0 /* MODE == encrypt */
var _DE1 int = 1 /* MODE == decrypt */

var _KnL = [32]uint32{}

const (
	_MAXPWLEN      = 8
	_CHALLENGESIZE = 16
)

var s_fixedkey = [8]byte{23, 82, 107, 6, 35, 78, 88, 7}
var bytebit = [8]byte{01, 02, 04, 010, 020, 040, 0100, 0200}

var bigbyte = [24]uint32{0x800000, 0x400000, 0x200000, 0x100000,
	0x80000, 0x40000, 0x20000, 0x10000,
	0x8000, 0x4000, 0x2000, 0x1000,
	0x800, 0x400, 0x200, 0x100,
	0x80, 0x40, 0x20, 0x10,
	0x8, 0x4, 0x2, 0x1}

/* Use the key schede specified in the Standard (ANSI X3.92-1981). */

var pc1 = [56]byte{56, 48, 40, 32, 24, 16, 8, 0, 57, 49, 41, 33, 25, 17,
	9, 1, 58, 50, 42, 34, 26, 18, 10, 2, 59, 51, 43, 35,
	62, 54, 46, 38, 30, 22, 14, 6, 61, 53, 45, 37, 29, 21,
	13, 5, 60, 52, 44, 36, 28, 20, 12, 4, 27, 19, 11, 3}

var totrot = [16]byte{1, 2, 4, 6, 8, 10, 12, 14, 15, 17, 19, 21, 23, 25, 27, 28}

var pc2 = [48]byte{13, 16, 10, 23, 0, 4, 2, 27, 14, 5, 20, 9,
	22, 18, 11, 3, 25, 7, 15, 6, 26, 19, 12, 1,
	40, 51, 30, 36, 46, 54, 29, 39, 50, 44, 32, 47,
	43, 48, 38, 55, 33, 52, 45, 41, 49, 35, 28, 31}

func deskey(key []byte, edf int) {
	var i, j, l, m, n int
	var pc1m, pcr [56]byte
	var kn [32]uint32

	for j = 0; j < 56; j++ {
		l = int(pc1[j])
		m = l & 07
		if (key[l>>3] & bytebit[m]) != 0 {
			pc1m[j] = 1
		} else {
			pc1m[j] = 0
		}
	}

	for i = 0; i < 16; i++ {
		if edf == _DE1 {
			m = (15 - i) << 1
		} else {
			m = i << 1
		}
		n = m + 1

		kn[m], kn[n] = 0, 0
		for j = 0; j < 28; j++ {
			l = j + int(totrot[i])
			if l < 28 {
				pcr[j] = pc1m[l]
			} else {
				pcr[j] = pc1m[l-28]
			}
		}
		for j = 28; j < 56; j++ {
			l = j + int(totrot[i])
			if l < 56 {
				pcr[j] = pc1m[l]
			} else {
				pcr[j] = pc1m[l-28]
			}
		}
		for j = 0; j < 24; j++ {
			if pcr[pc2[j]] != 0 {
				kn[m] |= bigbyte[j]
			}
			if pcr[pc2[j+24]] != 0 {
				kn[n] |= bigbyte[j]
			}
		}
	}
	cookey(kn)
}

func cookey(aw1 [32]uint32) {
	var idxdough, idxraw0, idxraw1 int
	var dough [32]uint32
	var i int
	for i = 0; i < 16; i++ {
		idxraw0 = idxraw1
		idxraw1++
		dough[idxdough] = (aw1[idxraw0] & 0x00fc0000) << 6
		dough[idxdough] |= (aw1[idxraw0] & 0x00000fc0) << 10
		dough[idxdough] |= (aw1[idxraw1] & 0x00fc0000) >> 10
		dough[idxdough] |= (aw1[idxraw1] & 0x00000fc0) >> 6
		idxdough++
		dough[idxdough] = (aw1[idxraw0] & 0x0003f000) << 12
		dough[idxdough] |= (aw1[idxraw0] & 0x0000003f) << 16
		dough[idxdough] |= (aw1[idxraw1] & 0x0003f000) >> 4
		dough[idxdough] |= (aw1[idxraw1] & 0x0000003f)
		idxdough++
		idxraw1++
	}
	usekey(dough)
}

func cpkey(from *[32]uint32) {
	for to := 0; to < len(_KnL); to++ {
		from[to] = _KnL[to]
	}
}

func usekey(from [32]uint32) {
	// copy(KnL, from)
	for to := 0; to < len(_KnL); to++ {
		_KnL[to] = from[to]
	}
}

func des(inblock []byte, outblock *[]byte) {
	var work = new([2]uint32)
	scrunch(inblock, work) //clone inblock -> work
	desfunc(work, _KnL)
	unscrun(*work, outblock)
}

func scrunch(outof []byte, into *[2]uint32) {
	(*into)[0] = (uint32(outof[0]) & uint32(0xff)) << 24
	(*into)[0] |= (uint32(outof[1]) & uint32(0xff)) << 16
	(*into)[0] |= (uint32(outof[2]) & uint32(0xff)) << 8
	(*into)[0] |= (uint32(outof[3]) & uint32(0xff))
	(*into)[1] = (uint32(outof[4]) & uint32(0xff)) << 24
	(*into)[1] |= (uint32(outof[5]) & uint32(0xff)) << 16
	(*into)[1] |= (uint32(outof[6]) & uint32(0xff)) << 8
	(*into)[1] |= (uint32(outof[7]) & uint32(0xff))
}

func unscrun(outof [2]uint32, into *[]byte) {
	(*into)[0] = byte((outof[0] >> 24) & 0xff)
	(*into)[1] = byte((outof[0] >> 16) & 0xff)
	(*into)[2] = byte((outof[0] >> 8) & 0xff)
	(*into)[3] = byte(outof[0] & 0xff)
	(*into)[4] = byte((outof[1] >> 24) & 0xff)
	(*into)[5] = byte((outof[1] >> 16) & 0xff)
	(*into)[6] = byte((outof[1] >> 8) & 0xff)
	(*into)[7] = byte(outof[1] & 0xff)
}

//block is rewrite
func desfunc(block *[2]uint32, keys [32]uint32) {
	var fval, work, right, leftt uint32
	var round int

	leftt = block[0]
	right = block[1]
	work = ((leftt >> 4) ^ right) & uint32(0x0f0f0f0f)
	right ^= work
	leftt ^= (work << 4)
	work = ((leftt >> 16) ^ right) & uint32(0x0000ffff)
	right ^= work
	leftt ^= (work << 16)
	work = ((right >> 2) ^ leftt) & uint32(0x33333333)
	leftt ^= work
	right ^= (work << 2)
	work = ((right >> 8) ^ leftt) & uint32(0x00ff00ff)
	leftt ^= work
	right ^= (work << 8)
	right = ((right << 1) | ((right >> 31) & 1)) & uint32(0xffffffff)
	work = (leftt ^ right) & uint32(0xaaaaaaaa)
	leftt ^= work
	right ^= work
	leftt = ((leftt << 1) | ((leftt >> 31) & uint32(1))) & uint32(0xffffffff)
	var idxkeys int
	for round = 0; round < 8; round++ {
		work = (right << 28) | (right >> 4)
		work ^= keys[idxkeys]
		idxkeys++
		fval = _SP7[work&0x3f]
		fval |= _SP5[(work>>8)&uint32(0x3f)]
		fval |= _SP3[(work>>16)&uint32(0x3f)]
		fval |= _SP1[(work>>24)&uint32(0x3f)]
		work = right ^ keys[idxkeys]
		idxkeys++
		fval |= _SP8[work&uint32(0x3f)]
		fval |= _SP6[(work>>8)&uint32(0x3f)]
		fval |= _SP4[(work>>16)&uint32(0x3f)]
		fval |= _SP2[(work>>24)&uint32(0x3f)]
		leftt ^= fval
		work = (leftt << 28) | (leftt >> 4)
		work ^= keys[idxkeys]
		idxkeys++
		fval = _SP7[work&uint32(0x3f)]
		fval |= _SP5[(work>>8)&uint32(0x3f)]
		fval |= _SP3[(work>>16)&uint32(0x3f)]
		fval |= _SP1[(work>>24)&uint32(0x3f)]
		work = leftt ^ keys[idxkeys]
		idxkeys++
		fval |= _SP8[work&uint32(0x3f)]
		fval |= _SP6[(work>>8)&uint32(0x3f)]
		fval |= _SP4[(work>>16)&uint32(0x3f)]
		fval |= _SP2[(work>>24)&uint32(0x3f)]
		right ^= fval
	}

	right = (right << 31) | (right >> 1)
	work = (leftt ^ right) & uint32(0xaaaaaaaa)
	leftt ^= work
	right ^= work
	leftt = (leftt << 31) | (leftt >> 1)
	work = ((leftt >> 8) ^ right) & uint32(0x00ff00ff)
	right ^= work
	leftt ^= (work << 8)
	work = ((leftt >> 2) ^ right) & uint32(0x33333333)
	right ^= work
	leftt ^= (work << 2)
	work = ((right >> 16) ^ leftt) & uint32(0x0000ffff)
	leftt ^= work
	right ^= (work << 16)
	work = ((right >> 4) ^ leftt) & uint32(0x0f0f0f0f)
	leftt ^= work
	right ^= (work << 4)
	block[0] = right
	block[1] = leftt
}

func VncEncryptBytes(rawbytes []byte, passwd []byte) (encryptedBytes []byte) {
	var i int
	var key = make([]byte, 8)
	/* key is simply the password padded with nulls */
	for i = 0; i < 8; i++ {
		if i < len(passwd) {
			key[i] = passwd[i]
		} else {
			key[i] = 0
		}
	}
	if len(rawbytes) > _MAXPWLEN {
		rawbytes = rawbytes[:_MAXPWLEN]
	}
	if len(rawbytes) < _CHALLENGESIZE {
		padding := make([]byte, _CHALLENGESIZE-len(passwd)+1)
		rawbytes = append(rawbytes, padding...)
	}
	deskey(key, _EN0)
	encryptedBytes = make([]byte, _CHALLENGESIZE)
	for i = 0; i < _CHALLENGESIZE; i += 8 {
		des(rawbytes[i:], &key)
		copy(encryptedBytes[i:], key)
	}
	return encryptedBytes
}

func VncEncryptPasswd(passwd string) (encryptedPasswd []byte) {
	encryptedPasswd = make([]byte, _MAXPWLEN)
	deskey(s_fixedkey[:], _EN0)
	rawpasswd := make([]byte, _MAXPWLEN)
	passwdbytes := []byte(passwd)
	if len(passwdbytes) >= _MAXPWLEN {
		rawpasswd = passwdbytes[:_MAXPWLEN]
	} else {
		copy(rawpasswd, passwdbytes)
	}
	des(rawpasswd, &encryptedPasswd)
	return
}

func VncEncryptPasswdToHexString(passwd string) (encryptedPasswd string) {
	byteArraybs := VncEncryptPasswd(passwd)
	encryptedPasswd = strings.ToUpper(hex.EncodeToString(byteArraybs))
	return
}

func VncDecryptPasswd(encryptedPasswd []byte) (decryptedPasswdStr string, ok bool) {
	if len(encryptedPasswd) == 0 {
		return "", false
	}
	var key = make([]byte, 8)

	deskey(s_fixedkey[:], _DE1)
	des(encryptedPasswd, &key)
	return string(key), true
}

func VncDecryptPasswdFromHexString(encryptedPasswd string) (decryptedPasswdStr string, ok bool) {
	if len(encryptedPasswd) == 0 {
		return "", false
	}
	var key = make([]byte, 8)
	if byteArr, err := hex.DecodeString(encryptedPasswd); err == nil {
		deskey(s_fixedkey[:], _DE1)
		des(byteArr, &key)
		return string(key), true
	}
	return
}
