package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// system_profiler -json SPUSBDataType
const indent = "  "

type VolumeInfo struct {
	Name string
	DevName string
	Size int64
	FileSystem string
	UUID string
	Mounted bool
	MountPoint string // may not be mounted
	Free int64 // only availabe if mounted
	Writable bool // only availabe if mounted
}

func (v VolumeInfo) ToString(prefix string) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%sVolume %q:\n", prefix, v.Name)
	fmt.Fprintf(&buf, "%s  Device: /dev/%s\n", prefix, v.DevName)
	fmt.Fprintf(&buf, "%s  Size: %d\n", prefix, v.Size)
	fmt.Fprintf(&buf, "%s  Filesystem: %s\n", prefix, v.FileSystem)
	fmt.Fprintf(&buf, "%s  Volume UUID: %s\n", prefix, v.UUID)
	fmt.Fprintf(&buf, "%s  Mounted: %t\n", prefix, v.Mounted)
	if v.Mounted {
		fmt.Fprintf(&buf, "%s  Mount point: %s\n", prefix, v.MountPoint)
		fmt.Fprintf(&buf, "%s  Free space: %d\n", prefix, v.Free)
		fmt.Fprintf(&buf, "%s  Writable: %v\n", prefix, v.Writable)
	}
	return buf.String()
}

func (v VolumeInfo) String() string {
	return v.ToString("")
}

type MediaInfo struct {
	Name string
	DevName string
	PartitionName string
	Size int64
	Volumes []*VolumeInfo
}

func (m MediaInfo) ToString(prefix string) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%sMedia %q:\n", prefix, m.Name)
	fmt.Fprintf(&buf, "%s  Device: /dev/%s\n", prefix, m.DevName)
	fmt.Fprintf(&buf, "%s  Partition: %s\n", prefix, m.PartitionName)
	fmt.Fprintf(&buf, "%s  Size: %d\n", prefix, m.Size)
	if len(m.Volumes) == 0 {
		fmt.Fprintf(&buf, "%s  Number of Volumes: none\n",  prefix)
	} else {
		fmt.Fprintf(&buf, "%s  Number of Volumes: %d\n", prefix, len(m.Volumes))
	}
	for _, v := range m.Volumes {
		fmt.Fprintf(&buf, "%s\n", v.ToString(prefix+indent+indent))
	}
	return buf.String()
}

func (m MediaInfo) String() string {
	return m.ToString("")
}

type USBInfo struct {
	Name string
	ProductID uint16
	VendorID uint16
	SerialNumber string
	Manufacturer string
	Media []*MediaInfo
}

func (u USBInfo) ToString(prefix string) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%sUSB Storage %q:\n", prefix, u.Name)
	fmt.Fprintf(&buf, "%s  Product ID: %#04x\n", prefix, u.ProductID)
	fmt.Fprintf(&buf, "%s  Vendor ID: %#04x\n", prefix, u.VendorID)
	fmt.Fprintf(&buf, "%s  Serial Number: %s\n", prefix, u.SerialNumber)
	fmt.Fprintf(&buf, "%s  Manufacturer: %s\n", prefix, u.Manufacturer)
	if len(u.Media) == 0 {
		fmt.Fprintf(&buf, "%s  Number of Media: none\n", prefix)
	} else {
		fmt.Fprintf(&buf, "%s  Number of Media: %d\n", prefix, len(u.Media))
	}
	for _, m := range u.Media {
		fmt.Fprintf(&buf, "%s\n", m.ToString(prefix+indent+indent))
	}
	return buf.String()
}

func (u USBInfo) String(prefix string) string {
	return u.ToString("")
}

