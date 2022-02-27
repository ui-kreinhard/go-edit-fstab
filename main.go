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
	device     string
	mountPoint string
	fsType     string
	options    string
	dump       bool
	pass       bool
	removed    bool
}

func boolToBinaryString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func (m *MountPoint) toFstabLine() string {
	return fmt.Sprint(m.device, "\t",
		m.mountPoint, "\t",
		m.fsType, "\t",
		m.options, "\t",
		boolToBinaryString(m.dump), "\t",
		boolToBinaryString(m.pass),
	)
}

func getFormatHeader() string {
	return fmt.Sprint("# <device>\t<mountPoint>\t<fsType>\t<options>\t<dump>\t<pass>")
}

func (m *MountPoint) applyTmpfsTemplate() {
	if m.device == "tmpfs" {
		m.fsType = "tmpfs"
		m.options = "nosuid,nodev"
		m.dump = false
		m.pass = false
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
		m.device = value
		m.applyTmpfsTemplate()
	case "mountPoint":
		m.mountPoint = value
	case "fsType":
		m.fsType = value
	case "options":
		m.options = value
	case "dump":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		m.dump = v
	case "pass":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		m.pass = v
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

func getMountPoints(rawFile string) []*MountPoint {
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
		if mountPoint.mountPoint == mountPointToRemove {
			mountPoint.removed = true
			return nil
		}
	}
	return errors.New("no mountpoint " + mountPointToRemove + " found")
}

func editOrAdd(mountPointToEdit string, expression string) error {
	for _, mountPoint := range mountPoints {
		if mountPoint.mountPoint == mountPointToEdit {
			return mountPoint.Edit(expression)
		}
	}
	newMountPoint := MountPoint{}
	newMountPoint.mountPoint = mountPointToEdit
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

func getFstabLines(mountPoints []*MountPoint) string {
	fstabLines := []string{
		getFormatHeader(),
	}
	for _, mountPoint := range mountPoints {
		if mountPoint.removed {
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
	mountPoints = getMountPoints(string(rawFile))

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
	pretty.Println(getFstabLines(mountPoints))
	err = ioutil.WriteFile(targetFstab, []byte(pretty.Sprint(getFstabLines(mountPoints))+"\n"), 0644)
	if err != nil {
		log.Fatalln(err)
	}
}
