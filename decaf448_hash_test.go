// SPDX-License-Group: MIT
//
// Copyright (C) 2022 Daniel Bourdrez. All Rights Reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree or at
// https://spdx.org/licenses/MIT.html

package decaf448_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/bytemare/decaf448"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type vectors struct {
	Group   string   `json:"group"`
	Hash    string   `json:"hash"`
	Vectors []vector `json:"vectors"`
}

type vector struct {
	Input  string `json:"in"`
	Output string `json:"out"`
}

func (v *vector) checkMappingToGroup(t *testing.T) []byte {
	in, err := hex.DecodeString(v.Input)
	if err != nil {
		t.Fatal(err)
	}

	e := decaf448.NewGroupElement()
	e.OneWayMap(in)
	encoded := e.Encode()

	out, err := hex.DecodeString(v.Output)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, encoded) {
		t.Fatalf("map to group failed\n\twant: %v\n\tgot : %v", out, encoded)
	}

	return encoded
}

func (v *vector) checkSerDe(t *testing.T, encoded []byte) {
	e := decaf448.NewGroupElement()
	d := e.Decode(encoded)
	re := d.Encode()

	if !bytes.Equal(encoded, re) {
		t.Fatalf("serde failed\n\twant: %v\n\tgot : %v", encoded, re)
	}
}

func (v *vector) run(t *testing.T) {
	// Test 1: check mapping input to the group
	encoded := v.checkMappingToGroup(t)

	// Test 2: check whether encode/decoding yields the same result
	v.checkSerDe(t, encoded)
}

func TestHashToCurve25519(t *testing.T) {
	if err := filepath.Walk("vectors",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}
			file, errOpen := os.Open(path)
			if errOpen != nil {
				t.Fatal(errOpen)
			}

			defer file.Close()

			val, errRead := ioutil.ReadAll(file)
			if errRead != nil {
				t.Fatal(errRead)
			}

			var v vectors
			errJSON := json.Unmarshal(val, &v)
			if errJSON != nil {
				t.Fatal(errJSON)
			}

			for _, vc := range v.Vectors {
				t.Run("", vc.run)
			}

			return nil
		}); err != nil {
		t.Fatalf("error opening vector files: %v", err)
	}
}
