package util

import (
	"archive/zip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

/**
 * EncryptCompress
 * 암호화된 파일을 압축하여 zip 파일로 생성
 *
 * @param targets []string
 * @return string
 * @return error
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.07.10
 */
func EncryptCompress(targets []string) (string, error) {

	currentTimeStr := time.Now().Format("2006-01-02_150405")

	// 압축 파일 생성
	zipFile, err := os.Create(fmt.Sprintf("data/log_%s.zip", currentTimeStr))
	if err != nil {
		log.Error(fmt.Errorf("failed to create archive: %v", err))
		return "", err
	}
	defer zipFile.Close()

	// zip.Writer 생성
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	cnt := 0
	for _, target := range targets {

		// 압축할 파일 열기
		fileToCompress, err := os.Open(target)
		if err != nil {
			log.Error(fmt.Errorf("failed to open %s: %v", target, err))
			continue
		}
		defer fileToCompress.Close()

		// 파일 내용 읽기
		plaintext, err := ioutil.ReadAll(fileToCompress)
		if err != nil {
			log.Error(fmt.Errorf("failed to read file_to_compress.txt: %v", err))
			continue
		}

		// 파일 내용 암호화
		block, err := aes.NewCipher(encryptKey)
		if err != nil {
			log.Error(fmt.Errorf("failed to create cipher: %v", err))
			continue
		}

		ciphertext := make([]byte, aes.BlockSize+len(plaintext))
		iv := ciphertext[:aes.BlockSize]
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			log.Error(fmt.Errorf("failed to read random iv: %v", err))
			continue
		}

		stream := cipher.NewCFBEncrypter(block, iv)
		stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

		// 암호화된 내용을 hex로 변환
		encryptedHex := hex.EncodeToString(ciphertext)

		// 파일 경로와 파일 이름 분리
		_, fileName := filepath.Split(target)

		// zip 내에 파일 정보 생성
		f, err := zipWriter.Create(fileName)
		if err != nil {
			log.Error(fmt.Errorf("failed to create %s in zip: %v", fileName, err))
			continue
		}

		// 암호화된 내용을 zip에 쓰기
		_, err = f.Write([]byte(encryptedHex))
		if err != nil {
			log.Error(fmt.Errorf("failed to write encrypted data to zip: %v", err))
			continue
		}

		cnt++
	}

	if cnt == 0 {
		return "", fmt.Errorf("no files to compress")
	}

	log.Info(fmt.Sprintf("%d files compressed", cnt))

	return zipFile.Name(), nil
}

/**
 * DecryptUnzip
 * 암호화된 zip 파일을 복호화하여 압축 해제
 *
 * @param zipFilePath string
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.07.10
 */
func DecryptUnzip(zipFilePath string) {
	// Open the zip file
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		log.Error(fmt.Errorf("failed to open zip file: %v", err))
		return
	}
	defer r.Close()

	// Iterate through the files in the zip archive
	cnt := 0
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			log.Error(fmt.Errorf("failed to open file %s: %v", f.Name, err))
			continue
		}

		// Read the file data
		encryptedHex, err := ioutil.ReadAll(rc)
		if err != nil {
			log.Error(fmt.Errorf("failed to read file data: %v", err))
			continue
		}
		rc.Close()

		// Decode the hex string
		ciphertext, err := hex.DecodeString(string(encryptedHex))
		if err != nil {
			log.Error(fmt.Errorf("failed to decode hex string: %v", err))
			continue
		}

		// Create a new cipher block
		block, err := aes.NewCipher(encryptKey)
		if err != nil {
			log.Error(fmt.Errorf("failed to create cipher: %v", err))
			continue
		}

		// Check if the ciphertext length is sufficient
		if len(ciphertext) < aes.BlockSize {
			log.Error(fmt.Errorf("ciphertext too short"))
			continue
		}

		// Separate the IV and the ciphertext
		iv := ciphertext[:aes.BlockSize]
		ciphertext = ciphertext[aes.BlockSize:]

		// Decrypt the ciphertext
		stream := cipher.NewCFBDecrypter(block, iv)
		stream.XORKeyStream(ciphertext, ciphertext)

		// Write the decrypted data to a file
		err = ioutil.WriteFile(f.Name, ciphertext, 0644)
		if err != nil {
			log.Error(fmt.Errorf("failed to write decrypted data to file: %v", err))
		}

		cnt++
	}

	log.Info(fmt.Sprintf("%d files decrypted", cnt))

}
