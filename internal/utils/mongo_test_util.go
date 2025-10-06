package utils

import (
	"runtime"

	"github.com/tryvium-travels/memongo"
	"github.com/tryvium-travels/memongo/memongolog"
)

type MongoTestUtil struct {
	MongoVersion string
}

func NewMongoTestUtil() *MongoTestUtil {
	return &MongoTestUtil{
		MongoVersion: "7.0.0",
	}
}

func (db MongoTestUtil) CreateServer() (*memongo.Server, error) {
	opts := &memongo.Options{
		MongoVersion: "7.0.0",
		LogLevel:     memongolog.LogLevelInfo,
	}

	if runtime.GOARCH == "arm64" {
		if runtime.GOOS == "darwin" {
			// Only set the custom url as workaround for arm64 macs
			opts.DownloadURL = "https://fastdl.mongodb.org/osx/mongodb-macos-x86_64-7.0.0.tgz"
		}
	}

	return memongo.StartWithOptions(opts)
}
