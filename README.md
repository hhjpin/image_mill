# image_mill 图像搜索简化库

依赖于阿里云的图像搜索服务，图像搜索的相关接口有频率限制，此包自动处理频率限制的问题，只需简单引用即可。

此包榨干频率限制，并自动处理超频错误；对图片缓存防止超频导致的多次下载，对图片开启多协程下载，有超时重试机制。

## 使用

可以参考测试样例 `image_mill_test.go`

```go
//初始化配置
prefix := "https://i.henghajiang.com/"
conf := &Conf{
    RegionId:        RegionId,    //"cn-shanghai"
    ProductId:       InstanceId,  //图像搜索实例id
    Endpoint:        EndPoint,    //"imagesearch.cn-shanghai.aliyuncs.com"
    AccessKeyId:     AccessKeyId, //阿里云key
    AccessKeySecret: AccessKeySecret, //阿里云secret
    DownloadUrlFunc: func(image string) string { //下载时的处理，可以下载符合图像搜索的尺寸和大小
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
    InstanceName: InstanceName1,     //实例名称
    CategoryId:   CategoryFurniture, //图片的类型
    StrAttr:      attr,              //图片的属性
}

images := []ImageItem{
    {
        PicName:   "id1", //图片名称，一般图片的id等，唯一表示图片的值
        ProductId: "p1",  //图片关联的产品id
        ImageUrl:  "Fsge91Z-SLho_w8luD5Z9ue8caEn", //可以只存储图片hash等特征值，下载回调里面完善链接即可
    }, {
        PicName:   "id2",
        ProductId: "p2",
        ImageUrl:  "Fq-uNn_1VVaVbZaHKiIFLpY4in2K",
    },
}
ids, err := mill.AddImage(images, attach) //添加图片索引
if err != nil {
    t.Fatal(err)
}
t.Log("AddImage:", ids)

images = []ImageItem{
    {
        PicName:   "id1",
        ProductId: "p1",
        ImageUrl:  "",
    },
}
ids, err = mill.DeleteImage(images, attach) //删除图片索引
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
res, err = mill.SearchImage(p) //搜索图片
if err != nil {
    t.Fatal(err)
}
t.Log("SearchImage:", res)
```

## License

MIT