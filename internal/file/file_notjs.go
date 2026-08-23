// Copyright 2023 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !js

package file

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// VirtualFS is a file system consisting of the given real paths.
//
// The root directory of a VirtualFS is the deepest directory containing all the given paths.
// The root directory lists only the given paths and the directories to reach them,
// so a file next to a given path is not exposed unless it is given.
type VirtualFS struct {
	root *virtualFSNode
}

func NewVirtualFS(paths []string) (*VirtualFS, error) {
	ps, err := absPaths(paths)
	if err != nil {
		return nil, err
	}
	rootDir := commonDirOfParents(ps)

	root := &virtualFSNode{
		absPath: rootDir,
	}
	for _, absPath := range ps {
		root.add(strings.Split(entryName(rootDir, absPath), "/"), absPath)
	}
	root.fillAbsPaths()
	root.sortRecursively()

	return &VirtualFS{
		root: root,
	}, nil
}

func (v *VirtualFS) Open(name string) (fs.File, error) {
	n, rest := v.find(name)
	if n == nil {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	if !n.given {
		return &virtualDir{node: n}, nil
	}

	absPath := n.join(rest)
	// os.File should implement fs.File interface, so this should be fine even on Windows.
	// See https://cs.opensource.google/go/go/+/refs/tags/go1.23.0:src/os/file.go;l=695-710
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	return &virtualFile{
		File:    f,
		absPath: absPath,
	}, nil
}

func (v *VirtualFS) ReadDir(name string) ([]fs.DirEntry, error) {
	n, rest := v.find(name)
	if n == nil {
		return nil, &fs.PathError{
			Op:   "readdir",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	if !n.given {
		d := virtualDir{node: n}
		return d.ReadDir(-1)
	}

	f, err := os.Open(n.join(rest))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	return f.ReadDir(-1)
}

func (v *VirtualFS) ReadFile(name string) ([]byte, error) {
	n, rest := v.find(name)
	if n == nil {
		return nil, &fs.PathError{
			Op:   "readfile",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	if !n.given {
		return nil, &fs.PathError{
			Op:   "readfile",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	return os.ReadFile(n.join(rest))
}

// find returns the node for the given name, and the path elements left unresolved by the node.
// The elements are left only for a node of a given path, where the real file system takes over.
// find returns a nil node if the name is invalid or is not in the file system.
func (v *VirtualFS) find(name string) (*virtualFSNode, []string) {
	if !fs.ValidPath(name) {
		return nil, nil
	}

	if name == "." {
		return v.root, nil
	}

	n := v.root
	es := strings.Split(name, "/")
	for i, e := range es {
		if n.given {
			return n, es[i:]
		}
		idx := slices.IndexFunc(n.children, func(c *virtualFSNode) bool {
			return c.name == e
		})
		if idx < 0 {
			return nil, nil
		}
		n = n.children[idx]
	}
	return n, nil
}

// virtualFSNode is an entry of a VirtualFS.
//
// A node is either one of the given paths, or exists only to reach such nodes.
// Only the latter has children.
type virtualFSNode struct {
	name     string
	absPath  string
	given    bool
	children []*virtualFSNode
}

func (n *virtualFSNode) add(es []string, absPath string) {
	name := es[0]
	idx := slices.IndexFunc(n.children, func(c *virtualFSNode) bool {
		return c.name == name
	})
	if idx < 0 {
		n.children = append(n.children, &virtualFSNode{name: name})
		idx = len(n.children) - 1
	}
	c := n.children[idx]

	if len(es) == 1 {
		c.absPath = absPath
		c.given = true
		return
	}
	c.add(es[1:], absPath)
}

// fillAbsPaths sets the absolute path of the nodes that are not one of the given paths.
func (n *virtualFSNode) fillAbsPaths() {
	for _, c := range n.children {
		if !c.given {
			if n.absPath != "" {
				c.absPath = filepath.Join(n.absPath, c.name)
			} else {
				// The file system has no root directory, so a child of the root is a volume name.
				c.absPath = joinVolumePath(c.name, nil)
			}
		}
		c.fillAbsPaths()
	}
}

func (n *virtualFSNode) sortRecursively() {
	slices.SortFunc(n.children, func(a, b *virtualFSNode) int {
		return strings.Compare(a.name, b.name)
	})
	for _, c := range n.children {
		c.sortRecursively()
	}
}

// join returns the absolute path for the given path elements under the node.
func (n *virtualFSNode) join(es []string) string {
	return filepath.Join(append([]string{n.absPath}, es...)...)
}

// absPaths returns the given paths as absolute paths, sorted, without duplications,
// and without a path under another path.
func absPaths(paths []string) ([]string, error) {
	var ps []string
	for _, p := range paths {
		// filepath.Abs returns the working directory for an empty path.
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("file: getting an absolute path of %s failed: %w", p, err)
		}
		// Skip root directory paths on purpose. This is almost the same behavior as the Chrome browser.
		if filepath.Dir(abs) == abs {
			continue
		}
		ps = append(ps, abs)
	}

	slices.Sort(ps)
	ps = slices.Compact(ps)

	// A node of a given path must not have children. Drop paths reachable from another path.
	var res []string
	for _, absPath := range ps {
		if slices.ContainsFunc(res, func(p string) bool {
			return contains(p, absPath)
		}) {
			continue
		}
		res = append(res, absPath)
	}
	return res, nil
}

// splitVolumePath splits an absolute path into its volume name and its path elements.
func splitVolumePath(absPath string) (string, []string) {
	vol := filepath.VolumeName(absPath)
	rest := strings.TrimPrefix(absPath[len(vol):], string(filepath.Separator))
	if rest == "" {
		return vol, nil
	}
	return vol, strings.Split(rest, string(filepath.Separator))
}

// joinVolumePath is the inverse of splitVolumePath.
// vol is empty only on a system that has no volume names.
func joinVolumePath(vol string, es []string) string {
	return vol + string(filepath.Separator) + strings.Join(es, string(filepath.Separator))
}

// contains reports whether absDir is absPath or an ancestor directory of absPath.
func contains(absDir, absPath string) bool {
	dirVol, dirEs := splitVolumePath(absDir)
	pathVol, pathEs := splitVolumePath(absPath)
	if dirVol != pathVol {
		return false
	}
	if len(dirEs) > len(pathEs) {
		return false
	}
	return slices.Equal(dirEs, pathEs[:len(dirEs)])
}

// commonDirOfParents returns the deepest directory containing all the given paths.
// commonDirOfParents returns an empty string if there is no such directory,
// which can happen on Windows when the paths are on different volumes.
func commonDirOfParents(absPaths []string) string {
	if len(absPaths) == 0 {
		return ""
	}

	vol, es := splitVolumePath(filepath.Dir(absPaths[0]))
	for _, absPath := range absPaths[1:] {
		v, e := splitVolumePath(filepath.Dir(absPath))
		if v != vol {
			return ""
		}
		if len(es) > len(e) {
			es = es[:len(e)]
		}
		for i := range es {
			if es[i] != e[i] {
				es = es[:i]
				break
			}
		}
	}
	return joinVolumePath(vol, es)
}

// entryName returns the name of the given real path in a file system whose root directory is rootDir.
// If rootDir is empty, the name starts with a volume name so that the name is still unique.
func entryName(rootDir, absPath string) string {
	vol, es := splitVolumePath(absPath)

	if rootDir != "" {
		_, rootEs := splitVolumePath(rootDir)
		return path.Join(es[len(rootEs):]...)
	}

	// A volume name can contain separators e.g. `\\host\share`. Keep it as one path element.
	return path.Join(append([]string{vol}, es...)...)
}

// virtualFile is a file in a VirtualFS that reports its own path in the real file system.
type virtualFile struct {
	*os.File
	absPath string
}

func (v *virtualFile) AbsPath() string {
	return v.absPath
}

// virtualDirEntry is an entry of a VirtualFS that reports its own path in the real file system.
type virtualDirEntry struct {
	fs.DirEntry
	absPath string
}

func (v *virtualDirEntry) AbsPath() string {
	return v.absPath
}

type virtualDir struct {
	node   *virtualFSNode
	offset int
	mu     sync.Mutex
}

func (v *virtualDir) Stat() (fs.FileInfo, error) {
	name := v.node.name
	if name == "" {
		name = "."
	}
	return &virtualDirFileInfo{
		name: name,
	}, nil
}

func (v *virtualDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{
		Op:   "read",
		Path: v.node.name,
		Err:  errors.New("is a directory"),
	}
}

func (v *virtualDir) Close() error {
	return nil
}

func (v *virtualDir) ReadDir(count int) ([]fs.DirEntry, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	n := len(v.node.children) - v.offset

	if n == 0 {
		if count <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}

	if count > 0 && n > count {
		n = count
	}

	ents := make([]fs.DirEntry, n)
	for i := range ents {
		c := v.node.children[v.offset+i]
		if !c.given {
			ents[i] = &virtualDirEntry{
				DirEntry: fs.FileInfoToDirEntry(&virtualDirFileInfo{
					name: c.name,
				}),
				absPath: c.absPath,
			}
			continue
		}
		fi, err := os.Stat(c.absPath)
		if err != nil {
			if count <= 0 {
				return ents, err
			}
			return nil, err
		}
		ents[i] = &virtualDirEntry{
			DirEntry: fs.FileInfoToDirEntry(fi),
			absPath:  c.absPath,
		}
	}
	v.offset += n

	return ents, nil
}

type virtualDirFileInfo struct {
	name string
}

func (v *virtualDirFileInfo) Name() string {
	return v.name
}

func (v *virtualDirFileInfo) Size() int64 {
	return 0
}

func (v *virtualDirFileInfo) Mode() fs.FileMode {
	return 0555 | fs.ModeDir
}

func (v *virtualDirFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (v *virtualDirFileInfo) IsDir() bool {
	return true
}

func (v *virtualDirFileInfo) Sys() any {
	return nil
}
