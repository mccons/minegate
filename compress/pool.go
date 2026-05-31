package compress

import (
	"compress/zlib"
	"io"
	"sync"

	kzlib "github.com/klauspost/compress/zlib"
)

// ZlibWriterPool is a sync.Pool wrapper for zlib.Writer.
type ZlibWriterPool struct {
	pool  sync.Pool
	level int
}

func NewZlibWriterPool(level int) *ZlibWriterPool {
	return &ZlibWriterPool{level: level}
}

func (zwp *ZlibWriterPool) Get(w io.Writer) (*kzlib.Writer, error) {
	if v := zwp.pool.Get(); v != nil {
		zw := v.(*kzlib.Writer)
		zw.Reset(w)
		return zw, nil
	}
	return kzlib.NewWriterLevel(w, zwp.level)
}

func (zwp *ZlibWriterPool) Put(zw *kzlib.Writer) {
	zw.Close()
	zwp.pool.Put(zw)
}

// ZlibReaderPool is a sync.Pool wrapper for zlib.Reader.
type ZlibReaderPool struct {
	pool sync.Pool
}

func NewZlibReaderPool() *ZlibReaderPool {
	return &ZlibReaderPool{}
}

func (zrp *ZlibReaderPool) Get(r io.Reader) (io.ReadCloser, error) {
	if v := zrp.pool.Get(); v != nil {
		zr := v.(io.ReadCloser)
		if rst, ok := zr.(zlib.Resetter); ok {
			if err := rst.Reset(r, nil); err != nil {
				return nil, err
			}
			return zr, nil
		}
	}
	return zlib.NewReader(r)
}

func (zrp *ZlibReaderPool) Put(zr io.ReadCloser) {
	zr.Close()
	zrp.pool.Put(zr)
}
