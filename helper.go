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
	"encoding/binary"
	"io"
)

func readUint32List(r io.Reader, num int) ([]uint32, error) {
	out := make([]uint32, num)

	for i := range num {
		b := make([]byte, 4)
		_, err := r.Read(b)
		if err != nil {
			return nil, err
		}

		out[i] = binary.LittleEndian.Uint32(b)
	}

	return out, nil
}

func readUint64List(r io.Reader, num int) ([]uint64, error) {
	out := make([]uint64, num)

	for i := range num {
		b := make([]byte, 8)
		_, err := r.Read(b)
		if err != nil {
			return nil, err
		}

		out[i] = binary.BigEndian.Uint64(b)
	}

	return out, nil
}
