package bencode

import (
	"fmt"
	"io"
	"sort"
)

func Encode(w io.Writer, v any) error {
	return encodeValue(w, v)
}

func encodeValue(w io.Writer, v any) error {
	switch val := v.(type) {
	case string:
		return encodeString(w, val)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return encodeInt(w, val)
	case []any:
		return encodeList(w, val)
	case map[string]any:
		return encodeDict(w, val)
	default:
		return fmt.Errorf("Cannot encode type %T (%#v)", v, v)
	}
}

func encodeString(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "%d:%s", len(s), s)
	return err
}

func encodeInt(w io.Writer, i any) error {
	_, err := fmt.Fprintf(w, "i%de", i)
	return err
}

func encodeList(w io.Writer, l []any) error {
	_, err := w.Write([]byte("l"))

	if err != nil {
		return err
	}

	for _, v := range l {
		err := encodeValue(w, v)
		if err != nil {
			return err
		}
	}

	_, err = w.Write([]byte("e"))
	return err
}

func encodeDict(w io.Writer, d map[string]any) error {
	_, err := w.Write([]byte("d"))
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(d))
	for key := range d {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		err = encodeString(w, key)
		if err != nil {
			return err
		}

		err = encodeValue(w, d[key])
		if err != nil {
			return err
		}
	}

	_, err = w.Write([]byte("e"))
	return err
}
