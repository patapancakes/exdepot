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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

type Keys map[int][]byte

type KeyFile struct {
	Keys map[string]string `json:"keys"`
}

func readKeys(path string) (Keys, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open keys file: %s", err)
	}

	defer file.Close()

	index, err := keysFromReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys file: %s", err)
	}

	return index, nil
}

func keysFromReader(r io.Reader) (Keys, error) {
	var keyfile KeyFile
	err := json.NewDecoder(r).Decode(&keyfile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode json: %s", err)
	}

	keys := make(Keys)

	for depot, key := range keyfile.Keys {
		depotInt, err := strconv.Atoi(depot)
		if err != nil {
			return nil, fmt.Errorf("failed to decode depot id: %s", err)
		}

		keyBytes, err := hex.DecodeString(key)
		if err != nil {
			return nil, fmt.Errorf("failed to decode key: %s", err)
		}

		keys[depotInt] = keyBytes
	}

	return keys, nil
}
