package client

type Request struct {
	Domain    string
	QueryType uint16
	NoLazy    bool
}
