/*
	Copyright (C) 2024-2026  Pancakes <patapancakes@pagefault.games>

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

type Mode uint64

const (
	Raw Mode = iota
	Compressed
	EncryptedCompressed
	Encrypted
)

type Index map[uint64]*File

func ReadIndex(r io.Reader) (Index, error) {
	index := make(Index)

	for {
		var f File
		var id, length uint64
		err := read(r, binary.BigEndian, &id, &length, &f.Mode)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, err
			}

			break
		}

		br := io.LimitReader(r, int64(length))

		for {
			var b Block
			err := read(br, binary.BigEndian, &b.Offset, &b.Length)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					return nil, fmt.Errorf("failed to read value: %w", err)
				}

				break
			}

			f.Blocks = append(f.Blocks, &b)
		}

		index[id] = &f
	}

	return index, nil
}
