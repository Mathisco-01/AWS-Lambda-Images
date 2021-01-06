package main

import (
	b64 "encoding/base64"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var REGION, BUCKET string
var imageLinks []string
var statusCode int
var s3session *s3.S3
var imageCache map[string]string

type Output struct {
	StatusCode      int               `json:"statusCode"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	IsBase64Encoded bool              `json:"isBase64Encoded"`
}

func init() {
	REGION = os.Getenv("REGION")
	BUCKET = os.Getenv("BUCKET")

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(REGION),
	}))

	s3session = s3.New(sess)

	resp, err := s3session.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(BUCKET),
	})
	if err != nil {
		log.Println(err)
		statusCode = 400
		return
	}

	for _, obj := range resp.Contents {
		imageLinks = append(imageLinks, formats3link(*obj.Key))
	}

	imageCache = make(map[string]string)
}

func main() {
	lambda.Start(handler)
}

func handler() (o Output, err error) {
	var url string
	url = randomImageLink()

	cachedImg := imageCache[url]
	if len(cachedImg) == 0 {
		c, err := getImage(url)
		if err != nil {
			statusCode = 400
		}

		imageCache[url] = b64.StdEncoding.EncodeToString(c)
	}


	var contentType string
	switch {
	case strings.Contains(url, ".jpeg"):
		contentType = "image/jpeg"
	case strings.Contains(url, ".jpg"):
		contentType = "image/jpeg"
	case strings.Contains(url, ".png"):
		contentType = "image/png"
	}

	switch statusCode {
	case 400:
		o = Output{
			StatusCode: statusCode,
		}
	default:
		o = Output{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": contentType,
			},
			Body:            imageCache[url],
			IsBase64Encoded: true,
		}
	}

	return o, err
}

func getImage(url string) (bytes []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return bytes, err
	}

	defer resp.Body.Close()

	bytes, err = ioutil.ReadAll(resp.Body)
	return bytes, err
}

func formats3link(k string) string {
	return fmt.Sprintf("https://%v.s3-%v.amazonaws.com/%v", BUCKET, REGION, k)
}

func randomImageLink() string {
	r := rand.Intn(len(imageLinks) - 1)
	return imageLinks[r]
}