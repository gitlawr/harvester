package image

import (
	"fmt"
	"net/http"

	"github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta1"
	"github.com/longhorn/longhorn-manager/types"
	v1 "github.com/rancher/wrangler/pkg/generated/controllers/storage/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	harvesterv1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	lhv1beta1 "github.com/harvester/harvester/pkg/generated/controllers/longhorn.io/v1beta1"
	"github.com/harvester/harvester/pkg/ref"
	"github.com/harvester/harvester/pkg/util"
)

const (
	optionBackingImageName = "backingImage"
	optionMigratable       = "migratable"

	longhornSystemNamespace = "longhorn-system"
)

// vmImageHandler syncs status on vm image changes, and manage a storageclass & a backingimage per vm image
type vmImageHandler struct {
	httpClient     http.Client
	storageClasses v1.StorageClassClient
	images         ctlharvesterv1.VirtualMachineImageClient
	backingImages  lhv1beta1.BackingImageClient
}

func (h *vmImageHandler) OnChanged(key string, image *harvesterv1.VirtualMachineImage) (*harvesterv1.VirtualMachineImage, error) {
	if image == nil || image.DeletionTimestamp != nil {
		return image, nil
	}
	if !harvesterv1.ImageInitialized.IsTrue(image) {
		return h.initialize(image)
	} else if image.Spec.URL != image.Status.AppliedURL {
		// URL is changed, recreate the storageclass and backingimage
		if err := h.backingImages.Delete(longhornSystemNamespace, getBackingImageName(image), &metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return image, err
		}
		if err := h.storageClasses.Delete(getImageStorageClassName(image.Name), &metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return image, err
		}
		return h.initialize(image)
	}
	return image, nil
}

func (h *vmImageHandler) OnRemove(key string, image *harvesterv1.VirtualMachineImage) (*harvesterv1.VirtualMachineImage, error) {
	if image == nil {
		return nil, nil
	}
	scName := getImageStorageClassName(image.Name)
	if err := h.storageClasses.Delete(scName, &metav1.DeleteOptions{}); !errors.IsNotFound(err) && err != nil {
		return image, err
	}
	return image, nil
}

func (h *vmImageHandler) initialize(image *harvesterv1.VirtualMachineImage) (*harvesterv1.VirtualMachineImage, error) {
	if err := h.createBackingImage(image); err != nil && !errors.IsAlreadyExists(err) {
		return nil, err
	}
	if err := h.createStorageClass(image); err != nil {
		return nil, err
	}

	toUpdate := image.DeepCopy()
	toUpdate.Status.AppliedURL = toUpdate.Spec.URL
	toUpdate.Status.StorageClassName = getImageStorageClassName(image.Name)
	harvesterv1.ImageInitialized.True(toUpdate)
	harvesterv1.ImageInitialized.Message(toUpdate, "")

	if image.Spec.SourceType == harvesterv1.VirtualMachineImageSourceTypeDownload {
		resp, err := h.httpClient.Head(image.Spec.URL)
		if err != nil {
			harvesterv1.ImageInitialized.False(toUpdate)
			harvesterv1.ImageInitialized.Message(toUpdate, err.Error())
			return h.images.Update(toUpdate)
		}
		defer resp.Body.Close()

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
			harvesterv1.ImageInitialized.False(toUpdate)
			harvesterv1.ImageInitialized.Message(toUpdate, fmt.Sprintf("got %d status code from %s", resp.StatusCode, image.Spec.URL))
			return h.images.Update(toUpdate)
		}

		if resp.ContentLength > 0 {
			toUpdate.Status.Size = resp.ContentLength
		}
	} else {
		toUpdate.Status.Progress = 0
	}

	return h.images.Update(toUpdate)
}

func (h *vmImageHandler) createBackingImage(image *harvesterv1.VirtualMachineImage) error {
	bi := &v1beta1.BackingImage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getBackingImageName(image),
			Namespace: longhornSystemNamespace,
			Annotations: map[string]string{
				util.AnnotationImageID: ref.Construct(image.Namespace, image.Name),
			},
		},
		Spec: types.BackingImageSpec{
			SourceType:       types.BackingImageDataSourceType(image.Spec.SourceType),
			SourceParameters: map[string]string{},
		},
	}
	if image.Spec.SourceType == harvesterv1.VirtualMachineImageSourceTypeDownload {
		bi.Spec.SourceParameters[types.DataSourceTypeDownloadParameterURL] = image.Spec.URL
	}

	_, err := h.backingImages.Create(bi)
	return err
}

func (h *vmImageHandler) createStorageClass(image *harvesterv1.VirtualMachineImage) error {
	recliamPolicy := corev1.PersistentVolumeReclaimDelete
	volumeBindingMode := storagev1.VolumeBindingImmediate
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: getImageStorageClassName(image.Name),
		},
		Provisioner:          types.LonghornDriverName,
		ReclaimPolicy:        &recliamPolicy,
		AllowVolumeExpansion: pointer.BoolPtr(true),
		VolumeBindingMode:    &volumeBindingMode,
		Parameters: map[string]string{
			types.OptionNumberOfReplicas:    "3",
			types.OptionStaleReplicaTimeout: "30",
			optionMigratable:                "true",
			optionBackingImageName:          image.Name,
		},
	}

	_, err := h.storageClasses.Create(sc)
	return err
}

func getImageStorageClassName(imageName string) string {
	return fmt.Sprintf("longhorn-%s", imageName)
}

func getBackingImageName(image *harvesterv1.VirtualMachineImage) string {
	return fmt.Sprintf("%s-%s", image.Namespace, image.Name)
}
