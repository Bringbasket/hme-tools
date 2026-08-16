// S3 兼容对象存储客户端。
package backup

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"gokeep/server/internal/common"
	"gokeep/server/internal/ent"
	"gokeep/server/internal/ent/sysconfig"
)

// S3Config S3 兼容存储配置（sys.backup.s3.*）
type S3Config struct {
	Endpoint       string
	Region         string
	Bucket         string
	Prefix         string
	AccessKey      string
	SecretKey      string
	ForcePathStyle bool
}

// LoadS3Config 从参数配置读取 S3 配置
func LoadS3Config(ctx context.Context, client *ent.Client) S3Config {
	return S3Config{
		Endpoint:       ConfigValue(ctx, client, "sys.backup.s3.endpoint"),
		Region:         ConfigValue(ctx, client, "sys.backup.s3.region"),
		Bucket:         ConfigValue(ctx, client, "sys.backup.s3.bucket"),
		Prefix:         ConfigValue(ctx, client, "sys.backup.s3.prefix"),
		AccessKey:      ConfigValue(ctx, client, "sys.backup.s3.accessKey"),
		SecretKey:      ConfigValue(ctx, client, "sys.backup.s3.secretKey"),
		ForcePathStyle: strings.EqualFold(ConfigValue(ctx, client, "sys.backup.s3.forcePathStyle"), "true"),
	}
}

// ConfigValue 读取 sys_config 值（缺失返回空）
func ConfigValue(ctx context.Context, client *ent.Client, key string) string {
	cfg, err := client.SysConfig.Query().Where(sysconfig.KeyEQ(key)).Only(ctx)
	if err != nil {
		return ""
	}
	return cfg.Value
}

func (c S3Config) configured() bool {
	return c.Endpoint != "" && c.Bucket != "" && c.AccessKey != ""
}

func (c S3Config) keyOf(name string) string {
	return strings.TrimRight(c.Prefix, "/") + "/" + name
}

// newClient 创建 minio 客户端
func (c S3Config) newClient() (*minio.Client, error) {
	if !c.configured() {
		return nil, common.NewBizError(http.StatusServiceUnavailable, "S3 存储未配置（请先在数据备份里填写端点/桶/密钥）")
	}
	endpoint := c.Endpoint
	secure := true
	if strings.HasPrefix(endpoint, "http://") {
		secure = false
		endpoint = strings.TrimPrefix(endpoint, "http://")
	} else if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.TrimPrefix(endpoint, "https://")
	}
	region := c.Region
	if region == "" {
		region = "us-east-1"
	}
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""),
		Secure: secure,
		Region: region,
	})
}

// TestConnection 测试 S3 连通性
func (c S3Config) TestConnection(ctx context.Context) error {
	mc, err := c.newClient()
	if err != nil {
		return err
	}
	exists, err := mc.BucketExists(ctx, c.Bucket)
	if err != nil {
		return common.NewBizError(http.StatusServiceUnavailable, "S3 连接失败: "+err.Error())
	}
	if !exists {
		return common.NewBizError(http.StatusBadRequest, "存储桶不存在: "+c.Bucket)
	}
	return nil
}

// Upload 上传对象
func (c S3Config) Upload(ctx context.Context, name string, r io.Reader, size int64) error {
	mc, err := c.newClient()
	if err != nil {
		return err
	}
	_, err = mc.PutObject(ctx, c.Bucket, c.keyOf(name), r, size, minio.PutObjectOptions{
		ContentType: "application/gzip",
	})
	if err != nil {
		return fmt.Errorf("上传备份失败: %w", err)
	}
	return nil
}

// Download 下载对象（调用方负责 Close）
func (c S3Config) Download(ctx context.Context, name string) (io.ReadCloser, error) {
	mc, err := c.newClient()
	if err != nil {
		return nil, err
	}
	obj, err := mc.GetObject(ctx, c.Bucket, c.keyOf(name), minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("下载备份失败: %w", err)
	}
	return obj, nil
}

// Stat 对象大小
func (c S3Config) Stat(ctx context.Context, name string) (int64, error) {
	mc, err := c.newClient()
	if err != nil {
		return 0, err
	}
	info, err := mc.StatObject(ctx, c.Bucket, c.keyOf(name), minio.StatObjectOptions{})
	if err != nil {
		return 0, fmt.Errorf("查询备份对象失败: %w", err)
	}
	return info.Size, nil
}

// Delete 删除对象
func (c S3Config) Delete(ctx context.Context, name string) error {
	mc, err := c.newClient()
	if err != nil {
		return err
	}
	return mc.RemoveObject(ctx, c.Bucket, c.keyOf(name), minio.RemoveObjectOptions{})
}
