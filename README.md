# Static Web Publication Generator

This is a script based on https://github.com/banux/webpub-streamer that takes a directory of EPUB files and generates web publication files for use on a static site.

## Usage Example

go run main.go -epubDir=books -outputDir=out -domain=http://example.com

## Uploading output to s3

Install the aws cli and run `aws configure` to set up credentials. Then run `aws s3 cp <outputDir> s3://<bucket>/<folder>/ --recursive --acl public-read` 
