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
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"regexp"
)

var uuidRegex = regexp.MustCompile(".*?([0-9a-fA-F]{8}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{12}).*?")

// GetNumericIDForObjectID parses the Settings Object ID of a Dynatrace Management Zone (only object with numeric IDs)
// into a numeric identifier. To achieve this it replicates the en-/decoding logic used in Dynatrace as closely as possible.
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
	if err != nil { // should never actually happen if the UUID regex matched
		return 0, fmt.Errorf("failed to parse UUID %q: %w", uuidString, err)
	}

	if u.Variant() == uuid.RFC4122 && u.Version() == 4 {
		return getLegacyNumericID(u)
	}

	return getNumericID(u), nil
}

// getNumericID implements the Dynatrace logic for transforming a "new" (non-random) UUID to a numeric ID
// by converting the UUID's most significant (big-endian) bytes into an integer
func getNumericID(u uuid.UUID) int {
	return int(binary.BigEndian.Uint64(u[0:8]))
}

// getLegacyNumericID implements the Dynatrace logic for transforming a "legacy" (variant RFC, version 4 (random)) UUID to a numeric ID
// by taking specific bytes of the UUID and decoding them as a signed variable-length integer
func getLegacyNumericID(u uuid.UUID) (int, error) {

	var b []byte
	b = u[0:6]                 // fill byte 0-5 with the UUID's most significant bytes (big-endian)
	b = append(b, u[12:16]...) // fill byte 6-9 with the last 4 bytes of the UUID/ending "integer" of the UUID's least significant LSB

	numericId, n := binary.Varint(b) // decode bytes as signed VarInt
	if n <= 0 {
		return 0, fmt.Errorf("failed to decode variable-length integer. %d/%d bytes read correctly", -n, len(b))
	}

	return int(numericId), nil
}
