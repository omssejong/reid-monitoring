package util

import (
	"mime/multipart"
	"sync/atomic"
)

var isPatch int32 = 0

const (
	IDLE int32 = iota
	PROGRESSING
)

func IsPatchProgress() int32 {
	return atomic.LoadInt32(&isPatch)
}

func StartPatch() {
	atomic.AddInt32(&isPatch, 1)
}

func EndPatch() {
	atomic.AddInt32(&isPatch, -1)
}

func DecryptZipFile(encryptFile *multipart.FileHeader) {
	//key := "THVLpyzTz+FrFECppoye8Q/ex3q2mBPYK7y8DWNh3Uc="
	//iv := "jpu/MSko4neDr7Uv8o3opg=="
	//
	//parseKey := make([]byte, 0)
	//parseIv := make([]byte, 0)
	//parseKey, err := base64.StdEncoding.AppendDecode(parseKey, []byte(key))
	//if err != nil {
	//	panic(err)
	//}
	//parseIv, err = base64.StdEncoding.AppendDecode(parseIv, []byte(iv))
	//if err != nil {
	//	panic(err)
	//}
	//
	//
	//block, err := aes.NewCipher(parseKey)
	//if err != nil {
	//	panic(err)
	//}
	//
	//
	//cipherData := raw
	//if len(cipherData)%aes.BlockSize != 0 {
	//	panic("not decrypt")
	//}
	//mode := cipher.NewCBCDecrypter(block, iv)
	//mode.CryptBlocks(cipherData, cipherData)
	//fmt.Println(string(cipherData))
	//err = os.WriteFile("index.txt", cipherData, 0777)
	//if err != nil {
	//	panic(err)
	//}
	//return cipherData
}
