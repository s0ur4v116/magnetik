package torrent

import (
	"path/filepath"
	"time"
)

// https://wiki.theory.org/BitTorrentSpecification#Metainfo_File_Structure

type TorrentFile struct {
	Announce     string
	AnnounceList [][]string
	CreationDate time.Time
	Comment      string
	CreatedBy    string
	Encoding     string
	Info         InfoDict
	InfoHash     [20]byte
	PiecesHash   [][20]byte
}

type InfoDict struct {
	PieceLength int64
	Pieces      []byte
	Private     bool
	Name        string
	Length      int64
	Files       []FileDict
	IsDirectory bool
}

type FileDict struct {
	Length int64
	Path   []string
}

func (t *TorrentFile) TotalLength() int64 {
	if !t.Info.IsDirectory {
		return t.Info.Length
	}

	length := int64(0)
	for _, file := range t.Info.Files {
		length += file.Length
	}

	return length
}

func (t *TorrentFile) NumPieces() int {
	return len(t.PiecesHash)
}

func (t *TorrentFile) PieceSize(index int) int64 {
	if index < 0 || index >= t.NumPieces() {
		return 0
	}

	// only last one is special
	if index < t.NumPieces()-1 {
		return t.Info.PieceLength
	}

	lastPieceSize := t.TotalLength() % t.Info.PieceLength
	if lastPieceSize == 0 {
		return t.Info.PieceLength
	}
	return lastPieceSize
}

func (t *TorrentFile) FilePathForPiece(index int) []string {
	if index < 0 || index >= t.NumPieces() {
		return nil
	}

	if !t.Info.IsDirectory {
		return []string{t.Info.Name}
	}

	pieceOffset := int64(index) * t.Info.PieceLength
	pieceEnd := pieceOffset + t.PieceSize(index)

	currentOffset := int64(0)
	var result []string

	for _, file := range t.Info.Files {
		fileStart := currentOffset
		fileEnd := fileStart + file.Length
		if fileEnd > pieceOffset && fileStart < pieceEnd {
			path := filepath.Join(append([]string{t.Info.Name}, file.Path...)...)
			result = append(result, path)
		}

		currentOffset = fileEnd
	}

	return result
}
