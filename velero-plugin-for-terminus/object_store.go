/*
Copyright 2017, 2019 the Velero contributors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"io"
	"os"
	"path"
	"time"

	"github.com/ngaut/log"
	"github.com/sirupsen/logrus"

	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"regexp"
	"strings"

	veleroplugin "github.com/vmware-tanzu/velero/pkg/plugin/framework"
)

const (
	OriginStr = "volumeId"
	TargetStr = "VolumeId"
	Workspace = "/tmp/velero-restore/"
)

// ObjectStore represents an object storage entity
type ObjectStore struct {
	log             logrus.FieldLogger
	encryptionKeyID string
	root            string
	privateKey      []byte
}

// newObjectStore init ObjectStore
func newObjectStore(logger logrus.FieldLogger) *ObjectStore {
	return &ObjectStore{log: logger, root: Root}
}

func (o *ObjectStore) Init(config map[string]string) error {
	if err := veleroplugin.ValidateObjectStoreConfigKeys(config); err != nil {
		return err
	}
	path := filepath.Join(o.root, config["bucket"], config["prefix"])
	return os.MkdirAll(path, 0755)
}

// PutObject put objects to oss bucket
func (o *ObjectStore) PutObject(bucket, key string, body io.Reader) error {
	var p = path.Join(o.root, bucket, key)
	dir := filepath.Dir(p)
	o.log.Infof("<plugin> PutObject bucket: %s, key: %s, root: %s, dir: %s", bucket, key, o.root, dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		o.log.Errorf("<plugin> PutObject mkdir error %v, dir: %s", err, dir)
		return err
	}

	file, err := os.Create(p)
	if err != nil {
		o.log.Errorf("<plugin> PutObject create file error %v, file: %s", err, p)
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, body)
	if err != nil {
		o.log.Errorf("<plugin> PutObject copy file error %v, file: %s", err, p)
	}
	log.Infof("<plugin> Done %s", p)

	return err
}

// ObjectExists check if object exist
func (o *ObjectStore) ObjectExists(bucket, key string) (bool, error) {
	o.log.Infof("<plugin> ObjectExists bucket: %s, key: %s, root: %s", bucket, key, o.root)
	var root = path.Join(o.root, bucket, key)
	root = path.Dir(root)

	_, err := os.Stat(root)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// GetObject get objects from a bucket
func (o *ObjectStore) GetObject(bucket, key string) (io.ReadCloser, error) {
	o.log.Infof("<plugin> GetObject bucket: %s, key: %s, root: %s", bucket, key, o.root)
	var p = path.Join(o.root, bucket, key)
	_, err := os.Stat(p)
	if err == nil {
		file, err := os.Open(p)
		if err != nil {
			o.log.Errorf("<plugin> GetObject open file error %v", err)
			return nil, err
		}
		// defer file.Close() // todo

		fileReadCloser := io.ReadCloser(file)
		return fileReadCloser, nil
	}

	return nil, fmt.Errorf("file not found")
}

// ListCommonPrefixes interface
func (o *ObjectStore) ListCommonPrefixes(bucket, prefix, delimiter string) ([]string, error) {
	path := filepath.Join(o.root, bucket, prefix, delimiter)
	o.log.Infof("<plugin> ListCommonPrefixes  bucket: %s, prefix: %s, delimiter: %s, path: %s", bucket, prefix, delimiter, path)
	log := o.log.WithFields(logrus.Fields{
		"bucket":    bucket,
		"delimiter": delimiter,
		"path":      path,
		"prefix":    prefix,
	})
	log.Infof("<plugin> ListCommonPrefixes")

	infos, err := ioutil.ReadDir(path)
	if err != nil {
		o.log.Errorf("<plugin> ListCommonPrefixes read dir error %v, path: %s", err, path)
		return nil, err
	}

	var dirs []string
	for _, info := range infos {
		if info.IsDir() {
			dirs = append(dirs, info.Name())
		}
	}

	o.log.Infof("<plugin> ListCommonPrefixes result %v", dirs)

	return dirs, nil
}

// ListObjects list objects of a bucket
func (o *ObjectStore) ListObjects(bucket, prefix string) ([]string, error) {
	o.log.Infof("<plugin> ListObjects bucket: %s, prefix: %s, root: %s", bucket, prefix, o.root)
	path := o.root

	log := o.log.WithFields(logrus.Fields{
		"bucket": bucket,
		"prefix": prefix,
		"path":   path,
	})
	log.Infof("<plugin> ListObjects")

	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var objects []string
	for _, info := range infos {
		objects = append(objects, filepath.Join(prefix, info.Name()))
	}

	return objects, nil
}

// DeleteObject delete objects from oss bucket
func (o *ObjectStore) DeleteObject(bucket, key string) error {
	pos := strings.LastIndex(key, bucket)
	if pos > 0 {
		key = key[:pos-1]
	}
	p := path.Join(o.root, bucket, key)
	o.log.Infof("<plugin> DeleteObject bucket: %s, key: %s, path: %s", bucket, key, p)
	_, err := os.Stat(p)
	if err != nil {
		return nil
	}
	return os.RemoveAll(p)
}

// CreateSignedURL create a signed URL
func (o *ObjectStore) CreateSignedURL(bucket, key string, ttl time.Duration) (string, error) {
	var url = path.Join(o.root, bucket, key)
	o.log.Infof("<plugin> CreateSignedURL bucket: %s, key: %s, url: %s", bucket, key, url)
	if ttl < 0 {
		return "", fmt.Errorf("bakcup expired")
	}
	// return fmt.Sprintf("file://%s", path.Join(o.root, bucket, key)), nil

	return url, nil
}

// CheckAndConvertVolumeId convert volumeId to VolumeId in persistentvolumes json files
func CheckAndConvertVolumeId(body io.ReadCloser) (io.ReadCloser, error) {
	log.Info("<plugin> CheckAndConvertVolumeId")
	randStr := CreateCaptcha()
	tmpWorkspace := filepath.Join(Workspace, randStr)
	tmpFileName := fmt.Sprintf("%s.tar.gz", randStr)
	if _, err := CheckPathExistsAndCreate(tmpWorkspace); err != nil {
		return nil, err
	}
	if err := os.Chdir(tmpWorkspace); err != nil {
		return nil, err
	}
	fd, err := os.OpenFile(tmpFileName, os.O_WRONLY|os.O_CREATE, 0660)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	if _, err := io.Copy(fd, body); err != nil {
		return nil, err
	}

	if err := DeCompress(tmpFileName, ""); err != nil {
		return nil, err
	}

	if err := os.Remove(tmpFileName); err != nil {
		return nil, err
	}

	tmpFiles := make([]string, 0)
	err = filepath.Walk(tmpWorkspace,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if f, _ := os.Stat(path); !f.IsDir() {
				if strings.Index(path, "resources/persistentvolumes/cluster") > 0 {
					tmpFiles = append(tmpFiles, path)
				}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	for _, f := range tmpFiles {
		if err := ReplaceVolumeId(f); err != nil {
			return nil, err
		}
	}

	if err := Compress(".", tmpFileName); err != nil {
		return nil, err
	}

	f1, err := ioutil.ReadFile(tmpFileName)
	if err != nil {
		return nil, err
	}
	f2 := ioutil.NopCloser(bytes.NewReader(f1))

	if err := os.RemoveAll(tmpWorkspace); err != nil {
		return nil, err
	}

	return f2, nil
}

// CheckPathExistsAndCreate
func CheckPathExistsAndCreate(path string) (bool, error) {
	log.Infof("<plugin> CheckPathExistsAndCreate path: %s", path)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return false, err
		} else {
			return true, nil
		}
	}
	return false, nil
}

// CreateCaptcha
func CreateCaptcha() string {
	return fmt.Sprintf("%08v", rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(1000000))
}

// DeCompress
func DeCompress(tarFile, dest string) error {
	log.Infof("<plugin> DeCompress tarFile: %s, dest: %s", tarFile, dest)
	srcFile, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	gr, err := gzip.NewReader(srcFile)
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
		filename := dest + hdr.Name
		file, err := CreateFile(filename)
		if err != nil {
			return err
		}
		io.Copy(file, tr)
	}
	return nil
}

// CreateFile
func CreateFile(name string) (*os.File, error) {
	log.Infof("<plugin> CreateFile name: %s", name)
	err := os.MkdirAll(string([]rune(name)[0:strings.LastIndex(name, "/")]), 0755)
	if err != nil {
		return nil, err
	}
	return os.Create(name)
}

// ReplaceVolumeId
func ReplaceVolumeId(filePath string) error {
	log.Infof("<plugin> ReplaceVolumeId filePath: %s", filePath)
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	output := make([]byte, 0)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if ok, _ := regexp.Match(OriginStr, line); ok {
			reg := regexp.MustCompile(OriginStr)
			newByte := reg.ReplaceAll(line, []byte(TargetStr))
			output = append(output, newByte...)
			output = append(output, []byte("\n")...)
		} else {
			output = append(output, line...)
			output = append(output, []byte("\n")...)
		}
	}

	if err := writeToFile(filePath, output); err != nil {
		return err
	}
	return nil
}

// writeToFile
func writeToFile(filePath string, outPut []byte) error {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(f)
	_, err = writer.Write(outPut)
	if err != nil {
		return err
	}
	writer.Flush()
	return nil
}

// Compress
func Compress(src, dst string) error {
	log.Infof("<plugin> Compress src: %s, dst: %s", src, dst)
	fw, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer fw.Close()

	gw := gzip.NewWriter(fw)
	defer gw.Close()

	tw := tar.NewWriter(gw)

	defer tw.Close()

	return filepath.Walk(src, func(fileName string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.Index(fileName, dst) > -1 {
			return nil
		}

		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}

		hdr.Name = strings.TrimPrefix(fileName, string(filepath.Separator))

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			return nil
		}

		fr, err := os.Open(fileName)
		defer fr.Close()
		if err != nil {
			return err
		}

		if _, err := io.Copy(tw, fr); err != nil {
			return err
		}

		return nil
	})
}
