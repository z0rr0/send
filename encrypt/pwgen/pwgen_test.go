package pwgen

import "testing"

func TestNew(t *testing.T) {
	var p string
	cases := []int{0, 1, 2, 5, 10, 20, 50}
	for i, c := range cases {
		p = New(c)
		t.Log(p)
		if n := len(p); n != c {
			t.Errorf("failed password length [%v] %v != %v", i, n, c)
		}
	}
}

func BenchmarkNew(b *testing.B) {
	// go test -count=1 -v -cover -benchmem -bench=. github.com/z0rr0/send/encrypt/pwgen
	const m = 20
	var p string
	for i := 0; i < b.N; i++ {
		p = New(m)
		if n := len(p); n != m {
			b.Errorf("failed password length [%v] %v != %v", i, n, m)
		}
	}
}
