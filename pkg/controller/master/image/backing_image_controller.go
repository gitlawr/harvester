package image

import (
	"reflect"

	"github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta1"
	"github.com/longhorn/longhorn-manager/types"

	v1beta12 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	harvesterv1beta1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	lhv1beta1 "github.com/harvester/harvester/pkg/generated/controllers/longhorn.io/v1beta1"
	"github.com/harvester/harvester/pkg/ref"
	"github.com/harvester/harvester/pkg/util"
)

// backingImageHandler syncs upload progress from backing image to vm image status
type backingImageHandler struct {
	vmImages          harvesterv1beta1.VirtualMachineImageClient
	vmImageCache      harvesterv1beta1.VirtualMachineImageCache
	backingImages     lhv1beta1.BackingImageClient
	backingImageCache lhv1beta1.BackingImageCache
}

func (h *backingImageHandler) OnChanged(_ string, backingImage *v1beta1.BackingImage) (*v1beta1.BackingImage, error) {
	if backingImage.Spec.SourceType != types.BackingImageDataSourceTypeUpload || backingImage.Annotations[util.AnnotationImageID] == "" {
		return nil, nil
	}
	namespace, name := ref.Parse(backingImage.Annotations[util.AnnotationImageID])
	vmImage, err := h.vmImageCache.Get(namespace, name)
	if err != nil {
		return nil, err
	}
	if !v1beta12.ImageUploaded.IsUnknown(vmImage) {
		return nil, nil
	}
	toUpdate := vmImage.DeepCopy()
	for _, status := range backingImage.Status.DiskFileStatusMap {
		if status.State == types.BackingImageStateFailed {
			v1beta12.ImageUploaded.False(toUpdate)
			v1beta12.ImageUploaded.Reason(toUpdate, "UploadFailed")
			v1beta12.ImageUploaded.Message(toUpdate, status.Message)
			toUpdate.Status.Progress = status.Progress
		} else if status.State == types.BackingImageStateReady {
			v1beta12.ImageUploaded.True(toUpdate)
			v1beta12.ImageUploaded.Reason(toUpdate, "Uploaded")
			v1beta12.ImageUploaded.Message(toUpdate, status.Message)
			toUpdate.Status.Progress = status.Progress
			toUpdate.Status.Size = backingImage.Status.Size
		} else if status.Progress != toUpdate.Status.Progress {
			toUpdate.Status.Progress = status.Progress
		}
	}

	if !reflect.DeepEqual(vmImage, toUpdate) {
		if _, err := h.vmImages.Update(toUpdate); err != nil {
			return nil, err
		}
	}

	return nil, nil
}
