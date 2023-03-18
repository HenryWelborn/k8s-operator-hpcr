// Copyright 2023 IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.package datasource

package onprem

import (
	"log"

	libvirt "github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

// RemoveDataDisk removes the data disk
func RemoveDataDisk(client *LivirtClient) func(key string) error {
	return deleteVolumeByKey(client)
}

// CreateDataDisk creates a data disk or resizes an existing one if required
func CreateDataDisk(client *LivirtClient) func(storagePool, name string, size uint64) (*libvirt.StorageVol, error) {
	conn := client.LibVirt

	storageVolXMLDesc := getStorageVolXMLDesc(conn)

	return func(storagePool, name string, size uint64) (*libvirt.StorageVol, error) {
		// check if we already know the disk
		pool, err := conn.StoragePoolLookupByName(storagePool)
		if err != nil {
			return nil, err
		}
		// check if the volume exists
		existing, err := conn.StorageVolLookupByName(pool, name)
		if err == nil {
			// check some metadata
			existingXML, err := storageVolXMLDesc(&existing)
			if err != nil {
				return nil, err
			}
			// check if the capacity matches
			if existingXML.Capacity.Value < size {
				log.Printf("Resizing storage volume [%s] on pool [%s] from [%d] to [%d] ...", existingXML.Name, pool.Name, existingXML.Capacity.Value, size)
				// resize
				err := conn.StorageVolResize(existing, size, 0)
				if err != nil {
					return nil, err
				}
				log.Printf("Successfully resized volume [%s] on pool [%s]", existingXML.Name, pool.Name)
				return &existing, nil
			}
		}
		// need to create a new volume
		volumeDef := createDefaultVolume()
		volumeDef.Name = name
		volumeDef.Capacity.Value = size

		volumeDefXML, err := XMLMarshall(volumeDef)
		if err != nil {
			return nil, err
		}

		// create the volume
		log.Printf("Creating new volume [%s] on pool [%s] with size [%d] ...", volumeDef.Name, pool.Name, size)
		volume, err := conn.StorageVolCreateXML(pool, string(volumeDefXML), 0)
		if err != nil {
			return nil, err
		}

		log.Printf("Successfully created volume [%s] on pool [%s]", volumeDef.Name, pool.Name)

		return &volume, nil
	}
}

// CreateDataDiskXML creates the XML for the data disk
// func CreateDataDiskXML(client *LivirtClient) func(name string, size uint64) (*libvirtxml.DomainDisk, error) {
// 	conn := client.LibVirt

// 	return func(name string, size uint64) (*libvirtxml.DomainDisk, error) {

// 		return &libvirtxml.DomainDisk{
// 			Device: "disk",
// 			Target: &libvirtxml.DomainDiskTarget{
// 				Dev: "vda",
// 				Bus: "virtio",
// 			},
// 			Driver: &libvirtxml.DomainDiskDriver{
// 				Name:  "qemu",
// 				Type:  "qcow2",
// 				IOMMU: "on",
// 			},
// 			Source: &libvirtxml.DomainDiskSource{
// 				File: &libvirtxml.DomainDiskSourceFile{
// 					File: diskVolumeFile,
// 				},
// 			},
// 			Boot: &libvirtxml.DomainDeviceBoot{
// 				Order: 1,
// 			},
// 		}, nil
// 	}
// }

// DeleteDataDiskSync (synchronously) deletes a data disk
func DeleteDataDiskSync(client *LivirtClient) func(storagePool, name string) error {
	conn := client.LibVirt
	removeDataDisk := RemoveDataDisk(client)

	return func(storagePool, name string) error {
		// check if we already know the disk
		pool, err := conn.StoragePoolLookupByName(storagePool)
		if err != nil {
			return err
		}
		// check if the volume exists
		existing, err := conn.StorageVolLookupByName(pool, name)
		if err != nil {
			// nothing to delete
			log.Printf("Volume [%s] does not exist on pool [%s], nothing to do", name, pool.Name)
			return nil
		}
		// delete
		return removeDataDisk(existing.Key)
	}
}

// IsDataDiskValid tests if a data disk has a valid configuration
func IsDataDiskValid(client *LivirtClient) func(opt *DataDiskOptions) (*libvirtxml.StorageVolume, bool) {
	// connection
	conn := client.LibVirt
	storageVolXMLDesc := getStorageVolXMLDesc(conn)

	return func(opt *DataDiskOptions) (*libvirtxml.StorageVolume, bool) {
		// check for the pool
		pool, err := conn.StoragePoolLookupByName(opt.StoragePool)
		if err != nil {
			log.Printf("Unable to lookup storage pool [%s], cause: [%v]", opt.StoragePool, err)
			return nil, false
		}
		// lookup the volume
		vol, err := conn.StorageVolLookupByName(pool, opt.Name)
		if err != nil {
			log.Printf("Unable to lookup volume [%s] on pool [%s], cause: [%v]", opt.Name, pool.Name, err)
			return nil, false
		}
		// get some metadata
		volXML, err := storageVolXMLDesc(&vol)
		if err != nil {
			log.Printf("Unable to get information for volume [%s] on pool [%s], cause: [%v]", opt.Name, pool.Name, err)
			return nil, false
		}
		// check the capacity
		if volXML.Capacity.Value < opt.Size {
			log.Printf("Size of the existing volume [%s] is [%d] and is less than the requested size [%d]", volXML.Name, volXML.Capacity.Value, opt.Size)
			return volXML, false
		}
		// nothing to do
		log.Printf("Volume [%s] is already up to date.", volXML.Name)
		return volXML, true
	}
}

// CreateDataDiskSync creates a data disk or resizes an existing one if required
func CreateDataDiskSync(client *LivirtClient) func(opt *DataDiskOptions) (*libvirt.StorageVol, error) {
	createDataDisk := CreateDataDisk(client)
	return func(opt *DataDiskOptions) (*libvirt.StorageVol, error) {
		return createDataDisk(opt.StoragePool, opt.Name, opt.Size)
	}
}
