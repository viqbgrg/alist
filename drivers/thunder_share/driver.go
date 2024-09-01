package thunder_share

import (
	"context"
	"errors"
	"fmt"
	"github.com/alist-org/alist/v3/drivers/thunder"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"net/http"
)

type ThunderShare struct {
	model.Storage
	Addition
	C Common
}

func (d *ThunderShare) Config() driver.Config {
	return config
}

func (d *ThunderShare) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *ThunderShare) Init(ctx context.Context) error {
	storages := op.GetAllStorages()
	d.C.ShareId = d.ShareId
	d.C.SharePwd = d.SharePwd
	for _, storage := range storages {
		thunderDriver, ok := storage.(*thunder.ThunderExpert)
		if ok {
			d.C.ThunderDriver = thunderDriver
			break
		}
	}
	if d.C.ThunderDriver == nil {
		return fmt.Errorf("unsupported storage driver for offline download, only Pikpak is supported")
	}

	return nil
}

func (d *ThunderShare) Drop(ctx context.Context) error {
	return nil
}

func (d *ThunderShare) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	return d.C.GetFiles(dir.GetID())
}

func (d *ThunderShare) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	var hash string
	if o, ok := file.(*Files); ok {
		hash = o.Hash
	} else {
		return nil, errors.New("invalid file type")
	}

	lFile, err := d.C.GetLink(d.TempPathId, file.GetID(), hash)
	if err != nil {
		return nil, err
	}
	link := &model.Link{
		URL: lFile.WebContentLink,
		Header: http.Header{
			"User-Agent": {d.C.ThunderDriver.DownloadUserAgent},
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

var _ driver.Driver = (*ThunderShare)(nil)
