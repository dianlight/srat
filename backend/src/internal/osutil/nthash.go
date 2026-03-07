// Copyright 2025 The SRAT Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
//
// NT (NTLM) password hash computation in pure Go, without CGo.
//
// Algorithm: NT hash = MD4(UTF-16LE(password))
// Reference: http://en.wikipedia.org/wiki/NTLM
//            RFC 1320 (MD4 message-digest algorithm)

package osutil

import (
	"encoding/binary"
	"encoding/hex"
	"strings"
	"unicode/utf16"
)

// NTHash returns the NT (NTLM) password hash of password as an uppercase
// 32-character hex string.  This is the hash stored by Samba in its tdb/ldb
// database and shown by `pdbedit -L -w` in the fourth colon-separated field.
//
// Algorithm: MD4(UTF-16LE(password))
func NTHash(password string) string {
	// Encode password as UTF-16 little-endian (Windows / Samba native).
	u16 := utf16.Encode([]rune(password))
	buf := make([]byte, len(u16)*2)
	for i, c := range u16 {
		binary.LittleEndian.PutUint16(buf[i*2:], c)
	}
	sum := md4Sum(buf)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// ---------------------------------------------------------------------------
// MD4 message-digest algorithm (RFC 1320)
// ---------------------------------------------------------------------------

const (
	md4Init0 uint32 = 0x67452301
	md4Init1 uint32 = 0xEFCDAB89
	md4Init2 uint32 = 0x98BADCFE
	md4Init3 uint32 = 0x10325476
)

func md4RotL(x, n uint32) uint32 { return (x << n) | (x >> (32 - n)) }

// md4F, md4G, md4H are the three auxiliary functions from RFC 1320 §3.4.
func md4F(x, y, z uint32) uint32 { return (x & y) | (^x & z) }
func md4G(x, y, z uint32) uint32 { return (x & y) | (x & z) | (y & z) }
func md4H(x, y, z uint32) uint32 { return x ^ y ^ z }

// md4Block performs one 64-byte block compression (RFC 1320 §3.4, Step 4).
// The 16 little-endian words are in m[0..15].
func md4Block(a, b, c, d uint32, m [16]uint32) (uint32, uint32, uint32, uint32) {
	aa, bb, cc, dd := a, b, c, d

	// Round 1 — F function, no additive constant.
	a = md4RotL(a+md4F(b, c, d)+m[0], 3)
	d = md4RotL(d+md4F(a, b, c)+m[1], 7)
	c = md4RotL(c+md4F(d, a, b)+m[2], 11)
	b = md4RotL(b+md4F(c, d, a)+m[3], 19)
	a = md4RotL(a+md4F(b, c, d)+m[4], 3)
	d = md4RotL(d+md4F(a, b, c)+m[5], 7)
	c = md4RotL(c+md4F(d, a, b)+m[6], 11)
	b = md4RotL(b+md4F(c, d, a)+m[7], 19)
	a = md4RotL(a+md4F(b, c, d)+m[8], 3)
	d = md4RotL(d+md4F(a, b, c)+m[9], 7)
	c = md4RotL(c+md4F(d, a, b)+m[10], 11)
	b = md4RotL(b+md4F(c, d, a)+m[11], 19)
	a = md4RotL(a+md4F(b, c, d)+m[12], 3)
	d = md4RotL(d+md4F(a, b, c)+m[13], 7)
	c = md4RotL(c+md4F(d, a, b)+m[14], 11)
	b = md4RotL(b+md4F(c, d, a)+m[15], 19)

	// Round 2 — G function, additive constant 5A827999.
	const c1 = uint32(0x5A827999)
	a = md4RotL(a+md4G(b, c, d)+m[0]+c1, 3)
	d = md4RotL(d+md4G(a, b, c)+m[4]+c1, 5)
	c = md4RotL(c+md4G(d, a, b)+m[8]+c1, 9)
	b = md4RotL(b+md4G(c, d, a)+m[12]+c1, 13)
	a = md4RotL(a+md4G(b, c, d)+m[1]+c1, 3)
	d = md4RotL(d+md4G(a, b, c)+m[5]+c1, 5)
	c = md4RotL(c+md4G(d, a, b)+m[9]+c1, 9)
	b = md4RotL(b+md4G(c, d, a)+m[13]+c1, 13)
	a = md4RotL(a+md4G(b, c, d)+m[2]+c1, 3)
	d = md4RotL(d+md4G(a, b, c)+m[6]+c1, 5)
	c = md4RotL(c+md4G(d, a, b)+m[10]+c1, 9)
	b = md4RotL(b+md4G(c, d, a)+m[14]+c1, 13)
	a = md4RotL(a+md4G(b, c, d)+m[3]+c1, 3)
	d = md4RotL(d+md4G(a, b, c)+m[7]+c1, 5)
	c = md4RotL(c+md4G(d, a, b)+m[11]+c1, 9)
	b = md4RotL(b+md4G(c, d, a)+m[15]+c1, 13)

	// Round 3 — H function, additive constant 6ED9EBA1.
	const c2 = uint32(0x6ED9EBA1)
	a = md4RotL(a+md4H(b, c, d)+m[0]+c2, 3)
	d = md4RotL(d+md4H(a, b, c)+m[8]+c2, 9)
	c = md4RotL(c+md4H(d, a, b)+m[4]+c2, 11)
	b = md4RotL(b+md4H(c, d, a)+m[12]+c2, 15)
	a = md4RotL(a+md4H(b, c, d)+m[2]+c2, 3)
	d = md4RotL(d+md4H(a, b, c)+m[10]+c2, 9)
	c = md4RotL(c+md4H(d, a, b)+m[6]+c2, 11)
	b = md4RotL(b+md4H(c, d, a)+m[14]+c2, 15)
	a = md4RotL(a+md4H(b, c, d)+m[1]+c2, 3)
	d = md4RotL(d+md4H(a, b, c)+m[9]+c2, 9)
	c = md4RotL(c+md4H(d, a, b)+m[5]+c2, 11)
	b = md4RotL(b+md4H(c, d, a)+m[13]+c2, 15)
	a = md4RotL(a+md4H(b, c, d)+m[3]+c2, 3)
	d = md4RotL(d+md4H(a, b, c)+m[11]+c2, 9)
	c = md4RotL(c+md4H(d, a, b)+m[7]+c2, 11)
	b = md4RotL(b+md4H(c, d, a)+m[15]+c2, 15)

	return aa + a, bb + b, cc + c, dd + d
}

// md4Sum returns the 16-byte MD4 digest of data.
func md4Sum(data []byte) [16]byte {
	msgLen := uint64(len(data))

	// Append the mandatory 0x80 padding byte then zero bytes until the
	// message length (in bytes) is ≡ 56 (mod 64).
	data = append(data, 0x80)
	for len(data)%64 != 56 {
		data = append(data, 0x00)
	}

	// Append the original message length in *bits* as a 64-bit little-endian value.
	var lenBuf [8]byte
	binary.LittleEndian.PutUint64(lenBuf[:], msgLen*8)
	data = append(data, lenBuf[:]...)

	// Initialise state registers.
	a, b, c, d := md4Init0, md4Init1, md4Init2, md4Init3

	// Process each 64-byte block.
	for i := 0; i < len(data); i += 64 {
		var m [16]uint32
		for j := 0; j < 16; j++ {
			m[j] = binary.LittleEndian.Uint32(data[i+j*4:])
		}
		a, b, c, d = md4Block(a, b, c, d, m)
	}

	var digest [16]byte
	binary.LittleEndian.PutUint32(digest[0:], a)
	binary.LittleEndian.PutUint32(digest[4:], b)
	binary.LittleEndian.PutUint32(digest[8:], c)
	binary.LittleEndian.PutUint32(digest[12:], d)
	return digest
}
