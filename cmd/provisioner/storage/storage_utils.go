package storage

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"k8s-pv-provisioner/cmd/provisioner/config"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

var appConfig = config.GetInstance()

//ChooseBaseNameOfAsset is func which returns the base name of new storagge asset depending on input args
func ChooseBaseNameOfAsset(args ...string) string {
	result := make([]string, len(args))
	for index, item := range args {
		item = strings.Trim(item, "-")
		result[index] = item
	}

	result = append(result, "vol")
	return strings.Join(result, "-") //-> arg1-arg2...-argN-vol
}

func processOwnerAnnotation(pvc *v1.PersistentVolumeClaim, value string, defaultValue int) int {
	var result int
	var err error

	value, ok := pvc.Annotations[value]
	if ok {
		result, err = castToInt(value)
		if err != nil {
			klog.Warningf("PersistentVolumeClaim: %v annotation: %v could not parse value: %v: %v", pvc.Name, config.AnnotationOwnerNewAssetUID, value, err)
			result = defaultValue
		}
		return result
	}
	return defaultValue
}

//ChooseAssetOwner is func which decides of what uid gid attributes the new asset should have
func ChooseAssetOwner(pvc *v1.PersistentVolumeClaim) (int, int) {

	var uid, gid int

	currentStorageClass := appConfig.StorageClasses[*pvc.Spec.StorageClassName]

	uid = processOwnerAnnotation(pvc, config.AnnotationOwnerNewAssetUID, currentStorageClass.DefaultOwnerAssetUID) // Deprecated annotation
	if uid == currentStorageClass.DefaultOwnerAssetUID {
		uid = processOwnerAnnotation(pvc, config.AnnotationOwnerNewAssetUID1, currentStorageClass.DefaultOwnerAssetUID)
	}

	gid = processOwnerAnnotation(pvc, config.AnnotationOwnerNewAssetGID, currentStorageClass.DefaultOwnerAssetGID) // Deprecated annotation
	if gid == currentStorageClass.DefaultOwnerAssetGID {
		gid = processOwnerAnnotation(pvc, config.AnnotationOwnerNewAssetGID1, currentStorageClass.DefaultOwnerAssetGID)
	}

	return uid, gid
}

func castToInt(value string) (int, error) {
	id, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

//CreateStorageAsset is func which creates the storage asset
func CreateStorageAsset(assetPath string, uid, gid int, reuseExisting bool) error {

	if _, err := os.Stat(assetPath); err == nil && !reuseExisting {
		return fmt.Errorf("Storage asset: %v already exists", assetPath)
	}
	//If assetPath already exists os.MkdirAll does nothing
	if err := os.MkdirAll(assetPath, 0755); os.IsPermission(err) {
		return err
	}

	var action string
	if reuseExisting {
		action = "reused"
	} else {
		action = "created"
	}
	klog.Infof("Storage asset: %v was successfully %v", assetPath, action)

	if err := os.Chown(assetPath, uid, gid); err != nil {
		return err
	}
	klog.Infof("Storage asset: %v ownership was set as %v:%v", assetPath, uid, gid)

	return nil
}

//DeleteStorageAsset is func deleting the storage asset
func DeleteStorageAsset(assetPath string) error {

	err := os.RemoveAll(assetPath)
	if err != nil {
		return err
	}

	klog.Infof("Storage asset: %v was successfully deleted", assetPath)

	return nil
}
