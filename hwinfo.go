// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//TODO package description
package hwinfo

import (
	"fmt"
	"github.com/fedesog/hwinfo/byteunit"
	"path"
	"strings"
)

//A SystemInfo describes a computer.
type SystemInfo struct {
	Vendor   string
	Model    string
	Version  string
	Serial   string
	IsLaptop bool
}

//ModelVersion returns the Model concatenated with the Version if defined.
func (s SystemInfo) ModelVersion() string {
	fullname := s.Model
	if s.Version != "" &&
			s.Version != "Not Specified" {
		fullname += " - " + s.Version
	}
	return fullname
}

//ListSystemInfo returns a SystemInfo describing the current computer.
//It requires the following command line tools: dmidecode (root), laptop-detect.
func ListSystemInfo() (*SystemInfo, error) {
	es := run("dmidecode", "-t", "1")
	if es.Err != nil {
		return nil, newCmdError(es)
	}
	es = run("dmidecode", "-t", "1")
	if es.Err != nil {
		return nil, newCmdError(es)
	}
	system := &SystemInfo{}
	for _, l := range es.lines() {
		key, val := getKeyValue(l, ":")
		switch key {
		case "Manufacturer":
			system.Vendor = val
		case "Product Name":
			system.Model = val
		case "Version":
			system.Version = val
		case "Serial Number":
			system.Serial = val
		}
	}
	es = run("laptop-detect")
	switch es.code() {
	case 0:
		system.IsLaptop = true
	case 1:
		system.IsLaptop = false
	default:
		return nil, newCmdError(es)
	}
	return system, nil
}

//A CPUInfo describes an installed CPU.
type CPUInfo struct {
	Id int
	Model string
	FreqGHz float64
	PhysicalId int
	PhysicalCores int
	LogicalCores int
}

//ListCPUInfo returns a slice of CPUInfo that describes the installed CPUs.
//Note: if "/sys/devices/system/cpu/cpu%d/cpufreq/cpuinfo_max_freq" doesn't
//exists it returns the frequency listed in /proc/cpuinfo on the basis that
//if the BIOS is blocking CPU scaling the CPU will run at full speed.
//TODO check if this is not false on as many systems as possible
func ListCPUInfo() ([]CPUInfo, error) {
	es := read("/proc/cpuinfo")
	if es.Err != nil {
		return nil, newCmdError(es)
	}
	cpus := make([]CPUInfo, 0)
	var cpu CPUInfo
	currentProcessor := 0
	for _, l := range es.lines() {
		key, val := getKeyValue(l, ":")
		switch key {
		case "processor":
			n, err := strToInt(val)
			if err != nil {
				return nil, fmt.Errorf("processor number: %v\n%s", err, l)
			}
			if n != currentProcessor {
				cpus = append(cpus, cpu)
				cpu = CPUInfo{}
				currentProcessor = n
			}
		case "cpu MHz":
			format := "/sys/devices/system/cpu/cpu%d/cpufreq/cpuinfo_max_freq"
			es1 := read(fmt.Sprintf(format, cpu.Id))
			if es1.Err != nil {
				//(TODO fedesog) is cpuinfo frequency valid if /sys/.../cpuinfo_max_freq
				// doesn't exist?
				freqMHz, err := strToFloat(val)
				if err != nil {
					return nil, fmt.Errorf("error reading processor frequency: %v\n%s", err, l)
				} else {
					cpu.FreqGHz = freqMHz/1000
				}
			} else {
				freqkHz, err := strToInt(string(es1.Stdout))
				if err != nil {
					return nil, fmt.Errorf("error reading processor frequency (sysfs): %v\n%s", err, l)
				} else {
					cpu.FreqGHz = float64(freqkHz)/1000000
				}
			}
		case "model name":
			cpu.Model = val
		case "physical id":
			id, err := strToInt(val)
			if err != nil {
				return nil, fmt.Errorf("error reading processor id: %v\n%s", err, l)
			} else {
			 cpu.PhysicalId = id
			}
		case "cpu cores":
			cores, err := strToInt(val)
			if err != nil {
				return nil, fmt.Errorf("error reading processor cores: %v\n%s", err, l)
			} else {
				cpu.PhysicalCores = cores
			}
		case "siblings":
			siblings, err := strToInt(val)
			if err != nil {
				return nil, fmt.Errorf("error reading processor siblings: %v\n%s", err, l)
			} else {
				cpu.LogicalCores = siblings
			}
		}
	}
	cpus = append(cpus, cpu)
	//CPUs with the same PhysicalID, are the same CPU, merge them
	phycpus := make([]CPUInfo, 0)
	lastPhyId := -1
	for i := 0; i < len(cpus); i++ {
		if cpus[i].PhysicalId != lastPhyId {
			lastPhyId = cpus[i].PhysicalId
			phycpus = append(phycpus, cpus[i])
		}
	}
	return phycpus, nil
}

