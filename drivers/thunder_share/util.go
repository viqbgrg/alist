package thunder_share

import (
	"errors"
	"github.com/alist-org/alist/v3/drivers/thunder"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/go-resty/resty/v2"
	"net/http"
)

const (
	API_SHARE_URL           = "https://api-pan.xunlei.com/drive/v1/share"
	API_SHARE_DETAIL_URL    = "https://api-pan.xunlei.com/drive/v1/share/detail"
	API_SHARE_FILE_INFO_URL = "https://api-pan.xunlei.com/drive/v1/share/file_info"
	API_SHARE_RESTORE_URL   = "https://api-pan.xunlei.com/drive/v1/share/restore"
)
const (
	FOLDER = "drive#folder"
)

func GetFiles(id string, shardId string, sharePwd string, passCodeToken string, thunderDriver thunder.ThunderExpert) ([]model.Obj, error) {
	files := make([]model.Obj, 0)
	pageToken := "first"
	for pageToken != "" {
		if pageToken == "first" {
			pageToken = ""
		}
		query := map[string]string{
			"parent_id":       id,
			"share_id":        shardId,
			"page_token":      pageToken,
			"pass_code_token": passCodeToken,
		}
		var fileList FileList
		_, err := thunderDriver.Request(API_SHARE_DETAIL_URL, http.MethodGet, func(r *resty.Request) {
			r.SetQueryParams(query)
		}, &fileList)
		if err != nil {
			return nil, err
		}
		if fileList.ShareStatus != "OK" {
			if fileList.ShareStatus == "PASS_CODE_EMPTY" || fileList.ShareStatus == "PASS_CODE_ERROR" {
				passCodeToken, err = getSharePassToken(shardId, sharePwd, thunderDriver)
				if err != nil {
					return nil, err
				}
				return GetFiles(id, shardId, sharePwd, passCodeToken, thunderDriver)
			}
			return nil, errors.New(fileList.ShareStatusText)
		}
		if id == "" && len(fileList.Files) == 1 && fileList.Files[0].Kind == "drive#folder" {
			return GetFiles(fileList.Files[0].ID, shardId, sharePwd, passCodeToken, thunderDriver)
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

func getSharePassToken(shareId string, sharePwd string, thunderDriver thunder.ThunderExpert) (string, error) {
	query := map[string]string{
		"share_id":  shareId,
		"pass_code": sharePwd,
	}
	var fileList FileList
	_, err := thunderDriver.Request(API_SHARE_URL, http.MethodGet, func(r *resty.Request) {
		r.SetQueryParams(query)
	}, &fileList)
	if err != nil {
		return "", err
	}
	return fileList.PassCodeToken, nil
}
