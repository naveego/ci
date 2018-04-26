package build

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/coreos/go-semver/semver"
)

// UploadToS3 uploads a file to an AWS S3 bucket
func UploadToS3(bucket, src, target string) (bool, error) {
	sess, err := session.NewSession()
	if err != nil {
		return false, err
	}

	uploader := s3manager.NewUploader(sess)

	f, err := os.Open(src)
	if err != nil {
		return false, fmt.Errorf("failed to open file %q, %v", src, err)
	}

	// Upload the file to S3
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(target),
		Body:   f,
	})
	if err != nil {
		return false, fmt.Errorf("failed to upload file, %v", err)
	}

	return true, nil
}

// ToS3ReleasePath returns a path for upload to S3 in the format
// 'releases/{serviceID}/{version.Major}.{version.Minor}.{version.Patch}/{pkgName}'.
func ToS3ReleasePath(pkgName, serviceID string, version semver.Version) string {
	return fmt.Sprintf("releases/%s/%d.%d.%d/%s", serviceID, version.Major, version.Minor, version.Patch, pkgName)
}
