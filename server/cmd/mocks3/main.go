// mocks3：极简 S3 兼容 mock 服务（仅本地开发测试用）
// 忽略 SigV4 签名，实现 minio-go 备份流程所需的 BucketExists/PutObject/GetObject/StatObject/RemoveObject
// 用法：go run ./cmd/mocks3 -addr :9000 -dir ./tmp/mocks3
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	addr := flag.String("addr", ":9000", "listen address")
	dir := flag.String("dir", "./tmp/mocks3", "storage dir")
	flag.Parse()
	if err := os.MkdirAll(*dir, 0755); err != nil {
		panic(err)
	}
	s := &server{root: *dir}
	fmt.Printf("mocks3 listening on %s (storage %s)\n", *addr, *dir)
	if err := http.ListenAndServe(*addr, s); err != nil {
		panic(err)
	}
}

type server struct{ root string }

// decodeAWSCunked 解码 aws-chunked 流式上传分帧（minio-go 的 streaming signature）
// 非 chunked 的原始字节原样返回
func decodeAWSCunked(raw []byte) []byte {
	idx := 0
	for idx < len(raw) {
		// 找行首
		nl := bytesIndex(raw, idx, '\n')
		if nl < 0 {
			break
		}
		header := strings.TrimSuffix(string(raw[idx:nl]), "\r")
		if !strings.Contains(header, ";chunk-signature=") {
			// 非 chunked 编码，直接原样返回
			return raw
		}
		sizeHex := strings.SplitN(header, ";", 2)[0]
		size := int64(0)
		fmt.Sscanf(sizeHex, "%x", &size)
		if size == 0 {
			break
		}
		start := nl + 1
		end := start + int(size)
		if end > len(raw) {
			break
		}
		idx = end + 2 // 跳过数据后的 \r\n
	}
	// 已确认是 chunked：重新解析拼数据
	var out []byte
	idx = 0
	for idx < len(raw) {
		nl := bytesIndex(raw, idx, '\n')
		if nl < 0 {
			break
		}
		header := strings.TrimSuffix(string(raw[idx:nl]), "\r")
		if !strings.Contains(header, ";chunk-signature=") {
			break
		}
		sizeHex := strings.SplitN(header, ";", 2)[0]
		var size int64
		fmt.Sscanf(sizeHex, "%x", &size)
		if size == 0 {
			break
		}
		start := nl + 1
		end := start + int(size)
		if end > len(raw) {
			break
		}
		out = append(out, raw[start:end]...)
		idx = end + 2
	}
	return out
}

func bytesIndex(b []byte, from int, c byte) int {
	for i := from; i < len(b); i++ {
		if b[i] == c {
			return i
		}
	}
	return -1
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(p, "/", 2)
	bucket := parts[0]
	key := ""
	if len(parts) == 2 {
		key = parts[1]
	}
	if bucket == "" {
		// ListBuckets
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Buckets></Buckets></ListAllMyBucketsResult>`)
		return
	}
	objPath := filepath.Join(s.root, bucket, filepath.FromSlash(key))
	switch r.Method {
	case http.MethodHead:
		if key == "" {
			// bucket exists
			w.WriteHeader(http.StatusOK)
			return
		}
		if fi, err := os.Stat(objPath); err == nil {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
			w.Header().Set("ETag", fmt.Sprintf("\"%x\"", fi.ModTime().UnixNano()))
			w.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	case http.MethodPut:
		_ = os.MkdirAll(filepath.Dir(objPath), 0755)
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		data := decodeAWSCunked(raw)
		if err := os.WriteFile(objPath, data, 0644); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		fi, err := os.Stat(objPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		f, err := os.Open(objPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
		w.Header().Set("ETag", fmt.Sprintf("\"%x\"", fi.ModTime().UnixNano()))
		w.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, f)
	case http.MethodDelete:
		_ = os.Remove(objPath)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
