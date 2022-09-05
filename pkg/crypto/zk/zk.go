package zk

/*
#cgo LDFLAGS: -L./lib -lzk
#include <stdlib.h>
#include "./lib/zk.h"
*/
import "C"
import "unsafe"

func Verify(proof, key []byte) int {
	return int(C.verify((*C.uchar)(unsafe.Pointer(&proof[0])), C.uint(len(proof)), (*C.uchar)(unsafe.Pointer(&key[0])), C.uint(len(key))))
}
