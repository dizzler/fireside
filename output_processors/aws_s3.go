package fireside
// File: output_processors/aws_s3.go

import (
        "compress/gzip"
	"fmt"
	"io"
	"strings"

        "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
        "github.com/aws/aws-sdk-go/service/s3/s3manager"

        "github.com/dailyburn/ratchet/data"
        log "github.com/sirupsen/logrus"
)

// S3Writer sends data upstream to S3. By default, we will not compress data before sending it.
// Set the `Compress` flag to true to use gzip compression before storing in S3 (if this flag is
// set to true, ".gz" will automatically be appended to the key name specified).
//
// By default, we will separate each iteration of data sent to `ProcessData` with a new line
// when we piece back together to send to S3. Change the `LineSeparator` attribute to change
// this behavior.
type S3Writer struct {
	data           []string
	Compress       bool
	LineSeparator  string
	bucket         string
	key            string //object path to write to within the S3 bucket
        region         string
	sess           *session.Session
}

// NewS3Writer instantiates a new S3Writer
func NewS3Writer(config *OutputConfig) *S3Writer {
	w := S3Writer{
		bucket: config.AWS.S3Bucket,
	        key: config.AWS.S3BasePath,
		LineSeparator: "\n",
		Compress: false}

	client, err := NewAwsClient(config)
        if err != nil {
                log.WithError(err).Fatal("error in data processing pipeline")
        }
	w.sess = client.AwsSession

	return &w
}

// ProcessData enqueues all received data
func (w *S3Writer) ProcessData(d data.JSON, outputChan chan data.JSON, killChan chan error) {
	w.data = append(w.data, string(d))
}

// Finish writes all enqueued data to S3, defering to WriteS3Object
func (w *S3Writer) Finish(outputChan chan data.JSON, killChan chan error) {
	WriteS3Object(w.data, w.sess, w.bucket, w.key, w.LineSeparator, w.Compress)
}

func (w *S3Writer) String() string {
	return "S3Writer"
}

// WriteS3Object writes the data to the given key, optionally compressing it first
func WriteS3Object(data []string, sess *session.Session, bucket string, key string, lineSeparator string, compress bool) (string, error) {
	var reader io.Reader

	byteReader := strings.NewReader(strings.Join(data, lineSeparator))

	if compress {
		key = fmt.Sprintf("%v.gz", key)
		pipeReader, pipeWriter := io.Pipe()
		reader = pipeReader

		go func() {
			gw := gzip.NewWriter(pipeWriter)
			io.Copy(gw, byteReader)
			gw.Close()
			pipeWriter.Close()
		}()
	} else {
		reader = byteReader
	}

	uploader := s3manager.NewUploader(sess)

	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:   reader,
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return result.Location, err
}
