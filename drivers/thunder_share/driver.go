package thunder_share

import (
	"context"
	"errors"
	"fmt"
	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/drivers/thunder"
	"github.com/alist-org/alist/v3/internal/op"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"

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
	storages := op.GetAllStorages()

	for _, storage := range storages {
		thunderDriver, ok := storage.(*thunder.ThunderExpert)
		if ok {
			d.thunderDriver = thunderDriver
			break
		}
	}
	if d.thunderDriver == nil {
		return fmt.Errorf("unsupported storage driver for offline download, only Pikpak is supported")
	}

	return nil
}

func (d *ThunderShare) Drop(ctx context.Context) error {
	return nil
}

func (d *ThunderShare) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	return GetFiles(dir.GetID(), d.ShareId, d.SharePwd, "", *d.thunderDriver)
}

func (d *ThunderShare) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	var restoreInfo RestoreInfo
	var hash string
	if o, ok := file.(*Files); ok {
		hash = o.Hash
	} else {
		return nil, errors.New("invalid file type")
	}
	_, err := d.thunderDriver.Request(API_SHARE_RESTORE_URL, http.MethodPost, func(r *resty.Request) {
		r.SetContext(ctx)
		r.SetBody(&base.Json{
			"file_ids":          [1]string{file.GetID()},
			"parent_id":         d.TempPathId,
			"pass_code_token":   d.PassCodeToken,
			"share_id":          d.ShareId,
			"specify_parent_id": true,
		})
	}, &restoreInfo)
	if err != nil {
		return nil, err
	}
	var pageToken string
	var shareFile Files
	for {
		var fileList FileList
		_, err := d.thunderDriver.Request(thunder.FILE_API_URL, http.MethodGet, func(r *resty.Request) {
			r.SetContext(ctx)
			r.SetQueryParams(map[string]string{
				"space":      "",
				"__type":     "drive",
				"refresh":    "true",
				"__sync":     "true",
				"parent_id":  d.TempPathId,
				"page_token": pageToken,
				"with_audit": "true",
				"limit":      "100",
				"filters":    `{"phase":{"eq":"PHASE_TYPE_COMPLETE"},"trashed":{"eq":false}}`,
			})
		}, &fileList)
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(fileList.Files); i++ {
			if hash == fileList.Files[i].Hash {
				shareFile = fileList.Files[i]
				break
			}
		}

		if fileList.NextPageToken == "" {
			break
		}
		pageToken = fileList.NextPageToken
	}
	var lFile Files
	_, err1 := d.thunderDriver.Request(thunder.FILE_API_URL+"/{fileID}", http.MethodGet, func(r *resty.Request) {
		r.SetContext(ctx)
		r.SetPathParam("fileID", shareFile.GetID())
		//r.SetQueryParam("space", "")
	}, &lFile)
	if err1 != nil {
		return nil, err1
	}

	go d.deleteDelay(shareFile.GetID())

	link := &model.Link{
		URL: lFile.WebContentLink,
		Header: http.Header{
			"User-Agent": {d.thunderDriver.DownloadUserAgent},
		},
	}

	if d.UseVideoUrl {
		for _, media := range lFile.Medias {
			if media.Link.URL != "" {
				link.URL = media.Link.URL
				break
			}
		}
	}
	return link, nil
}

func (d *ThunderShare) deleteDelay(fileId string) {
	delayTime := 900
	time.Sleep(time.Duration(delayTime) * time.Second)

	log.Infoln("删除文件", fileId)
	d.thunderDriver.Request(thunder.FILE_API_URL+":batchDelete", http.MethodPost, func(r *resty.Request) {
		r.SetBody(base.Json{
			"ids":   []string{fileId},
			"space": "",
		})
	}, nil)
}

var _ driver.Driver = (*ThunderShare)(nil)
