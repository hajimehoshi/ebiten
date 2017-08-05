package ui

import "sync"

var runebuffer []rune

var rblock = new(sync.Mutex)
