package utils

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/phachon/mm-wiki/global"
	_ "github.com/phachon/mm-wiki/global"
	"time"
)
import "net/http"

var (
	// 强制性提示内容 {是否有新版本, 是否是重要版本, 更新主要内容, 当前版本号, 新版本号, 下载链接, GITHUB地址}
	VersionUpgrade = map[string]interface{}{
		// 是否有新版本, 有新版本, 则展示, 否则不展示
		"hasVewVersion": false,
		// 如果是非常重要的版本显示在最上面, 并标红
		"isImportant": false,
		"currVersion": "0.0.0",
		"newVersion":  "0.0.0",
		// 升级的内容
		"description": "<li>1.XX</li><li> 2.xx</li><li> 2.xx</li><li> 2.xx</li><li> 2.xx</li><li> 2.xx</li><li> 2.xx</li><li> 2.xx</li><li> 2.xx</li><li> 2.xx</li><li> 2.xx</li><li> 2.xx</li><li> 2.xx</li>",
		// [未实现]下载字段: 下载名称, 下载地址, 格式: 下载名称--下载地址;下载名称--下载地址;
		"downloadUrl": "download1--http://xxx/d.zip;download2--http://xxx/d.zip;",
		// 引导至github下载
		"github": "https://github.com/phachon/mm-wiki/releases",
	}
)

/*
	错误码:
	0: 正确
	1: 常用错误
	10:
*/
type VersionResult struct {
	Code        int         `json:"code"`
	Success     bool        `json:"success"`
	Data        VersionData `json:"data"`
	Description string      `json:"description"`
}

type VersionData struct {
	HasNewVersion bool              `json:"hasNewVersion"`
	CurrVersion   string            `json:"currVersion"`
	NewVersion    string            `json:"newVersion"`
	IsImportant   bool              `json:"isImportant"`
	Description   string            `json:"description"`
	DownloadUrl   []VersionDownData `json:"downloadUrl"`
}

type VersionDownData struct {
	// ID, 自增长等, 唯一ID
	Id int64 `json:"id"`
	// 下载包大小
	Size int64 `json:"size"`
	// 下载次数
	DownloadCount int64 `json:"downloadCount"`
	// 包名称
	Name string `json:"name"`
	// 下载地址
	Url string `json:"url"`
}

// 启动检查版本
func StartCheckVersion() {
	// 是否启动时检查版本号, 0为不检查, 1为检查(默认)
	versionCheck, _ := beego.AppConfig.Int("version::version_check")
	// 是否定时检查版本号, 0为不启动定时(默认), 1为每晚12点检查, 大于14400为多少秒后检查(最低不能低14400秒4个小时)
	versionTimer, _ := beego.AppConfig.Int("version::version_timer")
	if versionCheck == 1 && versionTimer == 0 {
		checkVersion()
	}

	if versionTimer > 0 {
		startTimer(checkVersion)
	}
}

/**
检查版本主方法
TODO 检查时上传用户的基础信息 操作系统, CPU, 网卡, 内存, 硬盘, 硬件指纹等
TODO 用户设置是否跳过检查及跳过当前版本检查
*/
func checkVersion() {
	// 调用主动检查
	GetHasNewVersion()
}

func GetHasNewVersion() (result VersionResult) {
	currVersion := global.SYSTEM_VERSION

	// 远程检查地址, 0为GitHub(默认), 1为mm-wiki服务检查
	versionFrom, _ := beego.AppConfig.Int("version_from")

	var versionResult VersionResult
	if versionFrom == 0 {
		versionResult, _ = GetVersionFromGitHub()
	} else {
		versionResult, _ = GetVersionFromServe()
	}

	var newVersionData VersionData

	if versionResult.Success {
		newVersionData = versionResult.Data
	} else {
		fmt.Printf("获取版本数据错误, 错误码:%d, 错误信息:%s \r\n", versionResult.Code, versionResult.Description)
		return versionResult
	}

	// 默认为无新版本
	VersionUpgrade["hasVewVersion"] = false
	if newVersionData.NewVersion != "" {
		compare := NewVersionCompare("v")
		isLt := compare.Lt(currVersion, newVersionData.NewVersion)

		isHaveNewVersion := "没有新版本"
		if isLt {
			isHaveNewVersion = "有新版本"
			newVersionData.HasNewVersion = true
			result.Data = newVersionData
			result.Code = 0
			result.Success = true
			result.Description = "获取新版本数据成功!"

			// 全局强制性提示, 强制性提示, 0不开启, 1开始强制性提示(会在顶部导航栏显示有升级提示)
			versionTipForce, _ := beego.AppConfig.Int("version_tip_force")
			if versionTipForce == 1 {
				VersionUpgrade["hasVewVersion"] = true
				VersionUpgrade["isImportant"] = newVersionData.IsImportant
				VersionUpgrade["newVersion"] = newVersionData.NewVersion
				VersionUpgrade["description"] = newVersionData.Description
				VersionUpgrade["downloadUrl"] = newVersionData.DownloadUrl
			}

			fmt.Printf("版本号检查结果:%s, 当前版本号:%s, 新版本号:%s \r\n", isHaveNewVersion, currVersion, newVersionData.NewVersion)
		} else {
			newVersionData.HasNewVersion = false
			result.Data = newVersionData
			result.Code = 10
			result.Success = false
			result.Description = "没有新的版本!"
			fmt.Printf("版本号检查结果:%s, 当前版本号:%s \r\n", isHaveNewVersion, currVersion)
		}
	} else {
		newVersionData.HasNewVersion = false
		result.Data = newVersionData
		result.Code = 2
		result.Success = false
		result.Description = "获取版本数据错误, 请联系管理员"
	}

	return result
}

