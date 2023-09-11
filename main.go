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
	IsForceStop    bool
	BackupApk      bool
	usingGZ        bool
}

func NewDataManager(adbPath string) *DataManager {
	ptr := new(DataManager)
	ptr.cmd = dcmd.NewDCmd(adbPath)
	ptr.BackupFilePath = "/data/local/tmp"
	ptr.BusyboxPath = ""
	ptr.AdbPath = adbPath
	ptr.IsForceStop = true
	ptr.BackupApk = false
	ptr.usingGZ = false
	return ptr
}
func (that *DataManager) SetForceStop(forceStop bool) {
	that.IsForceStop = forceStop
}
func (that *DataManager) SetBackupApk(backupApk bool) {
	that.BackupApk = backupApk
}
func (that *DataManager) SetUsingGZ(usingGZ bool) {
	that.usingGZ = usingGZ
}

//tar link: https://www.cnblogs.com/TreeDream/p/10277075.html
func (that *DataManager) Backup(packageName string) error {
	prefix := ".tar"
	if that.usingGZ {
		prefix += ".gz"
	}
	savePath := fmt.Sprintf("%s/HBackup_%s_%s_%s_%s%s",
		that.fixPath(that.BackupFilePath),
		packageName,
		that.fixPath(that.getVersionName(packageName)),
		that.fixPath(that.getModel()),
		time.Now().Format("20060102_150405"),
		prefix)
	commands := make([]string, 0)
	if that.IsForceStop {
		commands = append(commands, fmt.Sprintf("am force-stop %s", packageName))
	}
	commands = append(commands,
		fmt.Sprintf("%s rm -rf /data/data/.external.%s", that.BusyboxPath, packageName),
		fmt.Sprintf("%s ln -sf \"/sdcard/Android/data/%s\" \"/data/data/.external.%s\"", that.BusyboxPath, packageName, packageName))
	apkPathLink := ""
	if that.BackupApk {
		errPtr := new(error)
		apkPath := dcmd.Exec_(that.AdbPath+" shell pm path "+packageName, errPtr)
		if *errPtr != nil {
			return *errPtr
		}
		apkPath = strings.TrimSpace(apkPath)
		apkPath = strings.Split(apkPath, ":")[1]
		log.Println("apk path:", apkPath)
		apkPathLink = fmt.Sprintf("\"/data/data/%s_%s.apk\"", packageName, that.fixPath(that.getVersionName(packageName)))
		commands = append(commands,
			fmt.Sprintf("%s ln -sfn \"%s\" %s", that.BusyboxPath, apkPath, apkPathLink))

	}
	options := "-ch"
	if that.usingGZ {
		options += "z"
	}
	commands = append(commands,
		fmt.Sprintf("%s tar %s \"/data/data/%s/\" \"/data/data/.external.%s/.\" %s --exclude \"data/data/%s/lib/\" > \"%s\"", that.BusyboxPath, options, packageName, packageName, apkPathLink, packageName, savePath))

	commands = append(commands,
		fmt.Sprintf("%s rm -rf \"/data/data/.external.%s\"", that.BusyboxPath, packageName))

	if that.BackupApk {
		commands = append(commands,
			fmt.Sprintf("%s rm -rf %s", that.BusyboxPath, apkPathLink))
	}
	if that.IsForceStop {
		commands = append(commands, fmt.Sprintf("am force-stop %s", packageName))
	}
	commands = append(commands, fmt.Sprintf("%s chown media_rw:media_rw \"%s\"", that.BusyboxPath, savePath))

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
	var isForceStop bool
	var backupApk bool
	var usingGZ bool
	flag.StringVar(&adbPath, "a", "adb", "adb path")
	flag.StringVar(&serialNumber, "s", "", "device serial number")
	flag.BoolVar(&isForceStop, "f", true, "force stop target app")
	flag.BoolVar(&backupApk, "c", false, "backup apk file")
	flag.BoolVar(&usingGZ, "g", false, "backup file with gz")
	flag.Parse()
	if flag.NArg() != 1 || flag.Arg(0) == "" {
		println("The packageName must be specified.\neg:\nhbackup [-a=adb path] [-s=serial number] [-c=true/false] [-f=true/false] [-g=false/true] your.package.name")
		return
	}
	packageName = flag.Arg(0)

	if serialNumber != "" {
		adbPath += " -s " + serialNumber
	}
	dataManager := NewDataManager(adbPath)
	dataManager.SetForceStop(isForceStop)
	dataManager.SetBackupApk(backupApk)
	dataManager.SetUsingGZ(usingGZ)
	err := dataManager.Backup(packageName)
	if err != nil {
		println("HBackup error:", err)
	} else {
		println("HBackup finish")
	}

}
