package thunder_share

import (
	"errors"
	"github.com/Xhofe/go-cache"
	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/drivers/thunder"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

var cacheToken = cache.NewMemCache[string]()

const objTokenCacheDuration = 360 * time.Minute

var exOpts = cache.WithEx[string](objTokenCacheDuration)

const (
	API_SHARE_URL           = "https://api-pan.xunlei.com/drive/v1/share"
	API_SHARE_DETAIL_URL    = "https://api-pan.xunlei.com/drive/v1/share/detail"
	API_SHARE_FILE_INFO_URL = "https://api-pan.xunlei.com/drive/v1/share/file_info"
	API_SHARE_RESTORE_URL   = "https://api-pan.xunlei.com/drive/v1/share/restore"
)
const (
	FOLDER = "drive#folder"
)

type Common struct {
	ShareId       string
	SharePwd      string
	ThunderDriver *thunder.ThunderExpert
}

func (c *Common) GetFiles(id string) ([]model.Obj, error) {
	pageToken := "first"
	sharePassToken, err := c.getSharePassToken()
	if err != nil {
		return nil, err
	}
	files := make([]model.Obj, 0)
	for pageToken != "" {
		if pageToken == "first" {
			pageToken = ""
		}
		query := map[string]string{
			"parent_id":       id,
			"share_id":        c.ShareId,
			"page_token":      pageToken,
			"pass_code_token": sharePassToken,
		}
		var fileList FileList
		_, err := c.ThunderDriver.Request(API_SHARE_DETAIL_URL, http.MethodGet, func(r *resty.Request) {
			r.SetQueryParams(query)
		}, &fileList)
		if err != nil {
			return nil, err
		}
		if fileList.ShareStatus != "OK" {
			if fileList.ShareStatus == "PASS_CODE_EMPTY" || fileList.ShareStatus == "PASS_CODE_ERROR" {
				// clear cache
				return c.GetFiles(id)
			}
			return nil, errors.New(fileList.ShareStatus)
		}
		if id == "" && len(fileList.Files) == 1 && fileList.Files[0].Kind == "drive#folder" {
			return c.GetFiles(fileList.Files[0].ID)
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

func (c *Common) getSharePassToken() (string, error) {
	if token, ok := cacheToken.Get(c.ShareId); ok {
		return token, nil
	}
	query := map[string]string{
		"share_id":  c.ShareId,
		"pass_code": c.SharePwd,
	}
	var fileList FileList
	_, err := c.ThunderDriver.Request(API_SHARE_URL, http.MethodGet, func(r *resty.Request) {
		r.SetQueryParams(query)
	}, &fileList)
	if err != nil {
		return "", err
	}
	cacheToken.Set(c.ShareId, fileList.PassCodeToken, exOpts)
	return fileList.PassCodeToken, nil
}

func (c *Common) GetLink(tempPathId string, id string, hash string) (*Files, error) {
	var restoreInfo RestoreInfo
	token, tokenErr := c.getSharePassToken()
	if tokenErr != nil {
		return nil, tokenErr
	}
	_, err := c.ThunderDriver.Request(API_SHARE_RESTORE_URL, http.MethodPost, func(r *resty.Request) {
		r.SetBody(&base.Json{
			"file_ids":          [1]string{id},
			"parent_id":         tempPathId,
			"pass_code_token":   token,
			"share_id":          c.ShareId,
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
		_, err := c.ThunderDriver.Request(thunder.FILE_API_URL, http.MethodGet, func(r *resty.Request) {
			r.SetQueryParams(map[string]string{
				"space":      "",
				"__type":     "drive",
				"refresh":    "true",
				"__sync":     "true",
				"parent_id":  tempPathId,
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
	_, err1 := c.ThunderDriver.Request(thunder.FILE_API_URL+"/{fileID}", http.MethodGet, func(r *resty.Request) {
		r.SetPathParam("fileID", shareFile.GetID())
		//r.SetQueryParam("space", "")
	}, &lFile)
	if err1 != nil {
		return nil, err1
	}

	go c.deleteDelay(shareFile.GetID())
	return &lFile, nil
}

func (c Common) deleteDelay(fileId string) {
	delayTime := 900
	time.Sleep(time.Duration(delayTime) * time.Second)

	log.Infoln("删除文件", fileId)
	c.ThunderDriver.Request(thunder.FILE_API_URL+":batchDelete", http.MethodPost, func(r *resty.Request) {
		r.SetBody(base.Json{
			"ids":   []string{fileId},
			"space": "",
		})
	}, nil)
}
