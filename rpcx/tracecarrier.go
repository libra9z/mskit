package rpcx

import (
	"encoding/base64"
	"github.com/libra9z/mskit/metadata"
	"strings"
)

// A type that conforms to opentracing.TextMapReader and
// opentracing.TextMapWriter.
type metadataReaderWriter struct {
	*metadata.MD
}

func (w metadataReaderWriter) Set(key, val string) {
	key = strings.ToLower(key)
	if strings.HasSuffix(key, "-bin") {
		val = base64.StdEncoding.EncodeToString([]byte(val))
	}
	(*w.MD)[key] =  val
}

func (w metadataReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for k, vals := range *w.MD {
		if err := handler(k, vals); err != nil {
			return err
		}
	}
	return nil
}
