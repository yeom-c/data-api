package app

import (
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

var cfgOnce sync.Once
var cfgInstance *config

type config struct {
	Env                            string        `mapstructure:"ENV"`
	EnvUser                        string        `mapstructure:"ENV_USER"`
	ServerPort                     int           `mapstructure:"SERVER_PORT"`
	CorsHost                       string        `mapstructure:"CORS_HOST"`
	SqlShow                        bool          `mapstructure:"SQL_SHOW"`
	DbDriver                       string        `mapstructure:"DB_DRIVER"`
	DbConn                         string        `mapstructure:"DB_CONN"`
	DbStaticDataGenDriver          string        `mapstructure:"DB_STATIC_DATA_GEN_DRIVER"`
	DbStaticDataGenConn            string        `mapstructure:"DB_STATIC_DATA_GEN_CONN"`
	DbStaticDataLocalDriver        string        `mapstructure:"DB_STATIC_DATA_LOCAL_DRIVER"`
	DbStaticDataLocalConn          string        `mapstructure:"DB_STATIC_DATA_LOCAL_CONN"`
	DbStaticDataTestDriver         string        `mapstructure:"DB_STATIC_DATA_TEST_DRIVER"`
	DbStaticDataTestConn           string        `mapstructure:"DB_STATIC_DATA_TEST_CONN"`
	DbStaticDataDevDriver          string        `mapstructure:"DB_STATIC_DATA_DEV_DRIVER"`
	DbStaticDataDevConn            string        `mapstructure:"DB_STATIC_DATA_DEV_CONN"`
	DbStaticDataStagingDriver      string        `mapstructure:"DB_STATIC_DATA_STAGING_DRIVER"`
	DbStaticDataStagingConn        string        `mapstructure:"DB_STATIC_DATA_STAGING_CONN"`
	DbStaticDataProductionDriver   string        `mapstructure:"DB_STATIC_DATA_PRODUCTION_DRIVER"`
	DbStaticDataProductionConn     string        `mapstructure:"DB_STATIC_DATA_PRODUCTION_CONN"`
	AuthTokenSymmetricKey          string        `mapstructure:"AUTH_TOKEN_SYMMETRIC_KEY"`
	AuthTokenDuration              time.Duration `mapstructure:"AUTH_TOKEN_DURATION"`
	AuthGoogleClientId             string        `mapstructure:"AUTH_GOOGLE_CLIENT_ID"`
	AuthGoogleClientSecret         string        `mapstructure:"AUTH_GOOGLE_CLIENT_SECRET"`
	AwsUserFrontendAccessKeyId     string        `mapstructure:"AWS_USER_FRONTEND_ACCESS_KEY_ID"`
	AwsUserFrontendSecretAccessKey string        `mapstructure:"AWS_USER_FRONTEND_SECRET_ACCESS_KEY"`
	AwsS3CdnRegion                 string        `mapstructure:"AWS_S3_CDN_REGION"`
	AwsS3CdnBucket                 string        `mapstructure:"AWS_S3_CDN_BUCKET"`
	AwsS3CdnUrl                    string        `mapstructure:"AWS_S3_CDN_URL"`
	AwsCloudfrontCdnDistributionId string        `mapstructure:"AWS_CLOUDFRONT_CDN_DISTRIBUTION_ID"`
	AwsS3UploadRegion              string        `mapstructure:"AWS_S3_UPLOAD_REGION"`
	AwsS3UploadBucket              string        `mapstructure:"AWS_S3_UPLOAD_BUCKET"`
	AwsS3UploadRootPath            string        `mapstructure:"AWS_S3_UPLOAD_ROOT_PATH"`
	AwsS3UploadUrl                 string        `mapstructure:"AWS_S3_UPLOAD_URL"`
	SlackBotOauthToken             string        `mapstructure:"SLACK_BOT_OAUTH_TOKEN"`
	SlackDataChannelId             string        `mapstructure:"SLACK_DATA_CHANNEL_ID"`
	SlackEnumChannelId             string        `mapstructure:"SLACK_ENUM_CHANNEL_ID"`
	SlackMapChannelId              string        `mapstructure:"SLACK_MAP_CHANNEL_ID"`
}

func Config() *config {
	cfgOnce.Do(func() {
		if cfgInstance == nil {
			viper.SetConfigFile(".env")
			viper.AutomaticEnv()

			err := viper.ReadInConfig()
			if err != nil {
				log.Fatal("Failed to load config", err)
			}

			err = viper.Unmarshal(&cfgInstance)
			if err != nil {
				log.Fatal("failed to load config", err)
			}
		}
	})

	return cfgInstance
}