func GetVolumes(volumes any, path string) ([]*VolumeInfo, error) {
	fmt.Printf("-> Find Volumes in %s...\n", path)
	vols, ok := volumes.([]any)
	if !ok {
		return nil, fmt.Errorf("%s (%T) is not []interface{} type", path, volumes)
	}
	vis := make([]*VolumeInfo, 0)
	for i, vol := range vols {
		vMap, ok := vol.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s[%d] (%T) is not map[string]interface{} type", path, i, vol)
		}
		vi := &VolumeInfo{
			Name: vMap["_name"].(string),
			DevName: vMap["bsd_name"].(string),
			Size: int64(vMap["size_in_bytes"].(float64)),
			FileSystem: vMap["file_system"].(string),
			UUID: vMap["volume_uuid"].(string),
		}
		if m, ok := vMap["mount_point"]; ok {
			vi.Mounted = true
			vi.MountPoint = m.(string)
			vi.Free = int64(vMap["free_space_in_bytes"].(float64))
			vi.Writable = vMap["writable"].(string) == "yes"
		}
		vis = append(vis, vi)
	}
	return vis, nil
}

func GetMedia(media any, path string) ([]*MediaInfo, error) {
	fmt.Printf("-> Find Media in %s...\n", path)
	ma, ok := media.([]any)
	if !ok {
		return nil, fmt.Errorf("%s (%T) is not []interface{} type", path, media)
	}
	mis := make([]*MediaInfo, 0)
	for i, m := range ma {
		mi, ok := m.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s[%d] (%T) is not map[string]interface{} type", path, i, m)
		}
		mii := &MediaInfo{
			Name: mi["_name"].(string),
			DevName: mi["bsd_name"].(string),
			PartitionName: mi["partition_map_type"].(string),
			Size: int64(mi["size_in_bytes"].(float64)),
		}
		it, ok := mi["volumes"]
		if !ok {
			mis = append(mis, mii)
			continue
		}
		vols, err := GetVolumes(it, fmt.Sprintf("%s[%d][volumes]", path, i))
		if err != nil {
			return nil, fmt.Errorf("failed to get %s[%d][volumes]: %w", path, i, err)
		}
		mii.Volumes = vols
		mis = append(mis, mii)
	}
	return mis, nil
}

func FindInItems(items any, path string) ([]*USBInfo, error) {
	fmt.Printf("-> Find Items in %s...\n", path)
	ita, ok := items.([]any)
	if !ok {
		return nil, fmt.Errorf("%s (%T) is not []interface{} type", path, items)
	}
	uis := make([]*USBInfo, 0)
	for i, item := range ita {
		itemMap, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s[%d] (%T) is not map[string]interface{} type", path, i, item)
		}
		it, ok := itemMap["_items"]
		if ok {
			ui, err := FindInItems(it, fmt.Sprintf("%s[%d][_items]", path, i))
			if err != nil {
				return nil, fmt.Errorf(
					"failed to parse %s[%d][_items]: %w", path, i, err)
			}
			uis = append(uis, ui...)
		}
		it, ok = itemMap["Media"]
		if !ok {
			continue
		}

		usbInfo := &USBInfo{
			Name: itemMap["_name"].(string),
			SerialNumber: itemMap["serial_num"].(string),
			Manufacturer: itemMap["manufacturer"].(string),
		}
		s := itemMap["product_id"].(string)
		val, err := strconv.ParseUint(s, 0, 16)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s[%d][product_id] (%s): %w", path, i, s, err)
		}
		usbInfo.ProductID = uint16(val)
		s = itemMap["vendor_id"].(string)
		fields := strings.SplitN(s, " ", 2)
		val, err = strconv.ParseUint(fields[0], 0, 16)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s[%d][vendor_id] (%s): %w", path, i, fields[0], err)
		}
		usbInfo.VendorID = uint16(val)
		
		mi, err := GetMedia(it, fmt.Sprintf("%s[%d][Media]", path, i))
		if err != nil {
			return nil, fmt.Errorf("failed to get %s[%d][Media]: %w", path, i, err)
		}
		if len(mi) > 0 {
			usbInfo.Media = mi
		}
		uis = append(uis, usbInfo)
	}
	return uis, nil
}

