//go:build !linux

package main

import "log"

func watchTouch(cfg *Config, out chan<- touch) {
	log.Printf("touch: only supported on the Kindle")
}
