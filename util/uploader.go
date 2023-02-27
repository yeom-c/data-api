package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/yeom-c/data-api/app"
	"github.com/yeom-c/data-api/database"
)

type Uploader struct {
	File            []byte
	FileSize        int64
	FileType        string
	FileName        string
	UploadedTableId int32
	Url             string
	S3Config        S3Config
}

type S3Config struct {
	AccessKeyId     string
	SecretAccessKey string
	Region          string
	Bucket          string
	UploadPath      string
	Url             string
}

func (u *Uploader) ToS3(overwrite, writeTable bool) error {
	if u.FileSize > 50000000 {
		return errors.New("최대 파일크기 50MB 초과")
	}

	credentialsProvider := credentials.NewStaticCredentialsProvider(u.S3Config.AccessKeyId, u.S3Config.SecretAccessKey, "")
	cfg, _ := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(credentialsProvider),
		config.WithRegion(u.S3Config.Region),
	)
	s3Client := s3.NewFromConfig(cfg)
	s3Uploader := manager.NewUploader(s3Client)

	fileName := u.FileName
	if !overwrite {
		fileName = fmt.Sprintf("%v_%s", time.Now().Unix(), fileName)
	}

	path := u.S3Config.UploadPath
	if app.Config().Env != "production" {
		path = fmt.Sprintf("env_%s/%s/%s", app.Config().Env, app.Config().EnvUser, path)
	}
	path = fmt.Sprintf("%s/%s", path, fileName)

	// upload S3
	_, err := s3Uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(u.S3Config.Bucket),
		Key:    aws.String(path),
		Body:   bytes.NewReader(u.File),
	})
	if err != nil {
		return err
	}
	u.Url = fmt.Sprintf("%s/%s", u.S3Config.Url, path)

	// insert upload table
	if writeTable {
		upload := database.Upload{
			FileSize: int32(u.FileSize),
			FileType: u.FileType,
			FileName: u.FileName,
			Url:      u.Url,
		}
		_, err = database.Database().DataConn.Insert(&upload)
		if err != nil {
			return err
		}
		u.UploadedTableId = upload.Id
	}

	return nil
}
