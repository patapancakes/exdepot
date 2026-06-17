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
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/klauspost/compress/zlib"
)

type Block struct {
	Offset uint64 `json:"offset"`
	Length uint64 `json:"length"`

	data io.Reader
}

var ErrBlockNotPrepared = errors.New("block not prepared")

func (b *Block) Read(dst []byte) (int, error) {
	if b.Length == 0 {
		return 0, io.EOF
	}

	if b.data == nil {
		return 0, ErrBlockNotPrepared
	}

	return b.data.Read(dst)
}

func (b *Block) Close() error {
	closer, ok := b.data.(io.Closer)
	if !ok {
		return nil
	}

	err := closer.Close()
	if err != nil {
		return err
	}

	b.data = nil

	return nil
}

func (b *Block) Prepare(block cipher.Block, src io.Reader, mode Mode) error {
	// why do zero-length blocks exist?
	if b.Length == 0 {
		return nil
	}

	b.data = src

	// zlib buffer sizes if encrypted, not used
	var encSize, decSize uint32
	if mode == EncryptedCompressed {
		err := read(b.data, binary.LittleEndian, &encSize, &decSize)
		if err != nil {
			return fmt.Errorf("failed to read value: %w", err)
		}
	}

	// decrypt
	if mode == EncryptedCompressed || mode == Encrypted {
		if block == nil {
			return fmt.Errorf("missing decryption key")
		}

		b.data = &cipher.StreamReader{S: cipher.NewCFBDecrypter(block, make([]byte, 0x10)), R: b.data}
	}

	// decompress
	if mode == EncryptedCompressed || mode == Compressed {
		zr, err := zlib.NewReader(b.data)
		if err != nil {
			return fmt.Errorf("failed to create zlib reader: %w", err)
		}

		b.data = zr
	}

	return nil
}
