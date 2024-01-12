package mounts

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
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

func getFormatHeader() string {
	return fmt.Sprint("# <device>\t<mountPoint>\t<fsType>\t<options>\t<dump>\t<pass>")
}

func GetFstabLines(mountPoints []*MountPoint) string {
	fstabLines := []string{
		getFormatHeader(),
	}
	for _, mountPoint := range mountPoints {
		if mountPoint.Removed {
			continue
		}
		mountPoint.applyTmpfsTemplate()
		fstabLines = append(fstabLines, mountPoint.toFstabLine())
	}
	return strings.Join(fstabLines, "\n")
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
