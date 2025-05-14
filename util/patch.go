package util

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

func checkSha256Sum(src []byte, srcHash string) error {
	srcHash = strings.ToLower(srcHash)
	sum := sha256.New()
	_, err := sum.Write(src)
	if err != nil {
		return err
	}
	genHash := fmt.Sprintf("%x", sum.Sum(nil))
	genHash = strings.ToLower(genHash)
	if !strings.EqualFold(srcHash, genHash) {
		return errors.New("not match hash")
	} else {
		return nil
	}
}

func decryptZipFile(encryptFile []byte) ([]byte, error) {
	key := "THVLpyzTz+FrFECppoye8Q/ex3q2mBPYK7y8DWNh3Uc="
	iv := "jpu/MSko4neDr7Uv8o3opg=="

	parseKey := make([]byte, 0)
	parseIv := make([]byte, 0)
	parseKey, err := base64.StdEncoding.AppendDecode(parseKey, []byte(key))
	if err != nil {
		return nil, err
	}
	parseIv, err = base64.StdEncoding.AppendDecode(parseIv, []byte(iv))
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(parseKey)
	if err != nil {
		return nil, err
	}

	cipherData := encryptFile
	if len(cipherData)%aes.BlockSize != 0 {
		return nil, errors.New("not decrypt")
	}
	mode := cipher.NewCBCDecrypter(block, parseIv)
	mode.CryptBlocks(cipherData, cipherData)

	return cipherData, nil
}

func unzipPatchFile() error {
	zipfile, err := zip.OpenReader("temp/patch.zip")
	if err != nil {
		log.Error(err)
		return err
	}
	defer zipfile.Close()
	for _, f := range zipfile.File {
		log.Info(fmt.Sprintf("zip content file name: %s", f.Name)) // debug log
		rc, fopenErr := f.Open()
		if fopenErr != nil {
			log.Error(fopenErr)
			return fopenErr
		}

		buffer := bytes.Buffer{}
		_, readErr := buffer.ReadFrom(rc)
		if readErr != nil {
			log.Error(readErr)
			return readErr
		}
		saveFileErr := os.WriteFile("temp/"+f.Name, buffer.Bytes(), 0755)
		if saveFileErr != nil {
			log.Error(saveFileErr)
			return saveFileErr
		}
		rc.Close()
	}
	return nil
}

func executePatch() error {
	//now := time.Now()
	//today := now.Format("2006-01-02")
	//patchLogFileName := fmt.Sprintf("patch_%s.log", today)
	//args := []string{">>", "/usr/local/oms/omeye/omeye2/log/back/" + patchLogFileName}
	//cmd := exec.Command("temp/patch_script.sh", args...)
	cmd := exec.Command("temp/patch_script.sh")
	//buffer := bytes.NewReader([]byte(configs.SC.Setting.UserPassword))
	buffer := strings.NewReader(configs.SC.Setting.UserPassword)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error(err)
		return err
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Error(err)
		return err
	}
	defer stderr.Close()

	cmdStdoutBuffer := bytes.Buffer{}
	cmdStderrBuffer := bytes.Buffer{}
	defer func() {
		log.Info(fmt.Sprintf("patch log script: %s", string(cmdStdoutBuffer.Bytes())))
		log.Info(fmt.Sprintf("patch log err: %s", string(cmdStderrBuffer.Bytes())))
	}()
	cmd.Stdin = buffer

	runErr := cmd.Start()
	if runErr != nil {
		log.Error(runErr)
		return runErr
	}

	_, readErr := cmdStdoutBuffer.ReadFrom(stdout)
	if readErr != nil {
		log.Error(readErr)
		return readErr
	}
	_, errPipeReadErr := cmdStderrBuffer.ReadFrom(stderr)
	if errPipeReadErr != nil {
		log.Error(errPipeReadErr)
		return errPipeReadErr
	}

	waitErr := cmd.Wait()
	if waitErr != nil {
		log.Error(waitErr)
		return waitErr
	}
	log.Info("patch success")
	return nil
}
