package main

import "github.com/tendermint/go-amino"

var cdc = amino.NewCodec()

func init() {
	cdc.RegisterInterface((*Resolver)(nil),nil)
	cdc.RegisterConcrete(LocalResolver{}, "LocalResolver", nil)
}

