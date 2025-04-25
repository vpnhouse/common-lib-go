package xproxy

import "io"

type accounter struct {
	customInfo any
	reporter   Reporter
	parent     io.ReadCloser
}

func (i *accounter) Read(p []byte) (n int, err error) {
	if i.reporter != nil {
		i.reporter(i.customInfo, uint64(n))
	}

	return i.parent.Read(p)
}

func (i *accounter) Close() error {
	return i.parent.Close()
}
