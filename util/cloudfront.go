package util

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/yeom-c/data-api/app"
)

type CloudFront struct {
	AccessKeyId     string
	SecretAccessKey string
	Region          string
	DistributionId  string
}

func (cf *CloudFront) CreateInvalidation(patterns ...string) (*cloudfront.CreateInvalidationOutput, error) {
	items := []string{}
	for _, pattern := range patterns {
		if app.Config().Env != "production" {
			pattern = fmt.Sprintf("/env_%s/%s%s", app.Config().Env, app.Config().EnvUser, pattern)
		}
		items = append(items, *aws.String(pattern))
	}

	cfClient := cloudfront.NewFromConfig(aws.Config{
		Region:      cf.Region,
		Credentials: credentials.NewStaticCredentialsProvider(cf.AccessKeyId, cf.SecretAccessKey, ""),
	})

	result, err := cfClient.CreateInvalidation(context.Background(), &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(cf.DistributionId),
		InvalidationBatch: &types.InvalidationBatch{
			CallerReference: aws.String(fmt.Sprintf("quasar-data_%s", time.Now().Format("2006-01-02 15:04:05"))),
			Paths: &types.Paths{
				Quantity: aws.Int32(int32(len(items))),
				Items:    items,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
