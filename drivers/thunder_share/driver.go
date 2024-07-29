package thunder_share

import (
	"context"
	"errors"
	"fmt"
	"github.com/alist-org/alist/v3/drivers/thunder"
	"github.com/alist-org/alist/v3/internal/op"
	"net/http"

	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/go-resty/resty/v2"
)

type ThunderShare struct {
	model.Storage
	Addition
	thunderDriver *thunder.ThunderExpert
	PassCodeToken string
}

func (d *ThunderShare) Config() driver.Config {
	return config
}

func (d *ThunderShare) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *ThunderShare) Init(ctx context.Context) error {
	storage, _, err := op.GetStorageAndActualPath(d.ThunderPath)
	if err != nil {
		return err
	}
	thunderDriver, ok := storage.(*thunder.ThunderExpert)
	if !ok {
		return fmt.Errorf("unsupported storage driver for offline download, only Pikpak is supported")
	}
	d.thunderDriver = thunderDriver
	return nil
}

func (d *ThunderShare) Drop(ctx context.Context) error {
	return nil
}

func (d *ThunderShare) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	return d.getFiles(dir.GetID())
}

func (d *ThunderShare) getFiles(id string) ([]model.Obj, error) {
	files := make([]model.Obj, 0)
	pageToken := "first"
	for pageToken != "" {
		if pageToken == "first" {
			pageToken = ""
		}
		query := map[string]string{
			"parent_id":       id,
			"share_id":        d.ShareId,
			"page_token":      pageToken,
			"pass_code_token": d.PassCodeToken,
		}
		var fileList FileList
		_, err := d.thunderDriver.Request(API_SHARE_DETAIL_URL, http.MethodGet, func(r *resty.Request) {
			r.SetQueryParams(query)
		}, &fileList)
		if err != nil {
			return nil, err
		}
		if fileList.ShareStatus != "OK" {
			if fileList.ShareStatus == "PASS_CODE_EMPTY" || fileList.ShareStatus == "PASS_CODE_ERROR" {
				err = d.getSharePassToken()
				if err != nil {
					return nil, err
				}
				return d.getFiles(id)
			}
			return nil, errors.New(fileList.ShareStatusText)
		}
		for i := 0; i < len(fileList.Files); i++ {
			files = append(files, &fileList.Files[i])
		}

		if fileList.NextPageToken == "" {
			break
		}
		pageToken = fileList.NextPageToken
	}
	return files, nil
}

func (d *ThunderShare) getSharePassToken() error {
	query := map[string]string{
		"share_id":  d.ShareId,
		"pass_code": d.SharePwd,
	}
	var fileList FileList
	_, err := d.thunderDriver.Request(API_SHARE_URL, http.MethodGet, func(r *resty.Request) {
		r.SetQueryParams(query)
	}, &fileList)
	if err != nil {
		return err
	}
	d.PassCodeToken = fileList.PassCodeToken
	return nil
}

func (d *ThunderShare) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	var fileInfo FileInfo
	query := map[string]string{
		"share_id":        d.ShareId,
		"file_id":         file.GetID(),
		"pass_code_token": d.PassCodeToken,
	}
	_, err := d.thunderDriver.Request(API_SHARE_FILE_INFO_URL, http.MethodGet, func(r *resty.Request) {
		r.SetContext(ctx)
		r.SetQueryParams(query)
	}, &fileInfo)
	if err != nil {
		return nil, err
	}
	link := &model.Link{
		URL: fileInfo.Files.WebContentLink,
		Header: http.Header{
			"User-Agent": {d.thunderDriver.DownloadUserAgent},
		},
	}

	if d.thunderDriver.UseVideoUrl {
		for _, media := range fileInfo.Files.Medias {
			if media.Link.URL != "" {
				link.URL = media.Link.URL
				break
			}
		}
	}
	return link, nil
}

var _ driver.Driver = (*ThunderShare)(nil)
