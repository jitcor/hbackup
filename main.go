package main

import (
	"flag"
	"fmt"
	"github.com/Humenger/go-devcommon/dcmd"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DataManager struct {
	cmd            *dcmd.DCmd
	BackupFilePath string
	BusyboxPath    string
	AdbPath        string
}

func NewDataManager(adbPath string) *DataManager {
	ptr := new(DataManager)
	ptr.cmd = dcmd.NewDCmd(adbPath)
	ptr.BackupFilePath = "/data/local/tmp"
	ptr.BusyboxPath = ""
	ptr.AdbPath = adbPath
	return ptr
}
func (that *DataManager) Backup(packageName string) error {
	savePath := fmt.Sprintf("%s/HBackup_%s_%s_%s_%s.zip",
		that.fixPath(that.BackupFilePath),
		packageName,
		that.fixPath(that.getVersionName(packageName)), that.fixPath(that.getModel()), time.Now().Format("20060102_150405"))
	commands := []string{
		fmt.Sprintf("am force-stop %s", packageName),
		fmt.Sprintf("%s rm -rf /data/data/.external.%s", that.BusyboxPath, packageName),
		fmt.Sprintf("%s ln -sf \"/sdcard/Android/data/%s\" \"/data/data/.external.%s\"", that.BusyboxPath, packageName, packageName),
		fmt.Sprintf("%s tar -c \"/data/data/%s/\" \"/data/data/.external.%s/.\" --exclude \"data/%s/lib/\" > \"%s\"", that.BusyboxPath, packageName, packageName, that.BusyboxPath, savePath),
		fmt.Sprintf("%s rm -rf \"/data/data/.external.%s\"", that.BusyboxPath, packageName),
		fmt.Sprintf("am force-stop %s", packageName),
		fmt.Sprintf("%s chown media_rw:media_rw \"%s\"", that.BusyboxPath, savePath),
	}
	err := that.ExecCommands(commands...)
	if err != nil {
		return err
	}
	errPtr := new(error)
	result := dcmd.Exec_(that.AdbPath+" pull "+savePath, errPtr)
	if *errPtr != nil {
		return *errPtr
	}
	log.Println("result:", result)
	err = that.ExecCommands(fmt.Sprintf("%s rm -rf \"%s\"", that.BusyboxPath, savePath))
	if err != nil {
		return err
	}
	return nil
}
func (that *DataManager) ExecCommands(commands ...string) error {
	errPtr := new(error)
	for _, command := range commands {
		result := dcmd.Exec_(that.AdbPath+" shell su -c '"+command+"'", errPtr)
		if *errPtr != nil {
			return *errPtr
		}
		log.Println("result:", result)
	}
	return nil
}
func (that *DataManager) getVersionCode(packageName string) int {
	errPtr := new(error)
	result := dcmd.Exec_(that.AdbPath+" shell dumpsys package "+packageName+" | grep versionCode", errPtr)
	if *errPtr != nil {
		return -1
	}
	log.Println("result:", result)
	result = strings.TrimSpace(result)
	if result != "" {
		re, err := regexp.Compile("versionCode=(\\d*?) ")
		if err != nil {
			return -1
		}
		versionCode, err := strconv.Atoi(re.FindStringSubmatch(result)[1])
		if err != nil {
			return -1
		}
		return versionCode
	}
	return -1
}
func (that *DataManager) getVersionName(packageName string) string {
	errPtr := new(error)
	result := dcmd.Exec_(that.AdbPath+" shell dumpsys package "+packageName+" | grep versionName", errPtr)
	if *errPtr != nil {
		return ""
	}
	log.Println("result:", result)
	result = strings.TrimSpace(result)
	if result != "" {
		re, err := regexp.Compile("versionName=(.*)")
		if err != nil {
			log.Println("error:", err)
			return ""
		}
		return re.FindStringSubmatch(result)[1]
	}
	return ""
}
func (that *DataManager) getModel() string {
	errPtr := new(error)
	result := dcmd.Exec_(that.AdbPath+" shell getprop ro.product.model", errPtr)
	if *errPtr != nil {
		return ""
	}
	return strings.TrimSpace(result)
}
func (that *DataManager) fixPath(path string) string {
	return strings.ReplaceAll(path, " ", "-")
}

func (that *DataManager) PathExists(path string) bool {
	errPtr := new(error)
	result := dcmd.Exec_(that.AdbPath+" shell su -c ls "+path, errPtr)
	if *errPtr != nil {
		return false
	}
	log.Println("result:", result)
	return !strings.Contains(result, "No such file or directory")
}

func (that *DataManager) CreateDir(path string) error {
	errPtr := new(error)
	result := dcmd.Exec_(that.AdbPath+" shell su -c mkdir -p "+path, errPtr)
	log.Println("result:", result)
	return *errPtr
}

func main() {
	adbPath := "adb"
	var packageName string
	var serialNumber string
	flag.StringVar(&adbPath, "a", "adb", "adb path")
	flag.StringVar(&serialNumber, "s", "", "device serial number")
	flag.Parse()
	if flag.NArg() != 1 || flag.Arg(0) == "" {
		println("The packageName must be specified.\neg:\nhbackup xxx.xxx.xxx")
		return
	}
	packageName = flag.Arg(0)

	if serialNumber != "" {
		adbPath += " -s " + serialNumber
	}
	err := NewDataManager(adbPath).Backup(packageName)
	if err != nil {
		println("HBackup error:", err)
	} else {
		println("HBackup finish")
	}

}