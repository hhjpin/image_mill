package image_mill

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/endpoints"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/imagesearch"
	"github.com/hhjpin/goutils/logger"
)

type syncType string

const (
	syncTypeAdd    syncType = "add"
	syncTypeDelete syncType = "delete"
)

//图片搜索返回的部分错误码
const (
	CodeDeniedRequest         = "DeniedRequest"
	CodeThrottlingUser        = "Throttling.User"
	CodeInvalidStartParameter = "InvalidStartParameter"
)

type ImageItem struct {
	PicName   string
	ProductId string
	ImageUrl  string //图片链接，可以是url中的一部分，比如图片hash值等，能唯一确定一张图片即可，可通过DownloadUrlFunc补充
}

type SyncReturn struct {
	Image     ImageItem
	Error     error
	SyncAgain bool
}

type ImageAttach struct {
	InstanceName string           //图像搜索实例
	CategoryId   requests.Integer //图片分类
	StrAttr      string           //图片属性
}

type ImageMill struct {
	client *imagesearch.Client

	downloadUrlFunc DownloadUrlFunc
}

type DownloadUrlFunc func(image string) string

type Conf struct {
	RegionId        string
	ProductId       string
	Endpoint        string
	AccessKeyId     string
	AccessKeySecret string
	DownloadUrlFunc DownloadUrlFunc
}

func New(conf *Conf) (*ImageMill, error) {
	var mill = &ImageMill{}
	err := endpoints.AddEndpointMapping(conf.RegionId, conf.ProductId, conf.Endpoint)
	if err != nil {
		return mill, err
	}
	mill.client, err = imagesearch.NewClientWithAccessKey(conf.RegionId, conf.AccessKeyId, conf.AccessKeySecret)
	if err != nil {
		return mill, err
	}
	mill.downloadUrlFunc = conf.DownloadUrlFunc
	return mill, nil
}

func (m *ImageMill) AddImage(images []ImageItem, attach *ImageAttach) ([]string, error) {
	return m.syncImages(images, attach, syncTypeAdd, len(images)+1)
}

func (m *ImageMill) DeleteImage(images []ImageItem, attach *ImageAttach) ([]string, error) {
	return m.syncImages(images, attach, syncTypeDelete, len(images)+1)
}

func (m *ImageMill) syncImages(images []ImageItem, attach *ImageAttach, syncType syncType, tryTimes int) ([]string, error) {
	var okIds []string
	var timeout = false
	var imageCnt = 0
	var imageNum = len(images)
	var againImages []ImageItem
	var failImages []ImageItem
	if imageNum == 0 {
		return okIds, nil
	}
	//防止栈溢出
	if tryTimes <= 0 {
		return okIds, fmt.Errorf("func stack overflow")
	}

	var syncFunc func(image ImageItem, attach *ImageAttach, ret chan<- SyncReturn)
	if syncType == syncTypeAdd {
		syncFunc = m.syncImageForAdd
	} else if syncType == syncTypeDelete {
		syncFunc = m.syncImageForDelete
	} else {
		err := fmt.Errorf("同步图片类型错误！")
		logger.Error(err)
		return okIds, err
	}

	var ret = make(chan SyncReturn)
	for _, img := range images {
		go syncFunc(img, attach, ret)
	}

	for {
		select {
		case sync := <-ret:
			if sync.Error == nil {
				okIds = append(okIds, sync.Image.PicName)
			} else if sync.SyncAgain {
				againImages = append(againImages, sync.Image)
			} else {
				failImages = append(failImages, sync.Image)
			}
		case <-time.After(time.Second * 5):
			timeout = true
			logger.Error(fmt.Errorf("同步图片超时"))
			break
		}
		imageCnt++
		if imageCnt >= imageNum || timeout {
			break
		}
	}

	logger.Infof("单次欲同步:%d,成功:%d,再次尝试:%d,其他失败:%d,类型:%s-%s", len(images), len(okIds), len(againImages), len(failImages), syncType, attach.StrAttr)

	//重新上传因接口频率限制而失败的图片
	if len(againImages) > 0 {
		time.Sleep(time.Millisecond * 500) //歇会
		tryTimes--
		newOkIds, err := m.syncImages(againImages, attach, syncType, tryTimes)
		if err != nil {
			return okIds, nil
		}
		okIds = append(okIds, newOkIds...)
	}

	return okIds, nil
}

func (m *ImageMill) downloadImageToBase64(imageUrl string) (string, error) {
	resp, err := http.Get(imageUrl)
	if err != nil {
		logger.Error(err)
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error(err)
		}
	}()
	data, _ := ioutil.ReadAll(resp.Body)
	return base64.StdEncoding.EncodeToString(data), nil
}

//缓存图片
var imageAddCache = newImageCache()

func (m *ImageMill) syncImageForAdd(image ImageItem, attach *ImageAttach, ret chan<- SyncReturn) {
	picContent := imageAddCache.Load(image.ImageUrl)
	if picContent == "" {
		var err error
		imageUrl := image.ImageUrl
		if m.downloadUrlFunc != nil {
			imageUrl = m.downloadUrlFunc(imageUrl)
		}
		picContent, err = m.downloadImageToBase64(imageUrl)
		if err != nil {
			ret <- SyncReturn{Error: err}
			return
		}
		imageAddCache.Store(image.ImageUrl, picContent)
	}
	request := imagesearch.CreateAddImageRequest()
	request.InstanceName = attach.InstanceName
	request.PicName = image.PicName
	request.ProductId = image.ProductId
	request.CategoryId = attach.CategoryId
	request.PicContent = picContent
	request.StrAttr = attach.StrAttr
	resp, err := m.client.AddImage(request)
	if err != nil {
		if resp != nil {
			baseResp := m.formatBaseResponseError(resp.BaseResponse)
			//访问限制问题重新请求
			if m.isLimitUsedCode(baseResp.Code) {
				ret <- SyncReturn{
					Image:     image,
					SyncAgain: true,
					Error:     err,
				}
				return
			}
		}
		logger.Error(err)
		ret <- SyncReturn{
			Image: image,
			Error: err,
		}
		return
	}
	ret <- SyncReturn{Image: image}
	//logger.Info("addResponse:\n", resp)
	imageAddCache.Delete(image.ImageUrl)
}

