package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/kr/pretty"
	"github.com/ui-kreinhard/go-edit-fstab/mounts"
)

var mountPoints []*mounts.MountPoint

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
	newMountPoint := mounts.MountPoint{}
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
	mountPoints = mounts.GetMountPoints(string(rawFile))

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
	pretty.Println(mounts.GetFstabLines(mountPoints))
	err = os.WriteFile(targetFstab, []byte(pretty.Sprint(mounts.GetFstabLines(mountPoints))+"\n"), 0644)
	if err != nil {
		log.Fatalln(err)
	}
}
