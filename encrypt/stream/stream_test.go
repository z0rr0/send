package stream

import (
	"bytes"
	"io"
	"testing"
)

func buildKey(k []byte) []byte {
	const size = 32
	var n int
	key := make([]byte, size)

	if m := len(k); m < size {
		n = m
	} else {
		n = size
	}
	for i := 0; i < n; i++ {
		key[i] = k[i]
	}
	return key
}

func TestEncrypt(t *testing.T) {
	const secret = "secret stream content"
	var src, dst bytes.Buffer

	key := buildKey([]byte("abc"))
	_, err := src.WriteString(secret)
	if err != nil {
		t.Fatal(err)
	}
	err = Encrypt(&src, &dst, key)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := dst.ReadString('\n')
	if err != nil && err != io.EOF {
		t.Error(err)
	}
	if encrypted == secret {
		t.Errorf("failed encrypted value=%s", encrypted)
	}
	// decrypt
	src.Reset()
	dst.Reset()
	_, err = src.WriteString(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	err = Decrypt(&src, &dst, key)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := dst.ReadString('\n')
	if err != nil && err != io.EOF {
		t.Error(err)
	}
	if decrypted != secret {
		t.Errorf("failed decrypted value=%s", decrypted)
	}
}

func BenchmarkEncrypt(b *testing.B) {
	const secret = "secret stream content"
	var (
		value    string
		key      = buildKey([]byte("abc"))
		src, dst bytes.Buffer
	)
	for n := 0; n < b.N; n++ {
		_, err := src.WriteString(secret)
		if err != nil {
			b.Fatal(err)
		}
		err = Encrypt(&src, &dst, key)
		if err != nil {
			b.Fatal(err)
		}
		value, _ = dst.ReadString('\n')
		// decrypt
		src.Reset()
		dst.Reset()
		_, err = src.WriteString(value)
		if err != nil {
			b.Fatal(err)
		}
		err = Decrypt(&src, &dst, key)
		if err != nil {
			b.Fatal(err)
		}
		value, _ = dst.ReadString('\n')
		if value != secret {
			b.Errorf("failed decrypted value=%s", value)
		}
	}
}