func (m *ImageMill) syncImageForDelete(image ImageItem, attach *ImageAttach, ret chan<- SyncReturn) {
	request := imagesearch.CreateDeleteImageRequest()
	request.InstanceName = attach.InstanceName
	request.PicName = image.PicName
	request.ProductId = image.ProductId
	resp, err := m.client.DeleteImage(request)
	if err != nil {
		if resp != nil {
			baseResp := m.formatBaseResponseError(resp.BaseResponse)
			//访问限制问题重新请求
			if m.isLimitUsedCode(baseResp.Code) {
				ret <- SyncReturn{
					Image:     image,
					SyncAgain: true,
					Error:     err,
				}
				return
			}
		}
		logger.Error(err)
		ret <- SyncReturn{
			Image: image,
			Error: err,
		}
		return
	}
	ret <- SyncReturn{Image: image}
	//logger.Info("deleteResponse:\n", resp)
}

type SearchParam struct {
	ImageUrl       string
	UseOriginImage bool //是否使用原图，不使用已设置的修改下载url的函数
	Offset         int
	Limit          int              //0默认就是最大值100
	StrAttr        string           //图片附加属性，可分类图片
	IsRemoval      bool             //是否对productId去重
	InstanceName   string           //阿里云搜索实例
	CategoryId     requests.Integer //搜索分类id
}

type SearchResult struct {
	ImageUrl   string
	Offset     int
	ProductIds []string
	DocsFound  int
	DocsReturn int
	SearchTime int
}

func (m *ImageMill) SearchImage(p SearchParam) (*SearchResult, error) {
	res := &SearchResult{}
	res.ImageUrl = p.ImageUrl
	res.Offset = -1 //-1 means no more data
	res.ProductIds = []string{}
	if p.Offset < 0 {
		return res, fmt.Errorf("offset must >= 0")
	}
	if p.ImageUrl == "" {
		return res, fmt.Errorf("image url is empty")
	}
	if !p.UseOriginImage && m.downloadUrlFunc != nil {
		p.ImageUrl = m.downloadUrlFunc(p.ImageUrl)
	}
	picContent, err := m.downloadImageToBase64(p.ImageUrl)
	if err != nil {
		return res, err
	}

	if p.Limit > 100 || p.Limit <= 0 {
		p.Limit = 100 //接口限制的最大值
	}
	request := imagesearch.CreateSearchImageRequest()
	request.InstanceName = p.InstanceName
	request.PicContent = picContent
	request.Type = "SearchByPic"
	request.CategoryId = p.CategoryId
	request.Filter = fmt.Sprintf(`str_attr="%s"`, p.StrAttr)
	request.Start = requests.Integer(strconv.Itoa(p.Offset))
	request.Num = requests.Integer(strconv.Itoa(p.Limit))
	resp, err := m.client.SearchImage(request)
	if err != nil {
		if resp != nil {
			baseResp := m.formatBaseResponseError(resp.BaseResponse)
			if baseResp.Code == CodeInvalidStartParameter {
				return res, nil
			} else if m.isLimitUsedCode(baseResp.Code) {
				return res, fmt.Errorf("搜索量过大，稍后再试")
			} else if baseResp.Message != "" {
				return res, fmt.Errorf("图搜错误:%s", baseResp.Message)
			}
		}
		logger.Error(err)
		return res, fmt.Errorf("图搜错误:%s", err.Error())
	}
	//logger.Info("searchResponse:\n", resp)
	res.DocsFound = resp.Head.DocsFound
	res.DocsReturn = resp.Head.DocsReturn
	res.SearchTime = resp.Head.SearchTime

	if resp.Head.DocsReturn >= p.Limit {
		res.Offset = p.Offset + p.Limit
	}

	//id去重
	if p.IsRemoval {
		var productIdMap = map[string]bool{}
		for _, item := range resp.Auctions {
			if !productIdMap[item.ProductId] {
				productIdMap[item.ProductId] = true
				res.ProductIds = append(res.ProductIds, item.ProductId)
			}
		}
	} else {
		for _, item := range resp.Auctions {
			res.ProductIds = append(res.ProductIds, item.ProductId)
		}
	}

	return res, nil
}

type SearchBaseResp struct {
	Code      string `json:"Code"`
	Message   string `json:"Message"`
	RequestId string `json:"RequestId"`
}

func (m *ImageMill) formatBaseResponseError(response *responses.BaseResponse) SearchBaseResp {
	var resp SearchBaseResp
	_ = json.Unmarshal([]byte(response.GetHttpContentString()), &resp)
	//logger.Info("base:", response.GetHttpContentString())
	return resp
}

func (m *ImageMill) isLimitUsedCode(code string) bool {
	return code == CodeDeniedRequest || code == CodeThrottlingUser
}
