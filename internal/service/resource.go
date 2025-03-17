package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/fanchunke/deeppick-ai/internal/config"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type CosClient struct {
	*cos.Client
	tmpSecretId  string
	tmpSecretKey string
	token        string
	expiredTime  int64
}

type ResourceService struct {
	cosClient *CosClient
	tracer    trace.Tracer
	cfg       *config.Config
}

func NewResourceService(cfg *config.Config) *ResourceService {
	return &ResourceService{cosClient: nil, cfg: cfg, tracer: otel.Tracer("UploadService")}
}

type UploadResponse struct {
	Url string `json:"url"`
}

func (s *ResourceService) Upload() echo.HandlerFunc {
	return func(c echo.Context) error {
		file, err := c.FormFile("image")
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		f, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		defer f.Close()

		ctx := c.Request().Context()
		if s.cosClient == nil || s.cosClient.expiredTime-time.Now().Unix() < 0 {
			cosAuthCtx, cosAuthSpan := s.tracer.Start(ctx, "cosAuth")
			if err := s.initCosClient(cosAuthCtx); err != nil {
				return err
			}
			cosAuthSpan.End()
		}

		// 开始上传
		uploadCtx, span := s.tracer.Start(ctx, "upload")
		objectName := fmt.Sprintf("%s%s", uuid.New().String(), path.Ext(file.Filename))
		_, err = s.cosClient.Object.Put(uploadCtx, objectName, f, nil)
		if err != nil {
			return err
		}

		span.End()

		// 获取链接
		getPreSignedUrlCtx, span := s.tracer.Start(ctx, "upload")
		presignedURL, err := s.cosClient.Object.GetPresignedURL(getPreSignedUrlCtx, http.MethodGet, objectName, s.cosClient.tmpSecretId, s.cosClient.tmpSecretKey, time.Hour, s.cosClient.token)
		if err != nil {
			return err
		}
		span.End()

		return c.JSON(http.StatusOK, UploadResponse{
			Url: presignedURL.String(),
		})

	}
}

type CosAuthResponse struct {
	TmpSecretId  string `json:"TmpSecretId"`
	TmpSecretKey string `json:"TmpSecretKey"`
	Token        string `json:"Token"`
	ExpiredTime  int64  `json:"ExpiredTime"`
}

func (s *ResourceService) getCosAuth(ctx context.Context) (*CosAuthResponse, error) {
	url := "http://api.weixin.qq.com/_/cos/getauth"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cos auth response status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	var response CosAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *ResourceService) initCosClient(ctx context.Context) error {
	u, _ := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", s.cfg.Cos.Bucket, s.cfg.Cos.Region))
	b := &cos.BaseURL{BucketURL: u}

	authResponse, err := s.getCosAuth(ctx)
	if err != nil {
		return err
	}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:     authResponse.TmpSecretId,
			SecretKey:    authResponse.TmpSecretKey,
			SessionToken: authResponse.Token,
		},
	})
	s.cosClient = &CosClient{
		Client:       client,
		tmpSecretId:  authResponse.TmpSecretId,
		tmpSecretKey: authResponse.TmpSecretKey,
		token:        authResponse.Token,
		expiredTime:  authResponse.ExpiredTime,
	}
	return nil
}