//RAMInfo describes installed RAM
type RAMInfo struct {
	MaxSize byteunit.Size
	Modules []RAMModule
}

//InstalledSize returns the totat amount of RAM installed
func (m RAMInfo) InstalledSize() byteunit.Size {
	var sum byteunit.Size
	for _, module := range m.Modules {
		sum += module.Size
	}
	return sum
}

//RAMModule describes a RAM module
type RAMModule struct {
	Size    byteunit.Size
	Slot    string
	Class   string //DDR, DDR2, DDR3...
	Type    string //DIMM...
	FreqMHz int
}

//ReadRAMInfo returns a RAMInfo describing the installed RAM.
func ReadRAMInfo() (*RAMInfo, error) {
	es := run("dmidecode", "-t", "16,17")
	if es.Err != nil {
		return nil, newCmdError(es)
	}
	ram := &RAMInfo{}
	add := func(m *RAMModule) {
		if m == nil {
			return
		}
		//workarounds
		//exclude SYSTEM ROM from memory devices
		if m.Slot == "SYSTEM ROM" || m.Type == "Flash" {
			return
		}
		ram.Modules = append(ram.Modules, *m)
	}
	for _, l := range es.lines() {
		key, val := getKeyValue(l, ":")
		switch key {
		case "Maximum Capacity":
			size, err := byteunit.Parse(val)
			if err != nil {
				return nil, err
			}
			ram.MaxSize = size
		}
	}
	var module *RAMModule
	for _, l := range es.lines() {
		if l == "Memory Device" {
			add(module)
			module = &RAMModule{}
		}
		key, val := getKeyValue(l, ":")
		switch key {
		case "Size":
			if val == "No Module Installed" {
				module.Size = 0
			} else {
				size, err := byteunit.Parse(val)
				if err != nil {
					return nil, err
				}
				module.Size = size
			}
		case "Type":
			module.Class = val
		case "Form Factor":
			module.Type = val
		case "Speed":
			if module.Size != 0 {
				fmt.Sscan(val, &module.FreqMHz)
			}
		case "Locator":
			module.Slot = val
		}
	}
	add(module)
	return ram, nil
}

//DriveInfo describes an installed hard drive.
type DriveInfo struct {
	Device string
	Model string
	Serial string
	Size byteunit.Size
	SmartEnabled bool
	SmartPassed bool
	Type string
	NoPartitions bool
}

