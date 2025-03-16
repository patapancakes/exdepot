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
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type Chunk struct {
	Offset uint64 `json:"offset"`
	Length uint64 `json:"length"`

	data   io.Reader
	closer io.Closer
}

var ErrChunkNotPrepared = errors.New("chunk not prepared")

func (c Chunk) Read(dst []byte) (int, error) {
	if c.Length == 0 {
		return 0, io.EOF
	}

	if c.data == nil {
		return 0, ErrChunkNotPrepared
	}

	n, err := c.data.Read(dst)
	if err != nil {
		return n, err
	}

	return n, nil
}

func (c Chunk) Close() error {
	if c.closer == nil {
		return nil
	}

	c.closer.Close()

	return nil
}

func (c *Chunk) Prepare(key []byte, src io.ReaderAt, mode Mode) error {
	// why do zero-length chunks exist?
	if c.Length == 0 {
		return nil
	}

	chunk := make([]byte, c.Length)
	_, err := src.ReadAt(chunk, int64(c.Offset))
	if err != nil {
		return fmt.Errorf("failed to read data: %s", err)
	}

	c.data = bytes.NewReader(chunk)

	// zlib buffer sizes if encrypted, not used
	var encSize, decSize uint32
	if mode == EncryptedCompressed {
		err = read(c.data, binary.LittleEndian, &encSize, &decSize)
		if err != nil {
			return fmt.Errorf("failed to read value: %s", err)
		}
	}

	// decrypt
	if mode == EncryptedCompressed || mode == Encrypted {
		if key == nil {
			return fmt.Errorf("missing decryption key")
		}

		block, err := aes.NewCipher(key)
		if err != nil {
			return fmt.Errorf("failed to create aes cipher: %s", err)
		}

		c.data = &cipher.StreamReader{S: cipher.NewCFBDecrypter(block, make([]byte, 0x10)), R: c.data}
	}

	// decompress
	if mode == EncryptedCompressed || mode == Compressed {
		zr, err := zlib.NewReader(c.data)
		if err != nil {
			return fmt.Errorf("failed to create zlib reader: %s", err)
		}

		c.closer = zr
		c.data = zr
	}

	return nil
}
