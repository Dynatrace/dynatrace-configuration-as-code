/*
 * @license
 * Copyright 2023 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package idutils

import (
	"encoding/base64"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"regexp"
)

var uuidRegex = regexp.MustCompile(".*?([0-9a-fA-F]{8}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{12}).*?")

// GetNumericIDForObjectID parses the Settings Object ID of a Dynatrace Management Zone (only object with numeric IDs)
// into a numeric identifier. To achieve this is replicates the en-/decoding logic used in Dynatrace as closely as possible.
func GetNumericIDForObjectID(objectID string) (int, error) {
	decodedObjectID, err := base64.RawURLEncoding.DecodeString(objectID)
	if err != nil {
		return 0, fmt.Errorf("failed to decode objectID %q: %w", objectID, err)
	}

	matches := uuidRegex.FindSubmatch(decodedObjectID)
	if len(matches) != 2 {
		return 0, fmt.Errorf("failed to read UUID from decoded objectID %q: expected regex match for contained UUID but got %v", decodedObjectID, matches)
	}

	uuidString := string(matches[1])

	u, err := uuid.Parse(uuidString)
	if err != nil {
		return 0, fmt.Errorf("failed to parse UUID %q: %w", uuidString, err)
	}

	if u.Variant() == uuid.RFC4122 && u.Version() == 4 {
		return getLegacyNumericId(u), nil
	} else {
		return getNumericId(u), nil
	}

}

func getNumericId(u uuid.UUID) int {
	var val big.Int
	val.SetBytes(u[:])             // turn the 16 UUID bytes into a 128bit number
	val.SetBytes(val.Bytes()[0:8]) // keep the most significant 64bits (8byte)

	numID := int(val.Uint64())

	return numID
}

// getLegacyNumericId implements the Dynatrace logic for transforming a "legacy" random UUID to a numeric ID
func getLegacyNumericId(u uuid.UUID) int {
	uuidBytes := u[:] // work on the UUID's 16 bytes directly

	var b [10]byte // create 10 byte array from which an 8 byte numeric ID will be created

	// fill byte 0-5 with the UUID's most significant bytes (big-endian)
	b[0] = uuidBytes[0]
	b[1] = uuidBytes[1]
	b[2] = uuidBytes[2]
	b[3] = uuidBytes[3]
	b[4] = uuidBytes[4]
	b[5] = uuidBytes[5]

	// fill byte 6-9 with the last 4 bytes of the UUID/ending "integer" of the UUID's least significant LSB
	b[6] = uuidBytes[12]
	b[7] = uuidBytes[13]
	b[8] = uuidBytes[14]
	b[9] = uuidBytes[15]

	numericId := byteToInt64(b)

	numericId = zigZagDecode(numericId)

	return int(numericId)
}

// byteToInt64 transforms the given byte array version of a numeric ID to an integer
// using variable-length quantity encoding.
// This implementation matches how Dynatrace transforms the byte array to a "long" directly.
// Note the explicit use of int32 and int64 to match Java's int and long bit-lengths.
// see: https://en.wikipedia.org/wiki/Variable-length_quantity
// see:
func byteToInt64(b [10]byte) int64 {
	nextByte := int32(b[0])
	if nextByte >= 0 && nextByte <= 128 {
		return int64(nextByte)
	}
	res := int64(nextByte & 0x7F)
	isContinuationBitSet := true
	shift := 0
	i := 0
	for isContinuationBitSet {
		i++
		nextByte = int32(b[i])
		isContinuationBitSet = (nextByte & 0x80) != 0
		nextByte &= 0x7F
		shift += 7
		res |= int64(nextByte) << shift
	}

	return res
}

// zigZagDecode an int64
// zig-zag decode shifts the input number by 1 including its sign bit, then applies an XOR mask
// see: https://developers.google.com/protocol-buffers/docs/encoding
func zigZagDecode(num int64) int64 {
	return int64(uint64(num)>>1) ^ -(num & 1)
}
