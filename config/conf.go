package config

import "io/fs"

type Config struct {
	Port  string
	RAMDB bool

	Files fs.FS
}
