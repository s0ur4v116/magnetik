package bencode

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"
)

var (
	ErrInvalidBencode = errors.New("invalid bencode format")
	ErrIntegerFormat  = errors.New("invalid integer format")
)

func Decode(r io.Reader) (any, error) {
	br := bufio.NewReader(r)
	return decodeNext(br)
}

func decodeNext(r *bufio.Reader) (any, error) {
	b, err := r.Peek(1)
	if err != nil {
		return nil, err
	}

	switch {
	case b[0] >= '0' && b[0] <= '9':
		return decodeString(r)
	case b[0] == 'i':
		return decodeInt(r)
	case b[0] == 'l':
		return decodeList(r)
	case b[0] == 'd':
		return decodeDict(r)
	default:
		return nil, ErrInvalidBencode
	}
}

func decodeString(r *bufio.Reader) (string, error) {
	lengthStr, err := readUntil(r, ':')

	if err != nil {
		return "", err
	}

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", err
	}

	stringBuf := make([]byte, length)
	_, err = io.ReadFull(r, stringBuf)

	if err != nil {
		return "", err
	}

	return string(stringBuf), nil
}

func readUntil(r *bufio.Reader, delim byte) (string, error) {
	var res []byte
	for {
		b, err := r.ReadByte()

		if err != nil {
			return "", err
		}

		if b == delim {
			break
		}

		res = append(res, b)
	}

	return string(res), nil
}

func decodeInt(r *bufio.Reader) (int64, error) {
	_, err := r.ReadByte()

	if err != nil {
		return 0, err
	}

	numStr, err := readUntil(r, 'e')

	if err != nil {
		return 0, err
	}

	if len(numStr) > 1 && numStr[0] == '0' {
		return 0, ErrIntegerFormat
	}

	if len(numStr) > 1 && strings.HasPrefix(numStr, "-0") {
		return 0, ErrIntegerFormat
	}

	num, err := strconv.ParseInt(numStr, 10, 64)

	if err != nil {
		return 0, err
	}

	return num, nil
}

func decodeList(r *bufio.Reader) ([]any, error) {
	_, err := r.ReadByte()

	if err != nil {
		return nil, err
	}

	var list []any

	for {
		b, err := r.Peek(1)

		if err != nil {
			return nil, err
		}

		if b[0] == 'e' {
			_, err := r.ReadByte()
			return list, err
		}

		item, err := decodeNext(r)

		if err != nil {
			return nil, err
		}

		list = append(list, item)
	}
}

func decodeDict(r *bufio.Reader) (map[string]any, error) {
	_, err := r.ReadByte()

	if err != nil {
		return nil, err
	}

	dict := make(map[string]any)

	for {
		b, err := r.Peek(1)

		if err != nil {
			return nil, err
		}

		if b[0] == 'e' {
			_, err := r.ReadByte()
			return dict, err
		}

		key, err := decodeString(r)

		if err != nil {
			return nil, err
		}

		value, err := decodeNext(r)

		if err != nil {
			return nil, err
		}

		dict[key] = value
	}
}
