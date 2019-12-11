package image_mill

import (
	"fmt"
	"strings"
	"testing"
)

//阿里云图片搜索下的配置
const (
	RegionId        = "cn-shanghai"
	EndPoint        = "imagesearch.cn-shanghai.aliyuncs.com"
	AccessKeyId     = "xxxxxxxxxx"                   //使用自己的accesskey
	AccessKeySecret = "xxxxxxxxxxxxxxxxxxxxxxxxxxxx" //使用自己的accesskeysecret

	InstanceId    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" //实例id
	InstanceName1 = "xxxxxx"                               //实例名称
)

const (
	CategoryFurniture = "9" //家具类型
)

func TestImageMill(t *testing.T) {
	prefix := "https://i.henghajiang.com/"
	conf := &Conf{
		RegionId:        RegionId,
		ProductId:       InstanceId,
		Endpoint:        EndPoint,
		AccessKeyId:     AccessKeyId,
		AccessKeySecret: AccessKeySecret,
		DownloadUrlFunc: func(image string) string {
			if !(len(image) > 4 && image[0:4] == "http") {
				image = prefix + image
				if !strings.Contains(image, "?") {
					image = image + "?imageslim|imageView2/0/w/750/h/750"
				}
			} else if strings.Contains(image, prefix) && !strings.Contains(image, "?") {
				image = image + "?imageslim|imageView2/0/w/750/h/750"
			}
			fmt.Println("image:", image)
			return image
		},
	}
	mill, err := New(conf)
	if err != nil {
		t.Fatal(err)
	}
	attr := "test"
	attach := &ImageAttach{
		InstanceName: InstanceName1,
		CategoryId:   CategoryFurniture,
		StrAttr:      attr,
	}

	images := []ImageItem{
		{
			PicName:   "id1",
			ProductId: "p1",
			ImageUrl:  "Fsge91Z-SLho_w8luD5Z9ue8caEn",
		}, {
			PicName:   "id2",
			ProductId: "p2",
			ImageUrl:  "Fq-uNn_1VVaVbZaHKiIFLpY4in2K",
		},
	}
	ids, err := mill.AddImage(images, attach)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("AddImage:", ids)

	p := SearchParam{
		ImageUrl:     "Fq-uNn_1VVaVbZaHKiIFLpY4in2K",
		Offset:       0,
		Limit:        0,
		StrAttr:      attr,
		IsRemoval:    true,
		InstanceName: InstanceName1,
		CategoryId:   CategoryFurniture,
	}
	res, err := mill.SearchImage(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("SearchImage1:", res)

	images = []ImageItem{
		{
			PicName:   "id1",
			ProductId: "p1",
			ImageUrl:  "",
		},
	}
	ids, err = mill.DeleteImage(images, attach)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("DeleteImage:", ids)

	p = SearchParam{
		ImageUrl:     "Fq-uNn_1VVaVbZaHKiIFLpY4in2K",
		Offset:       0,
		Limit:        0,
		StrAttr:      attr,
		IsRemoval:    true,
		InstanceName: InstanceName1,
		CategoryId:   CategoryFurniture,
	}
	res, err = mill.SearchImage(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("SearchImage2:", res)

	p = SearchParam{
		ImageUrl:     "Fq-uNn_1VVaVbZaHKiIFLpY4in2K",
		Offset:       0,
		Limit:        0,
		StrAttr:      attr,
		IsRemoval:    true,
		InstanceName: InstanceName1,
		CategoryId:   CategoryFurniture,
	}
	res, err = mill.SearchImage(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("SearchImage3:", res)

	p = SearchParam{
		ImageUrl:     "Fp8Q1c7-YY9FocIg8su9cIWDekU_",
		Offset:       0,
		Limit:        0,
		StrAttr:      attr,
		IsRemoval:    true,
		InstanceName: InstanceName1,
		CategoryId:   CategoryFurniture,
	}
	res, err = mill.SearchImage(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("SearchImage3:", res)

	images = []ImageItem{
		{
			PicName:   "id2",
			ProductId: "p2",
			ImageUrl:  "",
		},
	}
	ids, err = mill.DeleteImage(images, attach)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("DeleteImage:", ids)
}
