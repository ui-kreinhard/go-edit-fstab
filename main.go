package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/kr/pretty"
)

type MountPoint struct {
	Device     string
	MountPoint string
	FsType     string
	Options    string
	Dump       bool
	Pass       bool
	Removed    bool
}

func boolToBinaryString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func (m *MountPoint) toFstabLine() string {
	return fmt.Sprint(m.Device, "\t",
		m.MountPoint, "\t",
		m.FsType, "\t",
		m.Options, "\t",
		boolToBinaryString(m.Dump), "\t",
		boolToBinaryString(m.Pass),
	)
}

func getFormatHeader() string {
	return fmt.Sprint("# <device>\t<mountPoint>\t<fsType>\t<options>\t<dump>\t<pass>")
}

func (m *MountPoint) applyTmpfsTemplate() {
	if m.Device == "tmpfs" {
		m.FsType = "tmpfs"
		m.Options = "nosuid,nodev"
		m.Dump = false
		m.Pass = false
	}
}

func (m *MountPoint) Edit(expression string) error {
	operands := strings.Split(expression, "=")
	if len(operands) != 2 {
		return errors.New("invalid expression " + expression)
	}
	variable := operands[0]
	value := operands[1]
	switch variable {
	case "device":
		m.Device = value
		m.applyTmpfsTemplate()
	case "mountPoint":
		m.MountPoint = value
	case "fsType":
		m.FsType = value
	case "options":
		m.Options = value
	case "dump":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		m.Dump = v
	case "pass":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		m.Pass = v
	default:
		return errors.New("invalid var name " + variable)
	}
	return nil
}

var mountPoints []*MountPoint

func FromLine(fstabLine []string) (MountPoint, error) {
	if len(fstabLine) != 6 {
		return MountPoint{}, errors.New("invalid count" + strconv.Itoa(len(fstabLine)))
	}
	return MountPoint{
		fstabLine[0],
		fstabLine[1],
		fstabLine[2],
		fstabLine[3],
		fstabLine[4] == "1",
		fstabLine[5] == "1",
		false,
	}, nil
}

func GetMountPoints(rawFile string) []*MountPoint {
	mountPoints := []*MountPoint{}
	rawLines := strings.Split(string(rawFile), "\n")
	for _, rawLine := range rawLines {
		if strings.HasPrefix(rawLine, "#") {
			continue
		}
		mountPoint, err := FromLine(strings.Fields(string(rawLine)))
		if err != nil {
			log.Println(err, "skipping line", rawLine)
			continue
		}
		mountPoints = append(mountPoints, &mountPoint)
	}
	return mountPoints
}

func remove(mountPointToRemove string) error {
	for _, mountPoint := range mountPoints {
		if mountPoint.MountPoint == mountPointToRemove {
			mountPoint.Removed = true
			return nil
		}
	}
	return errors.New("no mountpoint " + mountPointToRemove + " found")
}

func editOrAdd(mountPointToEdit string, expression string) error {
	for _, mountPoint := range mountPoints {
		if mountPoint.MountPoint == mountPointToEdit {
			return mountPoint.Edit(expression)
		}
	}
	newMountPoint := MountPoint{}
	newMountPoint.MountPoint = mountPointToEdit
	err := newMountPoint.Edit(expression)
	if err != nil {
		return err
	}
	mountPoints = append(mountPoints, &newMountPoint)
	return nil
}

func strategy(commandIndex int, command string) (int, error) {
	switch command {
	case "edit":
		if commandIndex+1 >= len(os.Args) {
			return 0, errors.New("no mountpoint given")
		}
		mountPoint := os.Args[commandIndex+1]
		if commandIndex+2 >= len(os.Args) {
			return 0, errors.New("no expressiongiven")
		}
		expression := os.Args[commandIndex+2]
		return 2, editOrAdd(mountPoint, expression)
	case "remove":
		if commandIndex+1 >= len(os.Args) {
			return 0, errors.New("no mountpoint given")
		}
		mountPoint := os.Args[commandIndex+1]
		return 1, remove(mountPoint)
	default:
		return 0, errors.New("Unknown command " + command + " at " + strconv.Itoa(commandIndex))
	}
}

func GetFstabLines(mountPoints []*MountPoint) string {
	fstabLines := []string{
		getFormatHeader(),
	}
	for _, mountPoint := range mountPoints {
		if mountPoint.Removed {
			continue
		}
		fstabLines = append(fstabLines, mountPoint.toFstabLine())
	}
	return strings.Join(fstabLines, "\n")
}

func GetenvDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	sourceFstab := GetenvDefault("fstab", "/etc/fstab")
	targetFstab := GetenvDefault("targetFstab", "/etc/fstab")
	pretty.Println(sourceFstab)

	rawFile, err := ioutil.ReadFile(sourceFstab)
	if err != nil {
		log.Fatalln(err)
	}
	mountPoints = GetMountPoints(string(rawFile))

	skipCounter := 1

	for i, osArgsValue := range os.Args {
		if skipCounter > 0 {
			skipCounter--
			continue
		}
		skipCounter, err = strategy(i, osArgsValue)
		if err != nil {
			log.Fatalln(err)
		}
	}
	pretty.Println(GetFstabLines(mountPoints))
	err = os.WriteFile(targetFstab, []byte(pretty.Sprint(GetFstabLines(mountPoints))+"\n"), 0644)
	if err != nil {
		log.Fatalln(err)
	}
}
