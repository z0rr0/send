package encrypt

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestText(t *testing.T) {
	const (
		secret    = "secret"
		plainText = "some text"
	)
	m1, err := Text(secret, plainText)
	if err != nil {
		t.Fatal(err)
	}
	if (m1.Value == "") || (m1.Value == plainText) {
		t.Errorf("failed value=%s", m1.Value)
	}
	// decrypt
	m2 := &Msg{Value: m1.Value, Salt: m1.Salt, Hash: m1.Hash}
	decrypted, err := DecryptText(secret, m2)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != plainText {
		t.Errorf("failed decrypted=%s", decrypted)
	}
}

func TestFile(t *testing.T) {
	const (
		secret    = "secret"
		plainText = "some text"
	)
	var src, dst bytes.Buffer

	_, err := src.WriteString(plainText)
	if err != nil {
		t.Fatal(err)
	}
	tmpFile, err := ioutil.TempFile(os.TempDir(), "test_file")
	if err != nil {
		t.Fatal(err)
	}
	fileName := tmpFile.Name()
	defer func() {
		if e := os.Remove(fileName); e != nil {
			t.Error(e)
		}
	}()
	m1, err := File(secret, &src, fileName)
	if err != nil {
		t.Fatal(err)
	}
	err = tmpFile.Close()
	if err != nil {
		t.Fatal(err)
	}
	// decrypt
	m2 := &Msg{Salt: m1.Salt}
	err = DecryptFile(secret, m2, &dst, fileName)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := dst.ReadString('\n')
	if err != nil && err != io.EOF {
		t.Error(err)
	}
	if decrypted != plainText {
		t.Errorf("failed decrypted value=%s", decrypted)
	}
}

func BenchmarkSalt(b *testing.B) {
	for n := 0; n < b.N; n++ {
		salt, err := Salt()
		if err != nil {
			b.Error(err)
		}
		if n := len(salt); n == 0 {
			b.Errorf("failed salt length=%d", n)
		}
	}
}

func BenchmarkKey(b *testing.B) {
	const secret = "secret"
	salt, err := Salt()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		k, h := Key(secret, salt)
		if n := len(k); n != aesKeyLength {
			b.Errorf("failed key length=%d", n)
		}
		if n := len(h); n != hashLength {
			b.Errorf("failed hash length=%d", n)
		}
	}
}
