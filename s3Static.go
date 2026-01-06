package cwbiechotoolkit

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
)

type (
	// S3StaticConfig define config for S3Static
	S3StaticConfig struct {
		// Skipper defines a function to skip middleware. Returning true skips processing
		// the middleware.
		Skipper func(c echo.Context) bool

		// Aws Configuration
		// Required.
		AwsConfig aws.Config

		// S3 bucket.
		// Required.
		Bucket string `yaml:"bucket"`

		// Allows you to enable the client to use path-style addressing, i.e.,
		// https://s3.amazonaws.com/BUCKET/KEY . By default, the S3 client will use virtual
		// hosted bucket addressing when possible( https://BUCKET.s3.amazonaws.com/KEY ).
		UsePathStyle bool

		// Prefix limits the response to keys that begin with the specified prefix.
		// Optional. Default value "/"
		Prefix string `yaml:"prefix"`

		// PrefixFunc is a function that returns the prefix to use for the request.
		PrefixFunc func(c echo.Context) string `yaml:"prefixfunc"`

		// IgnoreBase is a regexp to ignore
		// Optional.
		IgnoreBaseRegex string `yaml:"ignorebaseregex"`

		// Index file for serving content.
		// Optional. Default value "index.html".
		Index string `yaml:"index"`
	}
)

var (
	DefaultS3StaticConfig = S3StaticConfig{
		Skipper: DefaultSkipper,
		Index:   "index.html",
		Prefix:  "/",
	}
)

// DefaultSkipper returns false which processes the middleware.
func DefaultSkipper(echo.Context) bool {
	return false
}

// IgnoreBase struct method returning a new path using IgnoreBaseRegex
//
// Parameter:
// pin returns as a string
func (s *S3StaticConfig) IgnoreBase(pin string) (pout string, err error) {
	re, err := regexp.Compile(s.IgnoreBaseRegex)
	if err != nil {
		return "", err
	}
	p := path.Clean("/" + pin)
	if re.MatchString(p) {
		matches := re.FindStringSubmatch(p)
		if len(matches) > 0 {
			relativePath := p[len(matches[0]):]
			// remove leading separator
			if len(relativePath) > 0 && relativePath[0] == '/' {
				relativePath = relativePath[1:]
			}
			pout = relativePath
		}
	}
	return pout, err
}

// S3Satic middleware returning S3StaticWithConfig with Default configurations
func S3Satic(S3StaticConfig S3StaticConfig) echo.MiddlewareFunc {
	c := DefaultS3StaticConfig
	return S3StaticWithConfig(c)
}

// S3StaticWithConfig returns S3Static middleware with config
// See `S3Static()`
func S3StaticWithConfig(staticConfig S3StaticConfig) echo.MiddlewareFunc {
	if staticConfig.Skipper == nil {
		staticConfig.Skipper = DefaultS3StaticConfig.Skipper
	}
	if staticConfig.Index == "" {
		staticConfig.Index = DefaultS3StaticConfig.Index
	}
	if staticConfig.Prefix == "" {
		staticConfig.Prefix = DefaultS3StaticConfig.Prefix
	}
	if !strings.HasSuffix(staticConfig.Prefix, "/") {
		staticConfig.Prefix += "/"
	}
	if strings.HasPrefix(staticConfig.Prefix, "/") && staticConfig.Prefix != "/" {
		staticConfig.Prefix = strings.TrimPrefix(staticConfig.Prefix, "/")
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if staticConfig.Skipper(c) {
				return next(c)
			}

			// get a clean url path
			p := c.Request().URL.Path

			p, err = url.PathUnescape(p)
			if err != nil {
				log.Printf("PathUnescape error: %s\n", err)
				return
			}

			log.Printf("Path before IgnoreBase: %s; %s", p, staticConfig.IgnoreBaseRegex)
			// ignore the base with regex
			if staticConfig.IgnoreBaseRegex != "" {
				p, err = staticConfig.IgnoreBase(p)
				if err != nil {
					return err
				}
			}
			log.Printf("Path after IgnoreBase: %s; %s", p, staticConfig.IgnoreBaseRegex)

			// set the Prefix from the PrefixFunc if available
			// PrefixFunc will take precedence over Prefix
			if staticConfig.PrefixFunc != nil {
				staticConfig.Prefix = staticConfig.PrefixFunc(c)
			}

			// set the potential key from path and default key incase that does not exist
			pathKey := path.Join(staticConfig.Prefix, path.Clean("/"+p))
			log.Printf("Path Key to check S3 objects: %s", pathKey)

			// default to prefix/index.html
			key := path.Join(staticConfig.Prefix, path.Clean("/"+staticConfig.Index))

			if err != nil {
				log.Printf("LoadDefaultConfig error: %s\n", err)
				return
			}

			client := s3.NewFromConfig(staticConfig.AwsConfig,
				func(o *s3.Options) {
					o.UsePathStyle = staticConfig.UsePathStyle
				})

			// first, check if the bucket exists and we have permission to access
			_, err = client.HeadBucket(context.Background(), &s3.HeadBucketInput{Bucket: &staticConfig.Bucket})
			if err != nil {
				log.Printf("no permissions or bucket does not exist: %s", staticConfig.Bucket)
				return
			}

			// get a list of content in the bucket limited on the Prefix
			objects, err := client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
				Bucket: &staticConfig.Bucket,
				Prefix: &staticConfig.Prefix,
			})

			if err != nil {
				log.Printf("ListObjectsV2 error: %s\n", err)
				return
			}

			log.Printf("Found %d S3 Objects", len(objects.Contents))
			// check that the incoming path is available
			for _, objContent := range objects.Contents {
				if pathKey == *objContent.Key {
					key = pathKey
					break
				}
			}

			log.Printf("Key and Bucket to GetObject: %s; %s", key, staticConfig.Bucket)

			// if available, get and serve
			var obj *s3.GetObjectOutput
			obj, err = client.GetObject(context.Background(), &s3.GetObjectInput{
				Bucket: &staticConfig.Bucket,
				Key:    &key,
			})
			if err != nil {
				log.Printf("GetObject error on key '%s': %s\n", key, err)
				return
			}
			defer obj.Body.Close()

			log.Printf("Content length to stream: %d", *obj.ContentLength)

			// stream content
			err = c.Stream(http.StatusOK, *obj.ContentType, obj.Body)
			if err != nil {
				log.Printf("Streaming error for '%s': %s", key, err)
			}

			return err
		}
	}
}
