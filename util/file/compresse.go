package file

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

const paxGlobalHeader = "pax_global_header"

func UnTar(gz []byte, target string) error {
	gzReader := bytes.NewReader(gz)
	gzipReader, err := gzip.NewReader(gzReader)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	extractPath := target
	var extractFile *os.File
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		extractFilePath := filepath.Join(extractPath, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err := os.MkdirAll(extractFilePath, info.Mode()); err != nil {
				return err
			}
			continue
		}
		extractFile, err = os.OpenFile(extractFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		if _, err = io.Copy(extractFile, tarReader); err != nil {
			return err
		}
	}
	defer extractFile.Close()
	return os.Remove(paxGlobalHeader)
}

func Unzip(z []byte, target string) error {
	zipReader := bytes.NewReader(z)
	r, err := zip.NewReader(zipReader, int64(len(z)))
	if err != nil {
		return err
	}
	extractPath := target
	var extractFile *os.File
	var compressedFile *os.File
	for _, file := range r.File {
		extractFilePath := filepath.Join(extractPath, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(extractFilePath, file.Mode()); err != nil {
				return err
			}
			continue
		}
		extractFile, err = os.OpenFile(extractFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		compressedFile, err := file.Open()
		if err != nil {
			return err
		}
		if _, err = io.Copy(extractFile, compressedFile); err != nil {
			return err
		}
	}
	defer func() {
		extractFile.Close()
		compressedFile.Close()
	}()
	return nil
}

func UnzipPackage(zipFile, target string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()
	os.MkdirAll(target, 0755)
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		filePath := filepath.Join(target, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, f.Mode())
		} else {
			dirPath := filepath.Dir(filePath)
			os.MkdirAll(dirPath, f.Mode())

			file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				rc.Close()
				return err
			}
			_, err = io.Copy(file, rc)
			file.Close()
			rc.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
