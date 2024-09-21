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

package gozelle

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
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
	Chunks []Chunk `json:"chunks"`
	Mode   Mode    `json:"mode"`
}

type Chunk struct {
	Offset uint64 `json:"offset"`
	Length uint64 `json:"length"`
}

func IndexFromFile(storagedir string, depot int) (Index, error) {
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

			chunks = append(chunks, Chunk{Offset: start, Length: length})
		}

		index[int(id)] = IndexEntry{
			Chunks: chunks,
			Mode:   Mode(mode),
		}
	}

	return index, nil
}

func (e IndexEntry) WriteInto(key []byte, src io.ReaderAt, dst io.Writer) error {
	for _, cd := range e.Chunks {
		// why do zero-length chunks exist?
		if cd.Length == 0 {
			continue
		}

		chunk := make([]byte, cd.Length)
		_, err := src.ReadAt(chunk, int64(cd.Offset))
		if err != nil {
			return fmt.Errorf("failed to read data: %s", err)
		}

		var r io.Reader

		r = bytes.NewReader(chunk)

		// zlib buffer sizes if encrypted, not used
		//var encSize, decSize uint32
		if e.Mode == EncryptedCompressed {
			_, err := readUint32List(r, 2)
			if err != nil {
				return fmt.Errorf("failed to read value: %s", err)
			}

			//encSize = v[0] // unused
			//decSize = v[1] // unused
		}

		// decrypt
		if e.Mode == EncryptedCompressed || e.Mode == Encrypted {
			if key == nil {
				return fmt.Errorf("missing decryption key")
			}

			d := make([]byte, cd.Length)
			_, err = r.Read(d)
			if err != nil {
				return fmt.Errorf("failed to read data: %s", err)
			}

			c, err := aes.NewCipher(key)
			if err != nil {
				return fmt.Errorf("failed to create aes cipher: %s", err)
			}

			cipher.NewCFBDecrypter(c, make([]byte, 0x10)).XORKeyStream(d, d)

			if e.Mode == Encrypted {
				d = d[:cd.Length]
			}

			r = bytes.NewReader(d)
		}

		// decompress
		if e.Mode == Compressed || e.Mode == EncryptedCompressed {
			zr, err := zlib.NewReader(r)
			if err != nil {
				return fmt.Errorf("failed to create zlib reader: %s", err)
			}

			defer zr.Close()

			r = zr
		}

		_, err = io.Copy(dst, r)
		if err != nil {
			return fmt.Errorf("failed to write data to output file: %s", err)
		}
	}

	return nil
}
