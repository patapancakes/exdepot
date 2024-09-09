/*
	Copyright (C) 2024  Pancakes <patapancakes@pagefault.games>

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

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

type Mode int

const (
	Raw Mode = iota
	Compressed
	EncryptedCompressed
	Encrypted
)

type Index map[int]IndexEntry

type IndexEntry struct {
	Chunks []Chunk
	Mode   Mode
}

type Chunk struct {
	Start  uint64
	Length uint64
}

func readIndex(storagedir string, depot int) (Index, error) {
	file, err := os.Open(fmt.Sprintf("%s/%d.index", storagedir, depot))
	if err != nil {
		return nil, fmt.Errorf("failed to open index file: %s", err)
	}

	defer file.Close()

	index, err := indexFromReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read index: %s", err)
	}

	return index, nil
}

func indexFromReader(r io.Reader) (Index, error) {
	index := make(Index)

	for {
		v, err := readUint64List(r, 3)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return index, err
			}

			break
		}

		id := v[0]
		length := v[1]
		mode := v[2]

		var chunks []Chunk
		for i := 0; i < int(length); i += 0x10 {
			v, err := readUint64List(r, 2)
			if err != nil {
				return nil, fmt.Errorf("failed to read value: %s", err)
			}

			start := v[0]
			length := v[1]

			chunks = append(chunks, Chunk{Start: start, Length: length})
		}

		index[int(id)] = IndexEntry{
			Chunks: chunks,
			Mode:   Mode(mode),
		}
	}

	return index, nil
}
