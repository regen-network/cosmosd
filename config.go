package main

type Config struct {
	Genesis  Resolver            `json:"genesis"`
	Upgrades map[string]Resolver `json:"upgrades"`
}

