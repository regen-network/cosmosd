package main

type Resolver interface {
	BinaryPath() string
	Sha256Hash() string
}

