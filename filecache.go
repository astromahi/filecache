//Copyright 2014 Mahendra Kathirvel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filecache

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cache_dir = "/tmp"        // Cache directory
	expire    = 8 * time.Hour // Hours to keep the cache
)

func Set(key string, data interface{}) error {

	t1 := time.Now()
	key = regexp.MustCompile("[^a-zA-Z0-9_-]").ReplaceAllLiteralString(key, "")

	clean(key)

	file := "filecache." + key + "." + strconv.FormatInt(time.Now().Add(expire).Unix(), 10)
	fpath := filepath.Join(cache_dir, file)

	serialized, err := serialize(data)
	if err != nil {
		return err
	}

	var fmutex sync.RWMutex

	fmutex.Lock()
	fp, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer fp.Close()

	if _, err = fp.Write(serialized); err != nil {
		return err
	}
	defer fmutex.Unlock()

	t2 := time.Now()
	log.Println("Set filecache: ", key, " : ", t2.Sub(t1))

	return nil
}

func Get(key string, dst interface{}) error {

	t:= time.Now()
	key = regexp.MustCompile("[^a-zA-Z0-9_-]").ReplaceAllLiteralString(key, "")

	pattern := filepath.Join(cache_dir, "filecache."+key+".*")
	files, err := filepath.Glob(pattern)
	if len(files) < 1 || err != nil {
		return errors.New("filecache: no cache file found")
	}

	if _, err = os.Stat(files[0]); err != nil {
		return err
	}

	fp, err := os.OpenFile(files[0], os.O_RDONLY, 0400)
	if err != nil {
		return err
	}
	defer fp.Close()

	var serialized []byte
	buf := make([]byte, 1024)
	for {
		var n int
		n, err = fp.Read(buf)
		serialized = append(serialized, buf[0:n]...)
		if err != nil || err == io.EOF {
			break
		}
	}

	if err = deserialize(serialized, dst); err != nil {
		return err
	}

	for _, file := range files {
		exptime, err := strconv.ParseInt(strings.Split(file, ".")[2], 10, 64)
		if err != nil {
			return err
		}

		if exptime < time.Now().Unix() {
			if _, err = os.Stat(file); err == nil {
				os.Remove(file)
			}
		}
	}

	log.Println("Get filecache: ", key, " : ", time.Since(t))
	return nil
}

func clean(key string) {
	pattern := filepath.Join(cache_dir, "filecache."+key+".*")
	files, _ := filepath.Glob(pattern)
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			os.Remove(file)
		}
	}
}

// serialize encodes a value using binary.
func serialize(src interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(src); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// deserialize decodes a value using binary.
func deserialize(src []byte, dst interface{}) error {
	buf := bytes.NewReader(src)
	if err := gob.NewDecoder(buf).Decode(dst); err != nil {
		log.Println(dst)
		return err
	}
	return nil
}
