package torrent

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"magnetik/utils/bencode"
	"os"
	"time"
)

var (
	ErrInvalidTorrentFile = errors.New("invalid torrent file")
	ErrInvalidInfoDict    = errors.New("invalid info dictionary")
)

func calculateInfoHash(info map[string]any) ([20]byte, error) {
	var buf bytes.Buffer

	err := bencode.Encode(&buf, info)
	if err != nil {
		return [20]byte{}, fmt.Errorf("failed to calculate info hash: %w", err)
	}

	return sha1.Sum(buf.Bytes()), nil
}

func ParseFromFile(path string) (*TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	data, err := bencode.Decode(file)
	if err != nil {
		return nil, err
	}

	return Parse(data)
}

func Parse(data any) (*TorrentFile, error) {
	d, ok := data.(map[string]any)
	if !ok {
		return nil, ErrInvalidTorrentFile
	}

	t := &TorrentFile{}

	announceVal, ok := d["announce"]
	if !ok {
		return nil, fmt.Errorf("%w: missing announce url", ErrInvalidTorrentFile)
	}

	t.Announce = announceVal.(string)

	if announceListVal, ok := d["announce-list"]; ok {
		announceList, ok := announceListVal.([]any)
		if !ok {
			return nil, fmt.Errorf("%w: announce-list is not a list", ErrInvalidTorrentFile)
		}

		t.AnnounceList = make([][]string, len(announceList))
		for i, tier := range announceList {
			tierList, ok := tier.([]any)
			if !ok {
				return nil, fmt.Errorf("%w: announce-list tier is not a list", ErrInvalidInfoDict)
			}

			t.AnnounceList[i] = make([]string, len(tierList))
			for j, tracker := range tierList {
				trackerURL, ok := tracker.(string)
				if !ok {
					return nil, fmt.Errorf("%w: tracker URL is not a string", ErrInvalidTorrentFile)
				}

				t.AnnounceList[i][j] = trackerURL
			}
		}

	}

	if creationDateVal, ok := d["creation date"]; ok {
		creationDate, ok := creationDateVal.(int64) // unix time in seconds
		if !ok {
			return nil, fmt.Errorf("%w: creation date not in unix seconds", ErrInvalidTorrentFile)
		}
		t.CreationDate = time.Unix(creationDate, 0)
	}

	if commentVal, ok := d["comment"]; ok {
		comment, ok := commentVal.(string)
		if !ok {
			return nil, fmt.Errorf("%w: comment is not a string", ErrInvalidTorrentFile)
		}
		t.Comment = comment
	}

	if createdByVal, ok := d["created by"]; ok {
		createdBy, ok := createdByVal.(string)
		if !ok {
			return nil, fmt.Errorf("%w: created by is not a string", ErrInvalidTorrentFile)
		}
		t.CreatedBy = createdBy
	}

	if encodingVal, ok := d["encoding"]; ok {
		// I will probably ignore it
		encoding, ok := encodingVal.(string)
		if !ok {
			return nil, fmt.Errorf("%w: encoding format is not a string", ErrInvalidTorrentFile)
		}
		t.Encoding = encoding
	}

	infoVal, ok := d["info"]
	if !ok {
		return nil, fmt.Errorf("%w: no info dictionary", ErrInvalidTorrentFile)
	}

	info, ok := infoVal.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: info is not a dictionary", ErrInvalidTorrentFile)
	}

	// now let's parse the info dict
	err := parseInfoDict(info, &t.Info)
	if err != nil {
		return nil, err
	}

	infoHash, err := calculateInfoHash(info)
	if err != nil {
		return nil, err
	}

	t.InfoHash = infoHash

	// get the sha-1 hash of every piece
	pieces := t.Info.Pieces
	if len(pieces)%20 != 0 {
		return nil, fmt.Errorf("%w: pieces field length is not multiple of 20", ErrInvalidInfoDict)
	}
	numPieces := len(pieces) / 20
	hashes := make([][20]byte, numPieces)

	for i := range numPieces {
		copy(hashes[i][:], pieces[i*20:(i+1)*20])
	}

	t.PiecesHash = hashes

	return t, nil
}

func parseInfoDict(info map[string]any, infoDict *InfoDict) error {
	pieceLengthVal, ok := info["piece length"]
	if !ok {
		return fmt.Errorf("%w: no piece length", ErrInvalidInfoDict)
	}
	pieceLength, ok := pieceLengthVal.(int64)
	if !ok {
		return fmt.Errorf("%w: piece length is not an integer", ErrInvalidInfoDict)
	}
	infoDict.PieceLength = pieceLength

	// oof way too verbose
	piecesVal, ok := info["pieces"]
	if !ok {
		return fmt.Errorf("%w: no pieces in info dictionary", ErrInvalidInfoDict)
	}

	pieces, ok := piecesVal.([]byte)
	if !ok {
		return fmt.Errorf("%w: pieces is not a byte string", ErrInvalidInfoDict)
	}
	infoDict.Pieces = pieces

	if privateVal, ok := info["private"]; ok {
		private, ok := privateVal.(int64)
		if !ok {
			return fmt.Errorf("%w: private is not an integer", ErrInvalidInfoDict)
		}
		infoDict.Private = (private == 1)
	}

	if _, ok := info["files"]; ok {
		infoDict.IsDirectory = true
	} else if _, ok := info["length"]; ok {
		infoDict.IsDirectory = false
	} else {
		return fmt.Errorf("%w: no files nor length key", ErrInvalidInfoDict)
	}

	nameVal, ok := info["name"]
	if !ok {
		return fmt.Errorf("%w: file to be downloaded has no file or folder name", ErrInvalidInfoDict)
	}
	name, ok := nameVal.(string)
	if !ok {
		return fmt.Errorf("%w: invalid filename", ErrInvalidInfoDict)
	}
	infoDict.Name = name

	if infoDict.IsDirectory {
		// multi-file
		files, ok := info["files"].([]any)
		if !ok {
			return fmt.Errorf("%w: files is not a list", ErrInvalidInfoDict)
		}

		// ok now parse files list
		infoDict.Files = make([]FileDict, len(files))
		for i, fileVal := range files {
			fileDict, ok := fileVal.(map[string]any)
			if !ok {
				return fmt.Errorf("%w: invalid file dictionary", ErrInvalidInfoDict)
			}

			fileLengthVal, ok := fileDict["length"]
			if !ok {
				return fmt.Errorf("%w: no file length found", ErrInvalidInfoDict)
			}

			fileLength, ok := fileLengthVal.(int64)
			if !ok {
				return fmt.Errorf("%w: file length is not an integer", ErrInvalidInfoDict)
			}

			infoDict.Files[i].Length = fileLength

			// then file path
			pathListVal, ok := fileDict["path"]
			if !ok {
				return fmt.Errorf("%w: path of file is missing", ErrInvalidInfoDict)
			}

			pathList, ok := pathListVal.([]any)
			if !ok {
				return fmt.Errorf("%w: path list is invalid", ErrInvalidInfoDict)
			}

			infoDict.Files[i].Path = make([]string, len(pathList))
			for j, pathVal := range pathList {
				path, ok := pathVal.(string)
				if !ok {
					return fmt.Errorf("%w: path is not a string", ErrInvalidInfoDict)
				}
				infoDict.Files[i].Path[j] = path
			}
		}
	} else {
		// single-file
		length, ok := info["length"].(int64)
		if !ok {
			return fmt.Errorf("%w: length is not an integer", ErrInvalidInfoDict)
		}

		infoDict.Length = length
	}

	return nil
}