func FindUSBStickInfo(data any) ([]*USBInfo, error) {
	fmt.Printf("Find USB stick info...\n")
	d, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("data (%T) is not map[string]interface{} type", data)
	}
	dt, ok := d["SPUSBDataType"]
	if !ok {
		return nil, fmt.Errorf("data missing SPUSBDataType entry: %+v", data)
	}
	dta, ok := dt.([]any)
	if !ok {
		return nil, fmt.Errorf("data[SPUSBDataType] (%T) is not []interface{} type", dt)
	}
	uis := make([]*USBInfo, 0)
	for i, dti := range dta {
		dtiMap, ok := dti.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("data[SPUSBDataType][%d] (%T) is not map[string]interface{} type", i, dti)
		}
		dt, ok := dtiMap["_items"]
		if ok {
			ui, err := FindInItems(dt, fmt.Sprintf("data[SPUSBDataType][%d][_items]", i))
			if err != nil {
				return nil, fmt.Errorf("failed to parse data[SPUSBDataType][%d][_items]: %w", i, err)
			}
			uis = append(uis, ui...)
		}
	}
	fmt.Printf("Done finding USB stick info.\n")
	return uis, nil
}

func main() {
	data := []string{noPartition, GPTPartitioned, MBRPartitioned}

	for i, d := range data {
		var jd any
		if err := json.Unmarshal([]byte(d), &jd); err != nil {
			fmt.Printf("ERROR: Failed to unmarshal JSON data[%d]: %+v\n", i, err)
			return
		}
		uis, err := FindUSBStickInfo(jd)
		if err != nil {
			fmt.Printf(
				"ERROR: Failed to find USB stick info[%d]: %+v\n", i, err)
		}
		for i, ui := range uis {
			fmt.Printf("USB Storages[%d/%d]:\n", i+1, len(uis))
			fmt.Printf("%s\n", ui.ToString(indent))
		}
	}
}

