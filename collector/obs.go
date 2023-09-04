// --------------------
// File: obs.go
// Project: collector
// Purpose: 采集华为云OBS bucket资源使用情况信息
// Author: Jan Lam (honos628@foxmail.com)
// Last Modified: 2021-08-01 23:27:21
// --------------------

package collector

import (
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/zoulele123/obs_exporter/config"
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

type BucketInfo struct {
	Name         string
	ObjectNumber int
	Size         int64
	Quota        int64
	InUsedPec    float64
	Writable     bool
	Readable     bool
}

type ObsCollector struct {
	ObsClient  *obs.ObsClient
	TotelCount int
	Buckets    []*BucketInfo
}

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// 生成随机数
func randSeq(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// 初始化
func InitObsClient(endpoint, ak, sk string) *ObsCollector {
	obsClient, err := obs.New(ak, sk, endpoint)
	if err != nil {
		panic(err)
	}
	return &ObsCollector{ObsClient: obsClient}
}

// 获取单个Bucket的信息
func (c *ObsCollector) GetBucketInfo(bucketName string, results chan *BucketInfo, limiter chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		<-limiter
		wg.Done()
	}()
	defer func() {
		if err := recover(); err != nil {
			log.Printf("GetBucketInfo %s failed! recover: %v", bucketName, err)
		}
	}()
	limiter <- struct{}{}
	storageInfo, err := c.ObsClient.GetBucketStorageInfo(bucketName)
	if err != nil {
		// log.Printf(bucketName, "get strageInfo err", err)
		return
	}
	quota, err := c.ObsClient.GetBucketQuota(bucketName)
	if err != nil {
		// log.Printf(bucketName, "get quota failed", err)
		return
	}
	bucketInfo := &BucketInfo{
		Name:         bucketName,
		ObjectNumber: storageInfo.ObjectNumber,
		Size:         storageInfo.Size,
		Quota:        quota.BucketQuota.Quota,
	}

	rand.Seed(time.Now().UnixNano())
	content := randSeq(4)
	fileName := "__obs_exporter_test.txt"
	//读测试
	bucketInfo.Writable = c.WriteTest(bucketName, fileName, content)
	bucketInfo.Readable = c.ReadTest(bucketName, fileName, content)

	switch {
	case bucketInfo.Quota > 0:
		bucketInfo.InUsedPec = float64(bucketInfo.Size) / float64(bucketInfo.Quota) * 100
	default:
		bucketInfo.InUsedPec = 0

	}
	results <- bucketInfo
}

// 获取所有Buckets的信息
func (c *ObsCollector) GetAllBucketsInfo() {
	// log.Printf("start!")

	// panic恢复, 避免获取不了信息的时候 页面无返回
	defer func() {
		if err := recover(); err != nil {
			log.Printf("GetAllBucketsInfo failed! recover: %v", err)
		}
	}()
	r, err := c.ObsClient.ListBuckets(nil) // 获取所有Buckets基本信息
	if err != nil {
		log.Println(err)
		return
	}
	c.TotelCount = len(r.Buckets)

	// 并行处理
	results := make(chan *BucketInfo, c.TotelCount)
	limiter := make(chan struct{}, 30) // 并发限制50
	wg := sync.WaitGroup{}
	for _, v := range r.Buckets {
		wg.Add(1)
		go c.GetBucketInfo(v.Name, results, limiter, &wg)
	}
	wg.Wait() // 等待所有Buckets检索再统计
	close(results)
	for elem := range results {
		c.Buckets = append(c.Buckets, elem)
	}
}

// 去除重复BucketInfo
func removeDuplicateElement(s []*BucketInfo) []*BucketInfo {
	result := make([]*BucketInfo, 0, len(s))
	temp := map[string]struct{}{}
	for _, item := range s {
		if _, ok := temp[item.Name]; !ok {
			temp[item.Name] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

//检索所有账号
func ScrapeAll(cfg *config.Config) []*BucketInfo {
	var bucketInfos []*BucketInfo
	wg := sync.WaitGroup{}
	results := make(chan []*BucketInfo, 50)
	for _, v := range cfg.ObsAccounts {
		wg.Add(1)
		func() {
			conn := InitObsClient(v.Endpoint, v.Ak, v.Sk)
			defer conn.ObsClient.Close() //用完关闭obsclient 防止占用系统连接数
			conn.GetAllBucketsInfo()
			results <- conn.Buckets
			wg.Done()
		}()
	}
	wg.Wait()
	close(results)
	for elem := range results {
		bucketInfos = append(bucketInfos, elem...)
	}
	bucketInfos = removeDuplicateElement(bucketInfos)
	return bucketInfos
}

//对象存储写测试
func (c *ObsCollector) WriteTest(bucketName, fileName, content string) bool {
	input := &obs.PutObjectInput{}
	input.Bucket = bucketName
	input.Key = fileName
	input.Body = strings.NewReader(content)
	_, err := c.ObsClient.PutObject(input)
	return err == nil
}

//对象存储读测试
func (c *ObsCollector) ReadTest(bucketName, fileName, content string) bool {
	input := &obs.GetObjectInput{}
	input.Bucket = bucketName
	input.Key = fileName
	output, err := c.ObsClient.GetObject(input)
	if err == nil {
		defer output.Body.Close()
		p := make([]byte, 4)
		var readErr error
		var readCount int
		// 读取对象内容
		var payload string
		for {
			readCount, readErr = output.Body.Read(p)
			if readCount > 0 {
				payload += string(p[:readCount])
			}
			if readErr != nil {
				break
			}
		}
		if content == payload {
			return true
		}
	}
	return false
}
