// SuperFastHash - Paul Hsieh's hash function (Go port)
//
// Ported from Chromium base/third_party/superfasthash/superfasthash.c
// This is the same algorithm used by Chromium/Electron to compute display IDs.
//
// Copyright (c) 2010, Paul Hsieh. All rights reserved.
// BSD license — see superfasthash.h for full license text.

package display

// SuperFastHash computes Paul Hsieh's SuperFastHash.
// This must produce identical results to the C version in Chromium
// for Electron display ID compatibility.
func SuperFastHash(data []byte) uint32 {
	length := len(data)
	if length <= 0 {
		return 0
	}

	hash := uint32(length)
	rem := length & 3
	length >>= 2

	i := 0
	for ; length > 0; length-- {
		hash += uint32(data[i]) | uint32(data[i+1])<<8
		tmp := (uint32(data[i+2]) | uint32(data[i+3])<<8) << 11 ^ hash
		hash = (hash << 16) ^ tmp
		i += 4
		hash += hash >> 11
	}

	switch rem {
	case 3:
		hash += uint32(data[i]) | uint32(data[i+1])<<8
		hash ^= hash << 16
		// int8 cast for sign extension, matching C: (int8_t)data[2]
		hash ^= uint32(int32(int8(data[i+2])) << 18)
		hash += hash >> 11
	case 2:
		hash += uint32(data[i]) | uint32(data[i+1])<<8
		hash ^= hash << 11
		hash += hash >> 17
	case 1:
		// int8 cast for sign extension, matching C: (int8_t)*data
		hash += uint32(int8(data[i]))
		hash ^= hash << 10
		hash += hash >> 1
	}

	hash ^= hash << 3
	hash += hash >> 5
	hash ^= hash << 4
	hash += hash >> 17
	hash ^= hash << 25
	hash += hash >> 6

	return hash
}
