package main

import (
	"io"
	"io/fs"
	"time"
)

type emptyDirFS struct{}

func (emptyDirFS) Open(name string) (fs.File, error) {
	if name == "." || name == "" {
		return emptyDirFile{}, nil
	}
	return nil, fs.ErrNotExist
}

type emptyDirFile struct{}

func (emptyDirFile) Stat() (fs.FileInfo, error) { return emptyDirInfo{}, nil }
func (emptyDirFile) Read([]byte) (int, error)   { return 0, io.EOF }
func (emptyDirFile) Close() error               { return nil }
func (emptyDirFile) ReadDir(int) ([]fs.DirEntry, error) {
	return nil, nil
}

type emptyDirInfo struct{}

func (emptyDirInfo) Name() string       { return "." }
func (emptyDirInfo) Size() int64        { return 0 }
func (emptyDirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o755 }
func (emptyDirInfo) ModTime() time.Time { return time.Time{} }
func (emptyDirInfo) IsDir() bool        { return true }
func (emptyDirInfo) Sys() any           { return nil }
