package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Presigner struct {
	ps     *s3.PresignClient
	bucket string
}

func NewPresigner(ctx context.Context, bucket string, region string) (*Presigner, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg)
	ps := s3.NewPresignClient(client)
	return &Presigner{ps: ps, bucket: bucket}, nil
}

// contentType を nil/空にすると署名に Content-Type を含めません。
// 署名に含めた場合、クライアント PUT でも同じヘッダを必ず送ってください。
func (p *Presigner) PresignPutObject(ctx context.Context, key string, contentType *string, expires time.Duration) (string, map[string]string, error) {
	in := &s3.PutObjectInput{
		Bucket: &p.bucket,
		Key:    &key,
	}
	if contentType != nil && *contentType != "" {
		in.ContentType = contentType
	}
	if expires <= 0 {
		expires = 5 * time.Minute
	}
	out, err := p.ps.PresignPutObject(ctx, in, func(po *s3.PresignOptions) {
		po.Expires = expires
	})
	if err != nil {
		return "", nil, err
	}
	headers := map[string]string{}
	for k, v := range out.SignedHeader {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return out.URL, headers, nil
}
