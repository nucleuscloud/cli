package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

func main() {
	s3Client := getS3Client()
	bucketName := "nucleus-cloud-stage-service-bucket"
	uploadKey := getUrlUploadKey("nick-cli", "nodejs-sample")
	fmt.Println("getting request")
	uploadReq, _ := s3Client.PutObjectRequest(&s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(uploadKey),
		ContentType: aws.String("application/gzip"),
	})
	signedURL, err := uploadReq.Presign(15 * time.Minute)

	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(signedURL)
	fmt.Println("=====")
	fmt.Println(os.Getenv("AWS_ACCESS_KEY_ID"))
	fmt.Println(os.Getenv("AWS_SECRET_ACCESS_KEY"))
	fmt.Println(os.Getenv("AWS_ROLE_ARN"))
	fmt.Println(os.Getenv("AWS_SESSION_TOKEN"))
	fmt.Println(os.Getenv("AWS_SECURITY_TOKEN"))
}

func getS3Client() *s3.S3 {
	sess := session.Must(session.NewSession()) // AWS Credentials pulled from a variety of locations
	creds := stscreds.NewCredentials(sess, "arn:aws:iam::997306413652:role/allow-full-access-from-other-accounts")
	s3Client := s3.New(sess, &aws.Config{
		Credentials: creds,
	})
	fmt.Println("hit here")
	return s3Client
}

var UPLOAD_KEY_REPLACER *strings.Replacer = strings.NewReplacer("/", "", "#", "", "[", "", "]", "", "?", "", "*", "")

func getUrlUploadKey(environmentName string, serviceName string) string {
	sanitizedEnvironmentName := UPLOAD_KEY_REPLACER.Replace(environmentName)
	sanitizedServiceName := UPLOAD_KEY_REPLACER.Replace(serviceName)
	return fmt.Sprintf("%s/%s/%d_%s.tar.gz", sanitizedEnvironmentName, sanitizedServiceName, time.Now().UTC().Unix(), uuid.New().String())
}
