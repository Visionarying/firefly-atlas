package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/scrypt"
)

// Firefly encrypted backup format:
//
//	magic "FFBK1" | salt(16) | baseNonce(8) | (uint32 length | AES-GCM chunk)* | uint32(0)
//
// Every 1 MiB chunk is independently authenticated with AES-256-GCM. The key is
// derived with scrypt instead of a plain SHA-256 hash, so a weak password is
// considerably harder to brute force.
const (
	backupMagic     = "FFBK1"
	backupChunkSize = 1 << 20
)

var errNotEncrypted = errors.New("不是 Firefly 加密备份文件")

func deriveBackupKey(password string, salt []byte) ([]byte, error) {
	if password == "" {
		return nil, errors.New("加密密码不能为空")
	}
	return scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func EncryptBackupStream(dst io.Writer, src io.Reader, password string) error {
	salt := make([]byte, 16)
	baseNonce := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}
	if _, err := io.ReadFull(rand.Reader, baseNonce); err != nil {
		return err
	}
	key, err := deriveBackupKey(password, salt)
	if err != nil {
		return err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return err
	}

	if _, err := dst.Write([]byte(backupMagic)); err != nil {
		return err
	}
	if _, err := dst.Write(salt); err != nil {
		return err
	}
	if _, err := dst.Write(baseNonce); err != nil {
		return err
	}

	buf := make([]byte, backupChunkSize)
	var index uint32
	for {
		n, readErr := io.ReadFull(src, buf)
		if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
			return readErr
		}
		if n > 0 {
			nonce := make([]byte, 12)
			copy(nonce, baseNonce)
			binary.BigEndian.PutUint32(nonce[8:], index)
			sealed := gcm.Seal(nil, nonce, buf[:n], nil)
			var length [4]byte
			binary.BigEndian.PutUint32(length[:], uint32(len(sealed)))
			if _, err := dst.Write(length[:]); err != nil {
				return err
			}
			if _, err := dst.Write(sealed); err != nil {
				return err
			}
			index++
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
	}
	var end [4]byte
	_, err = dst.Write(end[:])
	return err
}

func DecryptBackupStream(dst io.Writer, src io.Reader, password string) error {
	magic := make([]byte, len(backupMagic))
	if _, err := io.ReadFull(src, magic); err != nil {
		return err
	}
	if string(magic) != backupMagic {
		return errNotEncrypted
	}
	salt := make([]byte, 16)
	baseNonce := make([]byte, 8)
	if _, err := io.ReadFull(src, salt); err != nil {
		return err
	}
	if _, err := io.ReadFull(src, baseNonce); err != nil {
		return err
	}
	key, err := deriveBackupKey(password, salt)
	if err != nil {
		return err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return err
	}
	var index uint32
	for {
		var length [4]byte
		if _, err := io.ReadFull(src, length[:]); err != nil {
			if err == io.EOF {
				return errors.New("备份文件不完整")
			}
			return err
		}
		size := binary.BigEndian.Uint32(length[:])
		if size == 0 {
			return nil
		}
		if size > 2*backupChunkSize {
			return errors.New("备份分块长度异常")
		}
		sealed := make([]byte, size)
		if _, err := io.ReadFull(src, sealed); err != nil {
			return err
		}
		nonce := make([]byte, 12)
		copy(nonce, baseNonce)
		binary.BigEndian.PutUint32(nonce[8:], index)
		plain, err := gcm.Open(nil, nonce, sealed, nil)
		if err != nil {
			return errors.New("解密失败：密码错误或备份文件已损坏")
		}
		if _, err := dst.Write(plain); err != nil {
			return err
		}
		index++
	}
}

func IsEncryptedBackup(path string) (bool, error) {
	f, err := openFile(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	head := make([]byte, len(backupMagic))
	if _, err := io.ReadFull(f, head); err != nil {
		return false, err
	}
	return string(head) == backupMagic, nil
}

func openFile(path string) (*os.File, error) {
	return os.Open(path)
}

func createFile(path string) (*os.File, error) {
	return os.Create(path)
}

func encryptFileToFile(srcPath, dstPath, password string) error {
	src, err := openFile(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := createFile(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	if err := EncryptBackupStream(dst, src, password); err != nil {
		return fmt.Errorf("加密备份失败: %w", err)
	}
	return dst.Sync()
}

func decryptFileToFile(srcPath, dstPath, password string) error {
	src, err := openFile(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := createFile(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	if err := DecryptBackupStream(dst, src, password); err != nil {
		return err
	}
	return dst.Sync()
}