// 向服务器请求是否有新的版本号
func GetVersionFromServe() (result VersionResult, err error) {
	versionUrl := "http://127.0.0.1:3000"

	resp, err := http.Get(versionUrl)
	if err != nil {
		fmt.Println("Get mm-wiki version from serve failed, reason:", err)
		return result, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println(fmt.Errorf("Get mm-wiki version from serve failed: %s", resp.Status))
		return result, err
	}

	// 转化为JSON数据
	var versionResult VersionResult
	if err := json.NewDecoder(resp.Body).Decode(&versionResult); err != nil {
		return result, err
	}

	return versionResult, nil
}

func GetVersionFromGitHub() (result VersionResult, err error) {
	versionUrl := "https://api.github.com/repos/phachon/mm-wiki/releases/latest"

	resp, err := http.Get(versionUrl)
	if err != nil {
		fmt.Println("Get mm-wiki version from github failed, reason:", err)
		result.Code = 1
		result.Description = "连接github失败或获取数据失败!"
		result.Success = false
		return result, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println(fmt.Errorf("Get mm-wiki version from github failed: %s", resp.Status))
		return result, err
	}

	// 转化为JSON数据
	var versionResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&versionResult); err != nil {
		return result, err
	}

	result.Code = 0
	result.Success = true
	result.Description = "从GitHub获取最新的版本信息正确"

	// 发布分支
	targetCommitish := versionResult["target_commitish"].(string)
	// 是否是预发布版本
	prerelease := versionResult["prerelease"].(bool)

	if !prerelease && targetCommitish == "master" {
		var versionData VersionData
		versionData.NewVersion = versionResult["tag_name"].(string)
		versionData.Description = versionResult["body"].(string)

		// 处理下载
		downAssets := versionResult["assets"]

		var versionDownDataArr = make([]VersionDownData, len(downAssets.([]interface{})))
		for index, item := range downAssets.([]interface{}) {
			itemMap := item.(map[string]interface{})
			var versionDownData VersionDownData
			idFloat := itemMap["id"].(float64)
			sizeFloat := itemMap["size"].(float64)
			downloadCountFloat := itemMap["download_count"].(float64)
			versionDownData.Id = int64(idFloat)
			versionDownData.Name = itemMap["name"].(string)
			versionDownData.Size = int64(sizeFloat)
			versionDownData.DownloadCount = int64(downloadCountFloat)
			versionDownData.Url = itemMap["browser_download_url"].(string)
			versionDownDataArr[index] = versionDownData
		}
		versionData.DownloadUrl = versionDownDataArr
		result.Data = versionData
	}

	return result, nil
}

// 定时器，启动的时候执行一次，以后每天晚上12点执行
func startTimer(fun func()) {
	// 是否定时检查版本号, 0为不启动定时(默认), 1为每晚12点检查, 大于14400为多少秒后检查(最低不能低14400秒4个小时)
	versionTimer, _ := beego.AppConfig.Int("version::version_timer")
	go func() {
		for {
			fun()
			now := time.Now()

			// 每20秒执行一次
			//next := now.Add(time.Second * 30)
			//next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), next.Minute(), next.Second(), 0, next.Location())
			// 计算下一个零点
			var next time.Time
			if versionTimer == 1 {
				next = now.Add(time.Hour * 24)
			}

			if versionTimer >= 14400 {
				duration := time.Second * time.Duration(versionTimer)
				next = now.Add(duration)
			}

			next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
			t := time.NewTimer(next.Sub(now))
			fmt.Printf("版本检查计时器启动, 下一次检查时间:%s \r\n", next.Format("2006年01月02日 15时04分05秒"))
			<-t.C
		}
	}()
}
