// Copyright 2021-2025 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/veraison/corim/corim"
	"github.com/veraison/corim/cots"
	"github.com/veraison/swid"
)

func filesList(files, dirs []string, ext string) []string {
	var l []string

	for _, file := range files {
		if _, err := fs.Stat(file); err == nil {
			if filepath.Ext(file) == ext {
				l = append(l, file)
			}
		}
	}

	for _, dir := range dirs {
		filesInfo, err := afero.ReadDir(fs, dir)
		if err != nil {
			continue
		}

		for _, fileInfo := range filesInfo {
			if !fileInfo.IsDir() && filepath.Ext(fileInfo.Name()) == ext {
				l = append(l, filepath.Join(dir, fileInfo.Name()))
			}
		}
	}

	return l
}

type FromCBORLoader interface {
	FromCBOR([]byte) error
}

func printJSONFromCBOR(fcl FromCBORLoader, cbor []byte, heading string) error {
	var (
		err error
		j   []byte
	)

	if err = fcl.FromCBOR(cbor); err != nil {
		return fmt.Errorf("CBOR decoding failed: %w", err)
	}

	indent := "  "
	if j, err = json.MarshalIndent(fcl, "", indent); err != nil {
		return fmt.Errorf("JSON encoding failed: %w", err)
	}

	fmt.Println(heading)
	fmt.Println(string(j))

	return nil
}

func printComidWithExtensions(cbor []byte, profile *corim.Profile, heading string) error {
	var (
		err error
		j   []byte
	)
	c, err := corim.UnmarshalComidFromCBOR(cbor, profile)
	if err != nil {
		return fmt.Errorf("error decoding CoMID from CBOR: %w", err)
	}

	indent := "  "
	if j, err = json.MarshalIndent(c, "", indent); err != nil {
		return fmt.Errorf("JSON encoding failed: %w", err)
	}

	fmt.Println(heading)
	fmt.Println(string(j))
	return nil
}

func printComid(cbor []byte, profile *corim.Profile, heading string) error {
	return printComidWithExtensions(cbor, profile, heading)
}

func printCoswid(cbor []byte, heading string) error {
	return printJSONFromCBOR(&swid.SoftwareIdentity{}, cbor, heading)
}

func printCots(cbor []byte, heading string) error {
	return printJSONFromCBOR(&cots.ConciseTaStore{}, cbor, heading)
}

func makeFileName(dirName, baseName, ext string) string {
	return filepath.Join(
		dirName,
		filepath.Base(
			strings.TrimSuffix(
				baseName,
				filepath.Ext(baseName),
			),
		)+ext,
	)
}
