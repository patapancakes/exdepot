/*
	Copyright (C) 2024-2025  Pancakes <patapancakes@pagefault.games>

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package gozelle

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type Mode int

const (
	Raw Mode = iota
	Compressed
	EncryptedCompressed
	Encrypted
)

type Index map[uint64]File

func ReadIndex(r io.Reader) (Index, error) {
	index := make(Index)

	for {
		var id, length, mode uint64
		err := read(r, binary.BigEndian, &id, &length, &mode)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return index, err
			}

			break
		}

		var chunks []Chunk
		for i := 0; i < int(length); i += 0x10 {
			var start, length uint64
			err := read(r, binary.BigEndian, &start, &length)
			if err != nil {
				return nil, fmt.Errorf("failed to read value: %s", err)
			}

			chunks = append(chunks, Chunk{Offset: start, Length: length})
		}

		index[id] = File{Chunks: chunks, Mode: Mode(mode)}
	}

	return index, nil
}
