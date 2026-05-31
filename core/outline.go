package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SceneInfo is display-oriented metadata for one scene, with an accurate word
// count read from the prose body. It backs the corkboard and outliner views.
//
// Like everything in core it is plain data: no UI types cross this boundary.
type SceneInfo struct {
	Path       string
	Name       string // file name without the .md extension
	Title      string // Meta.Title, or Name when no title is set
	Synopsis   string
	Status     Status
	WordCount  int // counted from the body, independent of any stored word_count
	WordTarget int
	DraftOrder int
}

// ReadSceneInfo loads a scene's display info, best-effort. A read or parse
// failure still returns a usable SceneInfo (Path/Name/Title set from the file
// name) so the views can show the file rather than silently dropping it.
func ReadSceneInfo(path string) SceneInfo {
	name := strings.TrimSuffix(filepath.Base(path), ".md")
	info := SceneInfo{Path: path, Name: name, Title: name}

	sc, err := LoadScene(path)
	if err != nil {
		return info
	}
	info.Status = sc.Meta.Status
	info.Synopsis = sc.Meta.Synopsis
	info.WordTarget = sc.Meta.WordTarget
	info.DraftOrder = sc.Meta.DraftOrder
	if sc.Meta.Title != "" {
		info.Title = sc.Meta.Title
	}
	info.WordCount = WordCount(sc.Body)
	return info
}

// FolderScenes returns SceneInfo for the direct .md children of dir (not
// recursive), ordered for reading: scenes with an explicit draft_order come
// first in ascending order, then the rest alphabetically. Hidden files are
// skipped. The directory must be readable.
func FolderScenes(dir string) ([]SceneInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var infos []SceneInfo
	for _, e := range entries {
		name := e.Name()
		if len(name) > 0 && name[0] == '.' {
			continue // skip hidden
		}
		if e.IsDir() || filepath.Ext(name) != ".md" {
			continue
		}
		infos = append(infos, ReadSceneInfo(filepath.Join(dir, name)))
	}

	SortScenesForReading(infos)
	return infos, nil
}

// SortScenesForReading orders scenes by draft_order (explicit orders first,
// ascending), falling back to case-insensitive name. It sorts in place. The
// same ordering will drive compile/export later, so it lives in one place.
func SortScenesForReading(infos []SceneInfo) {
	sort.SliceStable(infos, func(i, j int) bool {
		a, b := infos[i], infos[j]
		ao, bo := a.DraftOrder, b.DraftOrder
		// A scene with an explicit order sorts before one without.
		if (ao == 0) != (bo == 0) {
			return ao != 0
		}
		if ao != bo {
			return ao < bo
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
}
