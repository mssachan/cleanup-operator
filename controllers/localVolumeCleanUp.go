package controllers

import (
	"context"
	"fmt"
	"strings"

	localv1 "github.com/openshift/local-storage-operator/pkg/apis/local/v1"
	corev1 "k8s.io/api/core/v1"
	apiextenstionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (cr *CleanUpOperatorReconciler) localVolumeCleanUp(ctx context.Context, namespace string) error {
	localDisk := &localv1.LocalVolume{}
	err := cr.Get(ctx, types.NamespacedName{Name: localVolumeName, Namespace: namespace}, localDisk)
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Println("LocalVolume 'local-disk' not found")
			return err
		}
		fmt.Println(err, "Error in getting LocalVolume 'local-disk'")
		return err
	}

	localDisk.SetFinalizers([]string{})
	if err := cr.Update(ctx, localDisk); err != nil {
		fmt.Println(err, "Error is removing finalizers from Local-Volume 'local-disk'")
		return err
	}

	// Find PVs
	pvList := &corev1.PersistentVolumeList{}
	err = cr.List(ctx, pvList)
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Println("PVs not found")
			return err
		}
		fmt.Println(err, "Error in getting PVs")
		return err
	}

	// PV Deletion
	for _, pv := range pvList.Items {
		if strings.HasPrefix(pv.Name, "local-pv-") {
			fmt.Println("PV status- ", pv.Status.Phase)
			err = cr.Delete(ctx, &pv)
			if err != nil {
				if errors.IsNotFound(err) {
					fmt.Println("PV not found")
					return err
				}
				fmt.Print("Error in Deleting PV ", pv.Name)
				return err
			}
		}
	}
	fmt.Println("PV Deleted.....")

	// Remove Mounted Path
	nodesList := &corev1.NodeList{}
	err = cr.List(ctx, nodesList)
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Println("Nodes List not found")
			return err
		}
		fmt.Println(err, "Error in getting Nodes List")
		return err
	}

	for _, node := range nodesList.Items {
		command := "oc debug node/" + node.Name + " -- chroot /host rm -rf /mnt"
		_, out, err := ExecuteCommand(command)
		if err != nil {
			fmt.Println("Error in removing mounted path from node: ", node.Name)
			return err
		}
		fmt.Println(out)
	}
	fmt.Println("Mounted Paths Removed....")

	return nil
}

// removeLocalVolmeCRDs patches and deletes localVolume crds
func (cr *CleanUpOperatorReconciler) removeLocalVolmeCRDs(ctx context.Context) error {
	crdNames := []string{"localvolumediscoveries.local.storage.openshift.io", "localvolumediscoveryresults.local.storage.openshift.io",
		"localvolumes.local.storage.openshift.io", "localvolumesets.local.storage.openshift.io"}
	for _, crd := range crdNames {
		CRD := &apiextenstionsv1.CustomResourceDefinition{}
		err := cr.Get(ctx, types.NamespacedName{Name: crd}, CRD)
		if err != nil {
			if errors.IsNotFound(err) {
				fmt.Println("CRD not found: ", crd)
				continue
			}
			fmt.Println(err, "error in getting crd: ", crd)
			return err
		}

		CRD.SetFinalizers([]string{})
		if err := cr.Update(ctx, CRD); err != nil {
			fmt.Println(err, "Error is removing finalizers from CustomResoure ", CRD.Name)
			return err
		}

		err = cr.Delete(ctx, CRD)
		if err != nil {
			fmt.Println(err, "Error is deleting CustomResoure ", CRD.Name)
			return err
		}

		fmt.Println(CRD.Name)
	}
	return nil
}
