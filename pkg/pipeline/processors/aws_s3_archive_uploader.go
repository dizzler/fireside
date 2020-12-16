package processors

import (
    "os"
    "path/filepath"

    "github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3UploadIterator iterates through archive (*.tar.gz) files,
// uploading the list of archives to an AWS S3 bucket
type S3UploadIterator struct {
    filePaths []string
    bucket    string
    next      struct {
        path string
        f    *os.File
    }
    err       error
}

// NewS3UploadIterator creates and returns a new BatchUploadIterator
func NewS3UploadIterator(bucket string, filePaths []string) s3manager.BatchUploadIterator {
    return &S3UploadIterator{
        filePaths: filePaths,
        bucket:    bucket,
    }
}

// Next opens the next file and stops iteration if it fails to open a file.
func (iter *S3UploadIterator) Next() bool {
    if len(iter.filePaths) == 0 {
        iter.next.f = nil
        return false
    }

    f, err := os.Open(iter.filePaths[0])
    iter.err = err

    iter.next.f = f
    iter.next.path = iter.filePaths[0]

    iter.filePaths = iter.filePaths[1:]
    return true && iter.Err() == nil
}

// Err returns an error that was set during opening the file
func (iter *S3UploadIterator) Err() error {
    return iter.err
}

// UploadObject returns a BatchUploadObject and sets the After field to
// close the file.
func (iter *S3UploadIterator) UploadObject() s3manager.BatchUploadObject {
    f := iter.next.f
    p := iter.next.path
    k := "fireside-test" + "/" + filepath.Base(p)
    sse := "aws:kms"
    return s3manager.BatchUploadObject{
        Object: &s3manager.UploadInput{
            Bucket: &iter.bucket,
            Key:    &k,
            Body:   f,
            ServerSideEncryption: &sse,
        },
        After: func() error {
            return f.Close()
        },
    }
}