const (
	// Un-partitoned USB stick:
	noPartition = `{
		"SPUSBDataType" : [
			{
				"_name" : "USB31Bus",
				"host_controller" : "AppleT6000USBXHCI"
			},
			{
				"_items" : [
					{
						"_name" : "YubiKey OTP+FIDO+CCID",
						"bcd_device" : "5.43",
						"bus_power" : "500",
						"bus_power_used" : "30",
						"device_speed" : "full_speed",
						"extra_current_used" : "0",
						"location_id" : "0x00100000 / 1",
						"manufacturer" : "Yubico",
						"product_id" : "0x0407",
						"vendor_id" : "0x1050"
					}
				],
				"_name" : "USB31Bus",
				"host_controller" : "AppleT6000USBXHCI"
			},
			{
				"_name" : "USB31Bus",
				"host_controller" : "AppleT6000USBXHCI"
			},
			{
				"_items" : [
					{
						"_items" : [
							{
								"_items" : [
									{
										"_name" : "LG UltraFine Display Camera",
										"bcd_device" : "1.13",
										"bus_power" : "900",
										"bus_power_used" : "96",
										"device_speed" : "super_speed",
										"extra_current_used" : "0",
										"location_id" : "0x03543000 / 8",
										"manufacturer" : "LG Electronlcs Inc.",
										"product_id" : "0x9a4d",
										"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
									}
								],
								"_name" : "hub_device",
								"bcd_device" : "1.00",
								"bus_power" : "900",
								"bus_power_used" : "0",
								"device_speed" : "super_speed",
								"extra_current_used" : "0",
								"location_id" : "0x03540000 / 3",
								"product_id" : "0x9a00",
								"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
							}
						],
						"_name" : "USB3.1 Hub",
						"bcd_device" : "52.35",
						"bus_power" : "900",
						"bus_power_used" : "0",
						"device_speed" : "super_speed",
						"extra_current_used" : "0",
						"location_id" : "0x03500000 / 1",
						"manufacturer" : "LG Electronics Inc.",
						"product_id" : "0x9a44",
						"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
					},
					{
						"_items" : [
							{
								"_name" : "Magic Keyboard",
								"bcd_device" : "4.20",
								"bus_power" : "500",
								"bus_power_used" : "500",
								"device_speed" : "full_speed",
								"extra_current_used" : "1000",
								"location_id" : "0x03120000 / 5",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x029c",
								"serial_num" : "F0T2534RK0212HXAT",
								"sleep_current" : "1500",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_items" : [
									{
										"_name" : "USB Controls",
										"bcd_device" : "3.04",
										"bus_power" : "500",
										"bus_power_used" : "0",
										"device_speed" : "full_speed",
										"extra_current_used" : "0",
										"location_id" : "0x03142000 / 7",
										"manufacturer" : "LG Electronics Inc.",
										"product_id" : "0x9a40",
										"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
									},
									{
										"_name" : "USB Audio",
										"bcd_device" : "0.1e",
										"bus_power" : "500",
										"bus_power_used" : "0",
										"device_speed" : "high_speed",
										"extra_current_used" : "0",
										"location_id" : "0x03141000 / 6",
										"manufacturer" : "LG Electronics Inc.",
										"product_id" : "0x9a4b",
										"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
									}
								],
								"_name" : "hub_device",
								"bcd_device" : "1.00",
								"bus_power" : "500",
								"bus_power_used" : "0",
								"device_speed" : "high_speed",
								"extra_current_used" : "0",
								"location_id" : "0x03140000 / 4",
								"product_id" : "0x9a02",
								"serial_num" : "610C00596BFB",
								"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
							}
						],
						"_name" : "USB2.1 Hub",
						"bcd_device" : "52.35",
						"bus_power" : "500",
						"bus_power_used" : "100",
						"device_speed" : "high_speed",
						"extra_current_used" : "0",
						"location_id" : "0x03100000 / 2",
						"manufacturer" : "LG Electronics Inc.",
						"product_id" : "0x9a46",
						"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
					}
				],
				"_name" : "USB30Bus",
				"host_controller" : "AppleUSBXHCIFL1100",
				"pci_device" : "0x1100 ",
				"pci_revision" : "0x0010 ",
				"pci_vendor" : "0x1b73 "
			},
			{
				"_items" : [
					{
						"_items" : [
							{
								"_name" : "Apple Thunderbolt Display",
								"bcd_device" : "1.39",
								"Built-in_Device" : "Yes",
								"bus_power" : "500",
								"bus_power_used" : "2",
								"device_speed" : "full_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40170000 / 3",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x9227",
								"serial_num" : "182F0F36",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_name" : "FaceTime HD Camera (Display)",
								"bcd_device" : "71.60",
								"Built-in_Device" : "Yes",
								"bus_power" : "500",
								"bus_power_used" : "500",
								"device_speed" : "high_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40150000 / 2",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x1112",
								"serial_num" : "CC2D3C067PDJ9FLP",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_name" : "Display Audio",
								"bcd_device" : "2.09",
								"Built-in_Device" : "Yes",
								"bus_power" : "500",
								"bus_power_used" : "2",
								"device_speed" : "full_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40140000 / 4",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x1107",
								"serial_num" : "182F0F36",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_name" : "PenDrive",
								"bcd_device" : "0.01",
								"bus_power" : "500",
								"bus_power_used" : "200",
								"device_speed" : "high_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40120000 / 5",
								"manufacturer" : "Innostor",
								"Media" : [
									{
										"_name" : "Innostor",
										"bsd_name" : "disk5",
										"Logical Unit" : 0,
										"partition_map_type" : "unknown_partition_map_type",
										"removable_media" : "yes",
										"size" : "63.91 GB",
										"size_in_bytes" : 63909113344,
										"smart_status" : "Verified",
										"USB Interface" : 0
									}
								],
								"product_id" : "0x0917",
								"serial_num" : "000000000000005309",
								"vendor_id" : "0x1f75  (Innostor Co., Ltd.)"
							}
						],
						"_name" : "hub_device",
						"bcd_device" : "1.00",
						"Built-in_Device" : "Yes",
						"bus_power" : "500",
						"bus_power_used" : "100",
						"device_speed" : "high_speed",
						"extra_current_used" : "0",
						"location_id" : "0x40100000 / 1",
						"product_id" : "0x9127",
						"vendor_id" : "apple_vendor_id"
					}
				],
				"_name" : "USB20Bus",
				"host_controller" : "AppleUSBEHCIPI7C9X440SL",
				"pci_device" : "0x400f ",
				"pci_revision" : "0x0003 ",
				"pci_vendor" : "0x12d8 "
			}
		]
	}`

	// GPT partitoned USB stick:
	GPTPartitioned = `{
		"SPUSBDataType" : [
			{
				"_name" : "USB31Bus",
				"host_controller" : "AppleT6000USBXHCI"
			},
			{
				"_items" : [
					{
						"_name" : "YubiKey OTP+FIDO+CCID",
						"bcd_device" : "5.43",
						"bus_power" : "500",
						"bus_power_used" : "30",
						"device_speed" : "full_speed",
						"extra_current_used" : "0",
						"location_id" : "0x00100000 / 1",
						"manufacturer" : "Yubico",
						"product_id" : "0x0407",
						"vendor_id" : "0x1050"
					}
				],
				"_name" : "USB31Bus",
				"host_controller" : "AppleT6000USBXHCI"
			},
			{
				"_name" : "USB31Bus",
				"host_controller" : "AppleT6000USBXHCI"
			},
			{
				"_items" : [
					{
						"_items" : [
							{
								"_items" : [
									{
										"_name" : "LG UltraFine Display Camera",
										"bcd_device" : "1.13",
										"bus_power" : "900",
										"bus_power_used" : "96",
										"device_speed" : "super_speed",
										"extra_current_used" : "0",
										"location_id" : "0x03543000 / 8",
										"manufacturer" : "LG Electronlcs Inc.",
										"product_id" : "0x9a4d",
										"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
									}
								],
								"_name" : "hub_device",
								"bcd_device" : "1.00",
								"bus_power" : "900",
								"bus_power_used" : "0",
								"device_speed" : "super_speed",
								"extra_current_used" : "0",
								"location_id" : "0x03540000 / 3",
								"product_id" : "0x9a00",
								"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
							}
						],
						"_name" : "USB3.1 Hub",
						"bcd_device" : "52.35",
						"bus_power" : "900",
						"bus_power_used" : "0",
						"device_speed" : "super_speed",
						"extra_current_used" : "0",
						"location_id" : "0x03500000 / 1",
						"manufacturer" : "LG Electronics Inc.",
						"product_id" : "0x9a44",
						"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
					},
					{
						"_items" : [
							{
								"_name" : "Magic Keyboard",
								"bcd_device" : "4.20",
								"bus_power" : "500",
								"bus_power_used" : "500",
								"device_speed" : "full_speed",
								"extra_current_used" : "1000",
								"location_id" : "0x03120000 / 5",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x029c",
								"serial_num" : "F0T2534RK0212HXAT",
								"sleep_current" : "1500",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_items" : [
									{
										"_name" : "USB Controls",
										"bcd_device" : "3.04",
										"bus_power" : "500",
										"bus_power_used" : "0",
										"device_speed" : "full_speed",
										"extra_current_used" : "0",
										"location_id" : "0x03142000 / 7",
										"manufacturer" : "LG Electronics Inc.",
										"product_id" : "0x9a40",
										"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
									},
									{
										"_name" : "USB Audio",
										"bcd_device" : "0.1e",
										"bus_power" : "500",
										"bus_power_used" : "0",
										"device_speed" : "high_speed",
										"extra_current_used" : "0",
										"location_id" : "0x03141000 / 6",
										"manufacturer" : "LG Electronics Inc.",
										"product_id" : "0x9a4b",
										"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
									}
								],
								"_name" : "hub_device",
								"bcd_device" : "1.00",
								"bus_power" : "500",
								"bus_power_used" : "0",
								"device_speed" : "high_speed",
								"extra_current_used" : "0",
								"location_id" : "0x03140000 / 4",
								"product_id" : "0x9a02",
								"serial_num" : "610C00596BFB",
								"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
							}
						],
						"_name" : "USB2.1 Hub",
						"bcd_device" : "52.35",
						"bus_power" : "500",
						"bus_power_used" : "100",
						"device_speed" : "high_speed",
						"extra_current_used" : "0",
						"location_id" : "0x03100000 / 2",
						"manufacturer" : "LG Electronics Inc.",
						"product_id" : "0x9a46",
						"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
					}
				],
				"_name" : "USB30Bus",
				"host_controller" : "AppleUSBXHCIFL1100",
				"pci_device" : "0x1100 ",
				"pci_revision" : "0x0010 ",
				"pci_vendor" : "0x1b73 "
			},
			{
				"_items" : [
					{
						"_items" : [
							{
								"_name" : "Apple Thunderbolt Display",
								"bcd_device" : "1.39",
								"Built-in_Device" : "Yes",
								"bus_power" : "500",
								"bus_power_used" : "2",
								"device_speed" : "full_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40170000 / 3",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x9227",
								"serial_num" : "182F0F36",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_name" : "FaceTime HD Camera (Display)",
								"bcd_device" : "71.60",
								"Built-in_Device" : "Yes",
								"bus_power" : "500",
								"bus_power_used" : "500",
								"device_speed" : "high_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40150000 / 2",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x1112",
								"serial_num" : "CC2D3C067PDJ9FLP",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_name" : "Display Audio",
								"bcd_device" : "2.09",
								"Built-in_Device" : "Yes",
								"bus_power" : "500",
								"bus_power_used" : "2",
								"device_speed" : "full_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40140000 / 4",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x1107",
								"serial_num" : "182F0F36",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_name" : "PenDrive",
								"bcd_device" : "0.01",
								"bus_power" : "500",
								"bus_power_used" : "200",
								"device_speed" : "high_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40110000 / 5",
								"manufacturer" : "Innostor",
								"Media" : [
									{
										"_name" : "Innostor",
										"bsd_name" : "disk5",
										"Logical Unit" : 0,
										"partition_map_type" : "guid_partition_map_type",
										"removable_media" : "yes",
										"size" : "63.91 GB",
										"size_in_bytes" : 63909113344,
										"smart_status" : "Verified",
										"USB Interface" : 0,
										"volumes" : [
											{
												"_name" : "EFI",
												"bsd_name" : "disk5s1",
												"file_system" : "MS-DOS FAT32",
												"iocontent" : "EFI",
												"size" : "209.7 MB",
												"size_in_bytes" : 209715200,
												"volume_uuid" : "0E239BC6-F960-3107-89CF-1C97F78BB46B"
											},
											{
												"_name" : "OEL9",
												"bsd_name" : "disk5s2",
												"file_system" : "MS-DOS FAT32",
												"free_space" : "62.49 GB",
												"free_space_in_bytes" : 62491787264,
												"iocontent" : "Microsoft Basic Data",
												"mount_point" : "/Volumes/OEL9",
												"size" : "63.7 GB",
												"size_in_bytes" : 63697846272,
												"volume_uuid" : "6ABA678A-0FF6-3876-83B7-FE44B24110EB",
												"writable" : "yes"
											}
										]
									}
								],
								"product_id" : "0x0917",
								"serial_num" : "000000000000004010",
								"vendor_id" : "0x1f75  (Innostor Co., Ltd.)"
							}
						],
						"_name" : "hub_device",
						"bcd_device" : "1.00",
						"Built-in_Device" : "Yes",
						"bus_power" : "500",
						"bus_power_used" : "100",
						"device_speed" : "high_speed",
						"extra_current_used" : "0",
						"location_id" : "0x40100000 / 1",
						"product_id" : "0x9127",
						"vendor_id" : "apple_vendor_id"
					}
				],
				"_name" : "USB20Bus",
				"host_controller" : "AppleUSBEHCIPI7C9X440SL",
				"pci_device" : "0x400f ",
				"pci_revision" : "0x0003 ",
				"pci_vendor" : "0x12d8 "
			}
		]
	}`

	// MBR partitioned USB Stick:
	MBRPartitioned = `{
		"SPUSBDataType" : [
			{
				"_name" : "USB31Bus",
				"host_controller" : "AppleT6000USBXHCI"
			},
			{
				"_items" : [
					{
						"_name" : "YubiKey OTP+FIDO+CCID",
						"bcd_device" : "5.43",
						"bus_power" : "500",
						"bus_power_used" : "30",
						"device_speed" : "full_speed",
						"extra_current_used" : "0",
						"location_id" : "0x00100000 / 1",
						"manufacturer" : "Yubico",
						"product_id" : "0x0407",
						"vendor_id" : "0x1050"
					}
				],
				"_name" : "USB31Bus",
				"host_controller" : "AppleT6000USBXHCI"
			},
			{
				"_name" : "USB31Bus",
				"host_controller" : "AppleT6000USBXHCI"
			},
			{
				"_items" : [
					{
						"_items" : [
							{
								"_items" : [
									{
										"_name" : "LG UltraFine Display Camera",
										"bcd_device" : "1.13",
										"bus_power" : "900",
										"bus_power_used" : "96",
										"device_speed" : "super_speed",
										"extra_current_used" : "0",
										"location_id" : "0x03543000 / 8",
										"manufacturer" : "LG Electronlcs Inc.",
										"product_id" : "0x9a4d",
										"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
									}
								],
								"_name" : "hub_device",
								"bcd_device" : "1.00",
								"bus_power" : "900",
								"bus_power_used" : "0",
								"device_speed" : "super_speed",
								"extra_current_used" : "0",
								"location_id" : "0x03540000 / 3",
								"product_id" : "0x9a00",
								"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
							}
						],
						"_name" : "USB3.1 Hub",
						"bcd_device" : "52.35",
						"bus_power" : "900",
						"bus_power_used" : "0",
						"device_speed" : "super_speed",
						"extra_current_used" : "0",
						"location_id" : "0x03500000 / 1",
						"manufacturer" : "LG Electronics Inc.",
						"product_id" : "0x9a44",
						"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
					},
					{
						"_items" : [
							{
								"_name" : "Magic Keyboard",
								"bcd_device" : "4.20",
								"bus_power" : "500",
								"bus_power_used" : "500",
								"device_speed" : "full_speed",
								"extra_current_used" : "1000",
								"location_id" : "0x03120000 / 5",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x029c",
								"serial_num" : "F0T2534RK0212HXAT",
								"sleep_current" : "1500",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_items" : [
									{
										"_name" : "USB Controls",
										"bcd_device" : "3.04",
										"bus_power" : "500",
										"bus_power_used" : "0",
										"device_speed" : "full_speed",
										"extra_current_used" : "0",
										"location_id" : "0x03142000 / 7",
										"manufacturer" : "LG Electronics Inc.",
										"product_id" : "0x9a40",
										"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
									},
									{
										"_name" : "USB Audio",
										"bcd_device" : "0.1e",
										"bus_power" : "500",
										"bus_power_used" : "0",
										"device_speed" : "high_speed",
										"extra_current_used" : "0",
										"location_id" : "0x03141000 / 6",
										"manufacturer" : "LG Electronics Inc.",
										"product_id" : "0x9a4b",
										"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
									}
								],
								"_name" : "hub_device",
								"bcd_device" : "1.00",
								"bus_power" : "500",
								"bus_power_used" : "0",
								"device_speed" : "high_speed",
								"extra_current_used" : "0",
								"location_id" : "0x03140000 / 4",
								"product_id" : "0x9a02",
								"serial_num" : "610C00596BFB",
								"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
							}
						],
						"_name" : "USB2.1 Hub",
						"bcd_device" : "52.35",
						"bus_power" : "500",
						"bus_power_used" : "100",
						"device_speed" : "high_speed",
						"extra_current_used" : "0",
						"location_id" : "0x03100000 / 2",
						"manufacturer" : "LG Electronics Inc.",
						"product_id" : "0x9a46",
						"vendor_id" : "0x043e  (LG Electronics USA Inc.)"
					}
				],
				"_name" : "USB30Bus",
				"host_controller" : "AppleUSBXHCIFL1100",
				"pci_device" : "0x1100 ",
				"pci_revision" : "0x0010 ",
				"pci_vendor" : "0x1b73 "
			},
			{
				"_items" : [
					{
						"_items" : [
							{
								"_name" : "Apple Thunderbolt Display",
								"bcd_device" : "1.39",
								"Built-in_Device" : "Yes",
								"bus_power" : "500",
								"bus_power_used" : "2",
								"device_speed" : "full_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40170000 / 3",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x9227",
								"serial_num" : "182F0F36",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_name" : "FaceTime HD Camera (Display)",
								"bcd_device" : "71.60",
								"Built-in_Device" : "Yes",
								"bus_power" : "500",
								"bus_power_used" : "500",
								"device_speed" : "high_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40150000 / 2",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x1112",
								"serial_num" : "CC2D3C067PDJ9FLP",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_name" : "Display Audio",
								"bcd_device" : "2.09",
								"Built-in_Device" : "Yes",
								"bus_power" : "500",
								"bus_power_used" : "2",
								"device_speed" : "full_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40140000 / 4",
								"manufacturer" : "Apple Inc.",
								"product_id" : "0x1107",
								"serial_num" : "182F0F36",
								"vendor_id" : "apple_vendor_id"
							},
							{
								"_name" : "Flash Disk",
								"bcd_device" : "1.09",
								"bus_power" : "500",
								"bus_power_used" : "200",
								"device_speed" : "high_speed",
								"extra_current_used" : "0",
								"location_id" : "0x40110000 / 5",
								"manufacturer" : "USB",
								"Media" : [
									{
										"_name" : "Flash Disk",
										"bsd_name" : "disk5",
										"Logical Unit" : 0,
										"partition_map_type" : "master_boot_record_partition_map_type",
										"removable_media" : "yes",
										"size" : "1.93 GB",
										"size_in_bytes" : 1930428416,
										"smart_status" : "Verified",
										"USB Interface" : 0,
										"volumes" : [
											{
												"_name" : "TEST",
												"bsd_name" : "disk5s1",
												"file_system" : "MS-DOS FAT16",
												"free_space" : "1.93 GB",
												"free_space_in_bytes" : 1926168576,
												"iocontent" : "DOS_FAT_16",
												"mount_point" : "/Volumes/TEST",
												"size" : "1.93 GB",
												"size_in_bytes" : 1929379840,
												"volume_uuid" : "182684DE-533E-394C-A563-1D491C94108A",
												"writable" : "yes"
											}
										]
									}
								],
								"product_id" : "0x6387",
								"serial_num" : "59402D7A",
								"vendor_id" : "0x058f  (Alcor Micro, Corp.)"
							}
						],
						"_name" : "hub_device",
						"bcd_device" : "1.00",
						"Built-in_Device" : "Yes",
						"bus_power" : "500",
						"bus_power_used" : "100",
						"device_speed" : "high_speed",
						"extra_current_used" : "0",
						"location_id" : "0x40100000 / 1",
						"product_id" : "0x9127",
						"vendor_id" : "apple_vendor_id"
					}
				],
				"_name" : "USB20Bus",
				"host_controller" : "AppleUSBEHCIPI7C9X440SL",
				"pci_device" : "0x400f ",
				"pci_revision" : "0x0003 ",
				"pci_vendor" : "0x12d8 "
			}
		]
	}`
)