//ListDriveInfo returns a slice of DriveInfo
func ListDriveInfo() ([]*DriveInfo, error) {
	es := run("lsscsi", "-t")
	if es.Err != nil {
		return nil, newCmdError(es)
	}
	var drives []*DriveInfo
	for _, l := range es.lines() {
		if !strings.Contains(l, "disk") {
			continue
		}
		if strings.Contains(l, "usb:") {
			continue
		}
		dev := strings.TrimSpace(l[strings.Index(l, "/dev"):])
		drive := &DriveInfo{Device: dev}
		es := run("smartctl", "-i", dev)
		if es.Err != nil {
			return nil, newCmdError(es)
		}
		for _, l := range es.lines() {
			key, val := getKeyValue(l, ":")
			switch key {
			case "User Capacity":
				if i := strings.Index(l, "["); i != -1 {
					l = l[i+1:len(l)-1]
					size, err := byteunit.Parse(l)
					if err != nil {
						return nil, fmt.Errorf("error reading drive capacity: %v", err)
					}
					drive.Size = size
				} else {
					return nil, fmt.Errorf("error reading drive capacity: %s", l)
				}
			case "Device Model":
				drive.Model = val
			case "Serial Number":
				drive.Serial = val
			case "SMART support is":
				if val == "Enabled" {
					drive.SmartEnabled = true
					es := run("smartctl", "-H", dev)
					if (es.code() & 0x01) != 0 {
						//Bit 0: command line did not parse
						return nil, newCmdError(es)
					}
					//Checks only actual drive failing (bit 3 set)
					if (es.code() & 0x08) != 0 {
						drive.SmartPassed = false
					} else {
						drive.SmartPassed = true
					}
				} else if strings.Contains(val, "Unavailable") {
					drive.SmartEnabled = false
				}
			}
		}
		sysDriveRotational := path.Join("/sys/block", path.Base(dev), "queue/rotational")
		es = read(sysDriveRotational)
		if es.Err != nil {
			return nil, newCmdError(es)
		}
		ls := es.lines()
		if len(ls) == 1 && ls[0] == "0" {
			drive.Type = "SSD"
		} else if len(ls) == 1 && ls[0] == "1" {
			drive.Type = "HDD"
		} else {
			return nil, fmt.Errorf("error reading drive rotational value: %s", ls)
		}
		//use lsblk to check if kernel sees partitions
		es = run("lsblk", "-lnr", dev)
		if es.Err != nil {
			return nil, newCmdError(es)
		}
		switch n := len(es.lines()); {
		case n == 1:
			drive.NoPartitions = true
		case n > 1:
			drive.NoPartitions = false
		default:
			panic("unreachable")
		}
		drives = append(drives, drive)
	}
	return drives, nil
}

//InterfaceInfo describes an installed network interface.
type InterfaceInfo struct {
	Device string
	Type string
	State string
}

//IsWireless returns true if interface is wifi, false otherwise.
func (i InterfaceInfo) IsWireless() bool {
	return strings.Contains(i.Type, "wireless")
}

//IsEthernet returns true if interface is ethernet, false otherwise.
func (i InterfaceInfo) IsEthernet() bool {
	return strings.Contains(i.Type, "ethernet")
}

//CanScan returns true if interface can execute a scan and find at least one
//access point. If interface is not Wifi it always returns false.
func (i InterfaceInfo) CanScan() (bool, error) {
	if !i.IsWireless() {
		return false, nil
	}
	es := run("iwlist", i.Device, "scan")
	if es.Err != nil {
		return false, newCmdError(es)
	}
	cellCount := 0
	for _, line := range es.lines() {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Cell") {
			cellCount++
		}
	}
	if cellCount > 0 {
		return true, nil
	} else {
		return false, nil
	}		
}

//Update update the status (connected, unavailable, disconnected, connecting...)
//of all interfaces
func (i *InterfaceInfo) Update() error {
	ifaces, err := ListInterfaceInfo()
	if err != nil {
		return err
	}
	for _, iface := range ifaces {
		if iface.Device == i.Device {
			i.State = iface.State
		}
	}
	return nil
}

//IsHardBlocked returns true if interface is off because of a hardware switch.
//It returns false otherwise.
//Note: that interface could still be off because of a software switch
// ("rfkill unblock" to unblock a software switch)
func (i InterfaceInfo) IsHardBlocked() (bool, error) {
	//TODO should check the correct interface, I can extrapolate it from 'iw dev'
	es := run("rfkill", "list", "wifi")
	if es.Err != nil {
		return false, newCmdError(es)
	}
	for _, l := range es.lines() {
		if strings.Contains(l, "Hard Blocked: yes") {
			return true, nil
		}
	}
	return false, nil
}

//ListInterfaceInfo returns a slice of InterfaceInfo
func ListInterfaceInfo() ([]*InterfaceInfo, error) {
	es := run("rfkill", "unblock", "all")
	if es.Err != nil {
		return nil, newCmdError(es)
	}
	es = run("nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev")
	if es.Err != nil {
		return nil, newCmdError(es)
	}
	var ifaces []*InterfaceInfo
	for _, l := range es.lines() {
		v := strings.Split(l, ":")
		if len(v) != 3 {
			return nil, fmt.Errorf("error reading iface: unexpected nmcli output: %s ", l)
		}
		iface := &InterfaceInfo{Device: v[0], Type: v[1], State: v[2]}
		ifaces = append(ifaces, iface)
	}
	return ifaces, nil
}

